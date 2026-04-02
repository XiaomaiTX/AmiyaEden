package service

import (
	"amiya-eden/internal/model"
	"testing"
	"time"
)

func TestBuildShopOrderResponsesIncludesReviewerNickname(t *testing.T) {
	reviewerID := uint(77)
	createdAt := time.Date(2026, time.April, 3, 8, 0, 0, 0, time.UTC)

	orders := []model.ShopOrder{
		{
			BaseModel:  model.BaseModel{ID: 1, CreatedAt: createdAt},
			OrderNo:    "ORDER-1",
			Status:     model.OrderStatusDelivered,
			ReviewedBy: &reviewerID,
		},
		{
			BaseModel: model.BaseModel{ID: 2, CreatedAt: createdAt},
			OrderNo:   "ORDER-2",
			Status:    model.OrderStatusRequested,
		},
	}

	got := buildShopOrderResponses(orders, map[uint]string{reviewerID: "Logistics Fox"})

	if len(got) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(got))
	}
	if got[0].ReviewerName != "Logistics Fox" {
		t.Fatalf("expected reviewer nickname to be included, got %q", got[0].ReviewerName)
	}
	if got[1].ReviewerName != "" {
		t.Fatalf("expected empty reviewer nickname for unreviewed order, got %q", got[1].ReviewerName)
	}
}
