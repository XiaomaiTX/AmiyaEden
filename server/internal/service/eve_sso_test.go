package service

import (
	"slices"
	"testing"
)

func TestBuildLoginScopesIncludesPublicDataRegisteredAndExtraScopes(t *testing.T) {
	scopeMu.Lock()
	original := slices.Clone(registeredScopes)
	registeredScopes = []RegisteredScope{
		{Module: "killmail", Scope: "esi-killmails.read_killmails.v1"},
		{Module: "wallet", Scope: "  esi-wallet.read_character_wallet.v1  "},
		{Module: "empty", Scope: "   "},
	}
	scopeMu.Unlock()
	t.Cleanup(func() {
		scopeMu.Lock()
		registeredScopes = original
		scopeMu.Unlock()
	})

	scopes := buildLoginScopes([]string{
		"esi-location.read_location.v1",
		"esi-wallet.read_character_wallet.v1",
	})

	if !slices.Contains(scopes, "publicData") {
		t.Fatal("expected publicData to be included")
	}
	if !slices.Contains(scopes, "esi-killmails.read_killmails.v1") {
		t.Fatal("expected registered killmail scope to be included")
	}
	if !slices.Contains(scopes, "esi-wallet.read_character_wallet.v1") {
		t.Fatal("expected trimmed wallet scope to be included")
	}
	if !slices.Contains(scopes, "esi-location.read_location.v1") {
		t.Fatal("expected extra scope to be included")
	}

	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		if _, exists := seen[scope]; exists {
			t.Fatalf("duplicate scope %q in result %v", scope, scopes)
		}
		seen[scope] = struct{}{}
	}
}

func TestGetRegisteredScopesReturnsCopy(t *testing.T) {
	scopeMu.Lock()
	original := slices.Clone(registeredScopes)
	registeredScopes = []RegisteredScope{
		{Module: "killmail", Scope: "esi-killmails.read_killmails.v1"},
	}
	scopeMu.Unlock()
	t.Cleanup(func() {
		scopeMu.Lock()
		registeredScopes = original
		scopeMu.Unlock()
	})

	got := GetRegisteredScopes()
	got[0].Scope = "mutated"

	scopeMu.RLock()
	defer scopeMu.RUnlock()
	if registeredScopes[0].Scope != "esi-killmails.read_killmails.v1" {
		t.Fatalf("expected registeredScopes to remain unchanged, got %q", registeredScopes[0].Scope)
	}
}
