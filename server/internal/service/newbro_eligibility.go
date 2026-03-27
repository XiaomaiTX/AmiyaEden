package service

import (
	"amiya-eden/internal/model"
	"fmt"
	"time"
)

const (
	NewbroDisqualifiedReasonSkillPointThresholdReached               = "skill_point_threshold_reached"
	NewbroDisqualifiedReasonMultiCharacterSkillPointThresholdReached = "multi_character_skill_point_threshold_reached"
)

type NewbroEligibilityRules struct {
	MaxCharacterSP          int64
	MultiCharacterSP        int64
	MultiCharacterThreshold int
	AttributionLookbackDays int
}

type NewbroCharacterSnapshot struct {
	CharacterID   int64
	CorporationID int64
	TotalSP       int64
}

type NewbroEligibilityResult struct {
	IsCurrentlyNewbro  bool
	DisqualifiedReason string
}

func BuildNewbroRuleVersion(rules NewbroEligibilityRules) string {
	return fmt.Sprintf(
		"sp:%d;multi-sp:%d;multi-count:%d;lookback-days:%d",
		rules.MaxCharacterSP,
		rules.MultiCharacterSP,
		rules.MultiCharacterThreshold,
		rules.AttributionLookbackDays,
	)
}

func EvaluateNewbroEligibility(characters []NewbroCharacterSnapshot, rules NewbroEligibilityRules) NewbroEligibilityResult {
	for _, character := range characters {
		if character.TotalSP >= rules.MaxCharacterSP {
			return NewbroEligibilityResult{
				IsCurrentlyNewbro:  false,
				DisqualifiedReason: NewbroDisqualifiedReasonSkillPointThresholdReached,
			}
		}
	}

	qualifiedCharacterCount := 0
	for _, character := range characters {
		if character.TotalSP >= rules.MultiCharacterSP {
			qualifiedCharacterCount++
		}
	}

	if qualifiedCharacterCount >= rules.MultiCharacterThreshold {
		return NewbroEligibilityResult{
			IsCurrentlyNewbro:  false,
			DisqualifiedReason: NewbroDisqualifiedReasonMultiCharacterSkillPointThresholdReached,
		}
	}

	return NewbroEligibilityResult{IsCurrentlyNewbro: true}
}

func NeedsNewbroEligibilityRefresh(state *model.NewbroPlayerState, expectedRuleVersion string, now time.Time, refreshInterval time.Duration) bool {
	if state == nil {
		return true
	}
	if state.RuleVersion != expectedRuleVersion {
		return true
	}
	if !state.IsCurrentlyNewbro {
		return false
	}
	return now.Sub(state.EvaluatedAt) > refreshInterval
}
