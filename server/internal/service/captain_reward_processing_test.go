package service

import (
	"amiya-eden/internal/model"
	"testing"
	"time"
)

func TestCalculateCaptainRewardCreditRoundsToTwoDecimals(t *testing.T) {
	t.Run("uses percentage-based bonus rate", func(t *testing.T) {
		got := calculateCaptainRewardCredit(124_600_000, 20)
		if got != 24.92 {
			t.Fatalf("expected 24.92, got %v", got)
		}
	})

	t.Run("100 percent converts million isk directly to wallet value", func(t *testing.T) {
		got := calculateCaptainRewardCredit(124_600_000, 100)
		if got != 124.6 {
			t.Fatalf("expected 124.6, got %v", got)
		}
	})

	t.Run("rounds half up at two decimals", func(t *testing.T) {
		got := calculateCaptainRewardCredit(12_345_000, 20)
		if got != 2.47 {
			t.Fatalf("expected 2.47, got %v", got)
		}
	})
}

func TestBuildCaptainRewardProcessingBatchesGroupsRowsPerCaptain(t *testing.T) {
	processedAt := time.Date(2026, 3, 27, 21, 30, 0, 0, time.UTC)
	rows := []model.CaptainBountyAttribution{
		{BaseModel: model.BaseModel{ID: 1}, CaptainUserID: 3001, Amount: 100_000_000},
		{BaseModel: model.BaseModel{ID: 2}, CaptainUserID: 3001, Amount: 24_600_000},
		{BaseModel: model.BaseModel{ID: 3}, CaptainUserID: 3002, Amount: 10_000_000},
	}

	batches := buildCaptainRewardProcessingBatches(rows, 20, processedAt)
	if len(batches) != 2 {
		t.Fatalf("expected 2 captain batches, got %d", len(batches))
	}

	first := batches[0]
	if first.CaptainUserID != 3001 {
		t.Fatalf("expected first batch captain 3001, got %d", first.CaptainUserID)
	}
	if len(first.AttributionIDs) != 2 {
		t.Fatalf("expected first batch to include 2 attributions, got %d", len(first.AttributionIDs))
	}
	if first.AttributionCount != 2 {
		t.Fatalf("expected first batch count 2, got %d", first.AttributionCount)
	}
	if first.AttributedISKTotal != 124_600_000 {
		t.Fatalf("expected first batch ISK total 124600000, got %v", first.AttributedISKTotal)
	}
	if first.BonusRate != 20 {
		t.Fatalf("expected bonus rate 20, got %v", first.BonusRate)
	}
	if first.CreditedValue != 24.92 {
		t.Fatalf("expected credited value 24.92, got %v", first.CreditedValue)
	}
	if !first.ProcessedAt.Equal(processedAt) {
		t.Fatalf("expected processed_at %v, got %v", processedAt, first.ProcessedAt)
	}

	second := batches[1]
	if second.CaptainUserID != 3002 {
		t.Fatalf("expected second batch captain 3002, got %d", second.CaptainUserID)
	}
	if second.CreditedValue != 2 {
		t.Fatalf("expected second credited value 2, got %v", second.CreditedValue)
	}
}
