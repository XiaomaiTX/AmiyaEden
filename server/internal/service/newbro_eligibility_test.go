package service

import (
	"amiya-eden/internal/model"
	"testing"
	"time"
)

func TestBuildNewbroRuleVersion(t *testing.T) {
	got := BuildNewbroRuleVersion(NewbroEligibilityRules{
		MaxCharacterSP:          20_000_000,
		MultiCharacterSP:        10_000_000,
		MultiCharacterThreshold: 3,
		AttributionLookbackDays: 30,
	})

	const want = "sp:20000000;multi-sp:10000000;multi-count:3;lookback-days:30"
	if got != want {
		t.Fatalf("BuildNewbroRuleVersion() = %q, want %q", got, want)
	}
}

func TestEvaluateNewbroEligibility(t *testing.T) {
	rules := NewbroEligibilityRules{
		MaxCharacterSP:          20_000_000,
		MultiCharacterSP:        10_000_000,
		MultiCharacterThreshold: 3,
	}

	t.Run("eligible when all characters stay below thresholds", func(t *testing.T) {
		result := EvaluateNewbroEligibility([]NewbroCharacterSnapshot{
			{CharacterID: 9001, CorporationID: 1001, TotalSP: 19_999_999},
			{CharacterID: 9002, CorporationID: 2001, TotalSP: 9_999_999},
			{CharacterID: 9003, CorporationID: 1002, TotalSP: 10_000_000},
		}, rules)

		if !result.IsCurrentlyNewbro {
			t.Fatalf("expected user to remain newbro, got %+v", result)
		}
		if result.DisqualifiedReason != "" {
			t.Fatalf("expected empty reason for eligible user, got %q", result.DisqualifiedReason)
		}
	})

	t.Run("disqualifies when any character reaches skill threshold", func(t *testing.T) {
		result := EvaluateNewbroEligibility([]NewbroCharacterSnapshot{
			{CharacterID: 9001, CorporationID: 1001, TotalSP: 20_000_000},
			{CharacterID: 9002, CorporationID: 1002, TotalSP: 100_000},
		}, rules)

		if result.IsCurrentlyNewbro {
			t.Fatalf("expected user to be disqualified, got %+v", result)
		}
		if result.DisqualifiedReason != NewbroDisqualifiedReasonSkillPointThresholdReached {
			t.Fatalf("unexpected reason: %q", result.DisqualifiedReason)
		}
	})

	t.Run("disqualifies when three characters reach multi-character threshold", func(t *testing.T) {
		result := EvaluateNewbroEligibility([]NewbroCharacterSnapshot{
			{CharacterID: 1, CorporationID: 1001, TotalSP: 10_000_000},
			{CharacterID: 2, CorporationID: 2001, TotalSP: 12_000_000},
			{CharacterID: 3, CorporationID: 3001, TotalSP: 15_000_000},
			{CharacterID: 4, CorporationID: 4001, TotalSP: 9_000_000},
		}, rules)

		if result.IsCurrentlyNewbro {
			t.Fatalf("expected user to be disqualified, got %+v", result)
		}
		if result.DisqualifiedReason != NewbroDisqualifiedReasonMultiCharacterSkillPointThresholdReached {
			t.Fatalf("unexpected reason: %q", result.DisqualifiedReason)
		}
	})

	t.Run("prefers skill threshold reason when both conditions are met", func(t *testing.T) {
		result := EvaluateNewbroEligibility([]NewbroCharacterSnapshot{
			{CharacterID: 1, CorporationID: 1001, TotalSP: 20_000_001},
			{CharacterID: 2, CorporationID: 1001, TotalSP: 10_000_000},
			{CharacterID: 3, CorporationID: 1002, TotalSP: 11_000_000},
			{CharacterID: 4, CorporationID: 1002, TotalSP: 12_000_000},
		}, rules)

		if result.DisqualifiedReason != NewbroDisqualifiedReasonSkillPointThresholdReached {
			t.Fatalf("expected skill threshold to win tie, got %q", result.DisqualifiedReason)
		}
	})
}

func TestNeedsNewbroEligibilityRefresh(t *testing.T) {
	const version = "sp:20000000;multi-sp:10000000;multi-count:3;lookback-days:30"
	now := time.Date(2026, time.March, 27, 12, 0, 0, 0, time.UTC)
	refreshInterval := 7 * 24 * time.Hour

	t.Run("missing state requires refresh", func(t *testing.T) {
		if !NeedsNewbroEligibilityRefresh(nil, version, now, refreshInterval) {
			t.Fatal("expected nil state to require refresh")
		}
	})

	t.Run("mismatched version requires refresh", func(t *testing.T) {
		state := &model.NewbroPlayerState{RuleVersion: "old"}
		if !NeedsNewbroEligibilityRefresh(state, version, now, refreshInterval) {
			t.Fatal("expected stale rule version to require refresh")
		}
	})

	t.Run("matching version keeps non-newbro sticky", func(t *testing.T) {
		state := &model.NewbroPlayerState{
			RuleVersion:       version,
			IsCurrentlyNewbro: false,
			EvaluatedAt:       now.Add(-30 * 24 * time.Hour),
		}
		if NeedsNewbroEligibilityRefresh(state, version, now, refreshInterval) {
			t.Fatal("did not expect non-newbro state to refresh without rule change")
		}
	})

	t.Run("matching version keeps recent newbro state", func(t *testing.T) {
		state := &model.NewbroPlayerState{
			RuleVersion:       version,
			IsCurrentlyNewbro: true,
			EvaluatedAt:       now.Add(-6 * 24 * time.Hour),
		}
		if NeedsNewbroEligibilityRefresh(state, version, now, refreshInterval) {
			t.Fatal("did not expect recent newbro state to refresh")
		}
	})

	t.Run("matching version refreshes stale newbro state", func(t *testing.T) {
		state := &model.NewbroPlayerState{
			RuleVersion:       version,
			IsCurrentlyNewbro: true,
			EvaluatedAt:       now.Add(-8 * 24 * time.Hour),
		}
		if !NeedsNewbroEligibilityRefresh(state, version, now, refreshInterval) {
			t.Fatal("expected stale newbro state to refresh")
		}
	})
}

func TestSyncAffiliationWithEligibility(t *testing.T) {
	evaluatedAt := time.Date(2026, time.March, 28, 9, 30, 0, 0, time.UTC)

	t.Run("keeps affiliation when user is still a newbro", func(t *testing.T) {
		called := false
		svc := &NewbroEligibilityService{
			endAffiliationByUserID: func(userID uint, endedAt time.Time) error {
				called = true
				return nil
			},
		}

		if err := svc.syncAffiliationWithEligibility(42, true, evaluatedAt); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if called {
			t.Fatal("did not expect affiliation cleanup for eligible user")
		}
	})

	t.Run("ends affiliation when recalculation makes user ineligible", func(t *testing.T) {
		var gotUserID uint
		var gotEndedAt time.Time
		svc := &NewbroEligibilityService{
			endAffiliationByUserID: func(userID uint, endedAt time.Time) error {
				gotUserID = userID
				gotEndedAt = endedAt
				return nil
			},
		}

		if err := svc.syncAffiliationWithEligibility(42, false, evaluatedAt); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if gotUserID != 42 {
			t.Fatalf("expected cleanup for user 42, got %d", gotUserID)
		}
		if !gotEndedAt.Equal(evaluatedAt) {
			t.Fatalf("expected ended_at %v, got %v", evaluatedAt, gotEndedAt)
		}
	})
}
