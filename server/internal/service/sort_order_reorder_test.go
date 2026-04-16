package service

import "testing"

func TestBuildSortOrderAssignmentsPreservesVisibleSlots(t *testing.T) {
	assignments, err := buildSortOrderAssignments(
		[]uint{22, 11, 33},
		[]sortOrderRecord{
			{ID: 11, SortOrder: 10},
			{ID: 22, SortOrder: 20},
			{ID: 33, SortOrder: 30},
		},
	)
	if err != nil {
		t.Fatalf("buildSortOrderAssignments returned error: %v", err)
	}

	expected := []sortOrderAssignment{
		{ID: 22, SortOrder: 10},
		{ID: 11, SortOrder: 20},
		{ID: 33, SortOrder: 30},
	}

	if len(assignments) != len(expected) {
		t.Fatalf("expected %d assignments, got %d", len(expected), len(assignments))
	}
	for index := range expected {
		if assignments[index] != expected[index] {
			t.Fatalf("assignment %d = %+v, want %+v", index, assignments[index], expected[index])
		}
	}
}

func TestBuildSortOrderAssignmentsPreservesSparseSortValues(t *testing.T) {
	assignments, err := buildSortOrderAssignments(
		[]uint{8, 3, 5},
		[]sortOrderRecord{
			{ID: 3, SortOrder: 300},
			{ID: 5, SortOrder: 500},
			{ID: 8, SortOrder: 800},
		},
	)
	if err != nil {
		t.Fatalf("buildSortOrderAssignments returned error: %v", err)
	}

	expected := []sortOrderAssignment{
		{ID: 8, SortOrder: 300},
		{ID: 3, SortOrder: 500},
		{ID: 5, SortOrder: 800},
	}

	for index := range expected {
		if assignments[index] != expected[index] {
			t.Fatalf("assignment %d = %+v, want %+v", index, assignments[index], expected[index])
		}
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
