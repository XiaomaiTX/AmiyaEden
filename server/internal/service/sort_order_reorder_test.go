package service

import (
	"amiya-eden/internal/model"
	"testing"
)

func TestBuildSkillPlanScopeReorderUpdatesRebuildsContinuousSortOrder(t *testing.T) {
	updates, err := buildSkillPlanScopeReorderUpdates(
		[]uint{2, 1},
		[]model.SkillPlan{
			{ID: 1, SortOrder: 0},
			{ID: 2, SortOrder: 0},
			{ID: 3, SortOrder: 0},
		},
	)
	if err != nil {
		t.Fatalf("buildSkillPlanScopeReorderUpdates returned error: %v", err)
	}

	expectedIDs := []uint{2, 1, 3}
	expectedSortOrders := []int{10, 20, 30}
	if len(updates) != len(expectedIDs) {
		t.Fatalf("expected %d updates, got %d", len(expectedIDs), len(updates))
	}
	for index := range expectedIDs {
		if updates[index].ID != expectedIDs[index] || updates[index].SortOrder != expectedSortOrders[index] {
			t.Fatalf("update %d = %+v, want {ID:%d SortOrder:%d}", index, updates[index], expectedIDs[index], expectedSortOrders[index])
		}
	}
}

func TestBuildSkillPlanScopeReorderUpdatesKeepsOtherPagesStable(t *testing.T) {
	updates, err := buildSkillPlanScopeReorderUpdates(
		[]uint{4, 3},
		[]model.SkillPlan{
			{ID: 1, SortOrder: 10},
			{ID: 2, SortOrder: 20},
			{ID: 3, SortOrder: 30},
			{ID: 4, SortOrder: 40},
			{ID: 5, SortOrder: 50},
		},
	)
	if err != nil {
		t.Fatalf("buildSkillPlanScopeReorderUpdates returned error: %v", err)
	}

	expectedIDs := []uint{1, 2, 4, 3, 5}
	if len(updates) != len(expectedIDs) {
		t.Fatalf("expected %d updates, got %d", len(expectedIDs), len(updates))
	}
	for index := range expectedIDs {
		if updates[index].ID != expectedIDs[index] {
			t.Fatalf("update %d id = %d, want %d", index, updates[index].ID, expectedIDs[index])
		}
		wantSort := (index + 1) * skillPlanSortOrderStep
		if updates[index].SortOrder != wantSort {
			t.Fatalf("update %d sort_order = %d, want %d", index, updates[index].SortOrder, wantSort)
		}
	}
}

func TestBuildSkillPlanScopeReorderUpdatesRejectsInvalidIDs(t *testing.T) {
	plans := []model.SkillPlan{
		{ID: 1, SortOrder: 10},
		{ID: 2, SortOrder: 20},
	}

	_, err := buildSkillPlanScopeReorderUpdates([]uint{1, 1}, plans)
	if err == nil {
		t.Fatal("expected error for duplicate reorder id")
	}

	_, err = buildSkillPlanScopeReorderUpdates([]uint{3}, plans)
	if err == nil {
		t.Fatal("expected error for missing reorder id")
	}
}

func TestBuildSortOrderAssignmentsRejectsMissingIDs(t *testing.T) {
	_, err := buildSortOrderAssignments(
		[]uint{1, 2},
		[]sortOrderRecord{{ID: 1, SortOrder: 10}},
	)
	if err == nil {
		t.Fatal("expected error for missing reorder item")
	}
}
