package service

import (
	"amiya-eden/internal/model"
	"testing"
	"time"
)

func TestBuildCaptainRewardSettlementItemsUsesCurrentCaptainProfile(t *testing.T) {
	processedAt := time.Date(2026, 3, 27, 22, 0, 0, 0, time.UTC)
	rows := []model.CaptainRewardSettlement{
		{
			BaseModel:          model.BaseModel{ID: 11},
			CaptainUserID:      3001,
			ProcessedAt:        processedAt,
			AttributionCount:   4,
			AttributedISKTotal: 124_600_000,
			BonusRate:          20,
			CreditedValue:      24.92,
		},
	}
	profiles := map[uint]captainProfile{
		3001: {
			Nickname:             "Captain Bee",
			PrimaryCharacterID:   8001,
			PrimaryCharacterName: "Captain Prime",
		},
	}

	items := buildCaptainRewardSettlementItems(rows, profiles)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.ID != 11 {
		t.Fatalf("expected settlement id 11, got %d", item.ID)
	}
	if item.CaptainUserID != 3001 {
		t.Fatalf("expected captain user id 3001, got %d", item.CaptainUserID)
	}
	if item.CaptainCharacterID != 8001 {
		t.Fatalf("expected captain character id 8001, got %d", item.CaptainCharacterID)
	}
	if item.CaptainCharacterName != "Captain Prime" {
		t.Fatalf("expected captain character name Captain Prime, got %q", item.CaptainCharacterName)
	}
	if item.CaptainNickname != "Captain Bee" {
		t.Fatalf("expected captain nickname Captain Bee, got %q", item.CaptainNickname)
	}
	if item.AttributionCount != 4 {
		t.Fatalf("expected attribution count 4, got %d", item.AttributionCount)
	}
	if item.AttributedISKTotal != 124_600_000 {
		t.Fatalf("expected attributed total 124600000, got %v", item.AttributedISKTotal)
	}
	if item.BonusRate != 20 {
		t.Fatalf("expected bonus rate 20, got %v", item.BonusRate)
	}
	if item.CreditedValue != 24.92 {
		t.Fatalf("expected credited value 24.92, got %v", item.CreditedValue)
	}
	if !item.ProcessedAt.Equal(processedAt) {
		t.Fatalf("expected processed_at %v, got %v", processedAt, item.ProcessedAt)
	}
}
