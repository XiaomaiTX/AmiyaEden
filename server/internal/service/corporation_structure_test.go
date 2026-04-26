package service

import (
	"amiya-eden/internal/model"
	"testing"
)

func TestDeduplicateManagedCorporationIDs(t *testing.T) {
	chars := []model.EveCharacter{
		{CharacterID: 1, CorporationID: 100},
		{CharacterID: 2, CorporationID: 200},
		{CharacterID: 3, CorporationID: 100},
		{CharacterID: 4, CorporationID: 0},
		{CharacterID: 5, CorporationID: 300},
	}

	got := deduplicateManagedCorporationIDs(chars, []int64{100, 300, 400})
	want := []int64{100, 300}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestValidateAuthorizationBindings(t *testing.T) {
	managed := map[int64]struct{}{100: {}, 200: {}}
	directors := map[int64]map[int64]struct{}{
		100: {10: {}, 11: {}},
		200: {20: {}},
	}

	t.Run("accepts valid bindings", func(t *testing.T) {
		err := validateAuthorizationBindings(
			[]CorporationStructureAuthorizationBinding{
				{CorporationID: 100, CharacterID: 10},
				{CorporationID: 200, CharacterID: 0},
			},
			managed,
			directors,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects duplicate corporation binding", func(t *testing.T) {
		err := validateAuthorizationBindings(
			[]CorporationStructureAuthorizationBinding{
				{CorporationID: 100, CharacterID: 10},
				{CorporationID: 100, CharacterID: 11},
			},
			managed,
			directors,
		)
		if err == nil {
			t.Fatal("expected duplicate corporation to be rejected")
		}
	})

	t.Run("rejects unmanaged corporation", func(t *testing.T) {
		err := validateAuthorizationBindings(
			[]CorporationStructureAuthorizationBinding{
				{CorporationID: 300, CharacterID: 10},
			},
			managed,
			directors,
		)
		if err == nil {
			t.Fatal("expected unmanaged corporation to be rejected")
		}
	})

	t.Run("rejects non director character", func(t *testing.T) {
		err := validateAuthorizationBindings(
			[]CorporationStructureAuthorizationBinding{
				{CorporationID: 200, CharacterID: 10},
			},
			managed,
			directors,
		)
		if err == nil {
			t.Fatal("expected non-director character to be rejected")
		}
	})
}
