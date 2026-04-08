package service

import (
	"testing"
)

func TestBuildLoginScopes_excludesNonRequired(t *testing.T) {
	// Reset global scope state
	scopeMu.Lock()
	orig := registeredScopes
	registeredScopes = nil
	scopeMu.Unlock()
	defer func() {
		scopeMu.Lock()
		registeredScopes = orig
		scopeMu.Unlock()
	}()

	RegisterScope("test", "esi-required.v1", "required scope", true)
	RegisterScope("test", "esi-optional.v1", "optional scope", false)

	scopes := buildLoginScopes(nil)

	scopeSet := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		scopeSet[s] = true
	}

	if !scopeSet["esi-required.v1"] {
		t.Error("expected required scope to be included")
	}
	if scopeSet["esi-optional.v1"] {
		t.Error("expected optional scope to be excluded")
	}
	if !scopeSet["publicData"] {
		t.Error("expected publicData to always be included")
	}
}

func TestBuildLoginScopes_extraScopesOverrideOptional(t *testing.T) {
	scopeMu.Lock()
	orig := registeredScopes
	registeredScopes = nil
	scopeMu.Unlock()
	defer func() {
		scopeMu.Lock()
		registeredScopes = orig
		scopeMu.Unlock()
	}()

	RegisterScope("test", "esi-optional.v1", "optional scope", false)

	scopes := buildLoginScopes([]string{"esi-optional.v1"})

	scopeSet := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		scopeSet[s] = true
	}

	if !scopeSet["esi-optional.v1"] {
		t.Error("expected optional scope to be included when passed as extra")
	}
}
