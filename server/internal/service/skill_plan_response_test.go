package service

import (
	"amiya-eden/internal/model"
	"encoding/json"
	"testing"
	"time"
)

func TestSkillPlanCheckCharacterResponseOmitsPortraitURL(t *testing.T) {
	payload, err := json.Marshal(SkillPlanCheckCharacterResp{
		CharacterID:   9001,
		CharacterName: "Amiya Prime",
	})
	if err != nil {
		t.Fatalf("marshal skill plan character response: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("unmarshal skill plan character response: %v", err)
	}

	if _, exists := raw["portrait_url"]; exists {
		t.Fatalf("expected skill plan character response to omit portrait_url, got %#v", raw["portrait_url"])
	}
}

func TestNewSkillPlanDetailRespIncludesSortOrder(t *testing.T) {
	createdAt := time.Date(2026, time.April, 17, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)

	resp := newSkillPlanDetailResp(
		&model.SkillPlan{
			ID:          42,
			Title:       "Shield Core",
			Description: "Baseline doctrine",
			SortOrder:   17,
			CreatedBy:   99,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		},
		"Basilisk",
		[]SkillPlanSkillResp{{ID: 1, SkillTypeID: 3300}},
	)

	payload, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal skill plan detail response: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("unmarshal skill plan detail response: %v", err)
	}

	if got, ok := raw["sort_order"].(float64); !ok || int(got) != 17 {
		t.Fatalf("expected sort_order=17, got %#v", raw["sort_order"])
	}
}
