package service

import (
	"amiya-eden/internal/model"
	"testing"
	"time"
)

func TestNormalizeRecentAffiliations(t *testing.T) {
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	rows := make([]model.NewbroCaptainAffiliation, 0, 12)
	for i := 0; i < 12; i++ {
		rows = append(rows, model.NewbroCaptainAffiliation{
			BaseModel: model.BaseModel{ID: uint(100 + i)},
			StartedAt: now.Add(-time.Duration(i) * time.Hour),
		})
	}

	got := normalizeRecentAffiliations(rows)
	if len(got) != 10 {
		t.Fatalf("expected recent affiliations to be capped at 10, got %d", len(got))
	}
	if got[0].ID != 100 {
		t.Fatalf("expected first item to stay most recent, got %d", got[0].ID)
	}
	if got[9].ID != 109 {
		t.Fatalf("expected tenth item to be retained, got %d", got[9].ID)
	}
}

func TestShouldReuseCurrentAffiliation(t *testing.T) {
	current := &model.NewbroCaptainAffiliation{
		PlayerUserID:  9,
		CaptainUserID: 42,
		EndedAt:       nil,
	}

	if !shouldReuseCurrentAffiliation(current, 42) {
		t.Fatal("expected matching active captain to reuse current affiliation")
	}
	if shouldReuseCurrentAffiliation(current, 7) {
		t.Fatal("did not expect different captain to reuse current affiliation")
	}

	endedAt := time.Now()
	current.EndedAt = &endedAt
	if shouldReuseCurrentAffiliation(current, 42) {
		t.Fatal("did not expect ended affiliation to be reused")
	}
}

func TestBuildNewbroCaptainAffiliationTracksActor(t *testing.T) {
	now := time.Date(2026, 3, 27, 14, 30, 0, 0, time.UTC)

	row := buildNewbroCaptainAffiliation(1001, 90000001, 2002, 3003, now)

	if row.PlayerUserID != 1001 {
		t.Fatalf("expected player user ID 1001, got %d", row.PlayerUserID)
	}
	if row.PlayerPrimaryCharacterIDAtStart != 90000001 {
		t.Fatalf("expected player primary character ID to be captured, got %d", row.PlayerPrimaryCharacterIDAtStart)
	}
	if row.CaptainUserID != 2002 {
		t.Fatalf("expected captain user ID 2002, got %d", row.CaptainUserID)
	}
	if row.CreatedBy != 3003 {
		t.Fatalf("expected created_by to capture actor 3003, got %d", row.CreatedBy)
	}
	if !row.StartedAt.Equal(now) {
		t.Fatalf("expected started_at %v, got %v", now, row.StartedAt)
	}
}

func TestShouldBlockSelfAffiliation(t *testing.T) {
	if !shouldBlockSelfAffiliation(42, 42) {
		t.Fatal("expected self-affiliation to be blocked")
	}
	if shouldBlockSelfAffiliation(42, 84) {
		t.Fatal("did not expect different captain to be blocked")
	}
}

func TestFilterCaptainCandidateUsersExcludesCurrentUser(t *testing.T) {
	users := []model.User{
		{BaseModel: model.BaseModel{ID: 7}},
		{BaseModel: model.BaseModel{ID: 42}},
		{BaseModel: model.BaseModel{ID: 84}},
	}

	filtered := filterCaptainCandidateUsers(42, users)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 users after filtering, got %d", len(filtered))
	}
	if filtered[0].ID != 7 {
		t.Fatalf("expected first retained user to be 7, got %d", filtered[0].ID)
	}
	if filtered[1].ID != 84 {
		t.Fatalf("expected second retained user to be 84, got %d", filtered[1].ID)
	}
}
