package service

import (
	"amiya-eden/internal/model"
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

func TestValidateExtraScopesRejectsCorpKillmailScopeForNonAdmin(t *testing.T) {
	err := validateExtraScopes([]string{"esi-killmails.read_corporation_killmails.v1"}, []string{model.RoleSRP})
	if err == nil {
		t.Fatal("expected non-admin user to be rejected for corporation killmail scope")
	}
}

func TestValidateExtraScopesRejectsCorpKillmailScopeForPublicLogin(t *testing.T) {
	err := validateExtraScopes([]string{"esi-killmails.read_corporation_killmails.v1"}, nil)
	if err == nil {
		t.Fatal("expected public login request to be rejected for corporation killmail scope")
	}
}

func TestValidateExtraScopesAllowsCorpKillmailScopeForAdmins(t *testing.T) {
	for _, roles := range [][]string{{model.RoleAdmin}, {model.RoleSuperAdmin}} {
		if err := validateExtraScopes([]string{"esi-killmails.read_corporation_killmails.v1"}, roles); err != nil {
			t.Fatalf("expected roles %v to be allowed, got %v", roles, err)
		}
	}
}

func TestValidateExtraScopesAllowsRegularScopesForLoggedInUsers(t *testing.T) {
	err := validateExtraScopes([]string{"esi-location.read_location.v1"}, []string{model.RoleUser})
	if err != nil {
		t.Fatalf("expected regular scope to be allowed, got %v", err)
	}
}

func replaceRegisteredScopes(t *testing.T, scopes ...RegisteredScope) {
	t.Helper()
	scopeMu.Lock()
	orig := registeredScopes
	registeredScopes = scopes
	scopeMu.Unlock()
	t.Cleanup(func() {
		scopeMu.Lock()
		registeredScopes = orig
		scopeMu.Unlock()
	})
}

func TestIsAdminOnlyScope(t *testing.T) {
	if !isAdminOnlyScope(corpKillmailScope) {
		t.Fatal("expected corp killmail scope to be admin-only")
	}
	if isAdminOnlyScope("esi-assets.read_corporation_assets.v1") {
		t.Fatal("expected assets scope to not be admin-only")
	}
	if !isAdminOnlyScope(" " + corpKillmailScope + " ") {
		t.Fatal("expected admin-only check to trim whitespace")
	}
}

func TestAllowedScopeSetIncludesPublicDataAndOptional(t *testing.T) {
	replaceRegisteredScopes(t,
		RegisteredScope{Module: "structures", Scope: "esi-corporations.read_structures.v1", Required: true},
		RegisteredScope{Module: "structures", Scope: "esi-assets.read_corporation_assets.v1", Required: false},
	)

	set := allowedScopeSet()
	for _, want := range []string{"publicData", "esi-corporations.read_structures.v1", "esi-assets.read_corporation_assets.v1"} {
		if _, ok := set[want]; !ok {
			t.Fatalf("expected allowedScopeSet to contain %q, got %v", want, set)
		}
	}
}

func TestCandidateScopeSetMergesAndFiltersUnregistered(t *testing.T) {
	replaceRegisteredScopes(t,
		RegisteredScope{Module: "structures", Scope: "esi-corporations.read_structures.v1", Required: true},
		RegisteredScope{Module: "structures", Scope: "esi-assets.read_corporation_assets.v1", Required: false},
	)

	allowed := allowedScopeSet()
	got := candidateScopeSet(
		"publicData esi-corporations.read_structures.v1 esi-assets.read_corporation_assets.v1 esi-retired.v1",
		[]string{"publicData", "esi-corporations.read_structures.v1"},
		allowed,
	)

	for _, want := range []string{"publicData", "esi-corporations.read_structures.v1", "esi-assets.read_corporation_assets.v1"} {
		if _, ok := got[want]; !ok {
			t.Fatalf("expected candidate set to retain %q, got %v", want, got)
		}
	}
	if _, ok := got["esi-retired.v1"]; ok {
		t.Fatalf("expected unregistered scope to be filtered, got %v", got)
	}
}

func TestScopeSnapshotSortedDedupedAndStripsAdminOnly(t *testing.T) {
	set := map[string]struct{}{
		"esi-assets.read_corporation_assets.v1": {},
		corpKillmailScope:                        {},
		"publicData":                             {},
	}

	if got := scopeSnapshot(set, true); got != "esi-assets.read_corporation_assets.v1 esi-killmails.read_corporation_killmails.v1 publicData" {
		t.Fatalf("keepAdminOnly snapshot = %q", got)
	}
	if got := scopeSnapshot(set, false); got != "esi-assets.read_corporation_assets.v1 publicData" {
		t.Fatalf("strip snapshot = %q", got)
	}
}

func TestMissingOptionalScopesForReauth(t *testing.T) {
	replaceRegisteredScopes(t,
		RegisteredScope{Module: "structures", Scope: "esi-corporations.read_structures.v1", Required: true},
		RegisteredScope{Module: "structures", Scope: "esi-assets.read_corporation_assets.v1", Required: false},
		RegisteredScope{Module: "killmail", Scope: corpKillmailScope, Required: false},
	)

	existing := "publicData esi-corporations.read_structures.v1 esi-assets.read_corporation_assets.v1 " + corpKillmailScope

	mustContainOnly(t, missingOptionalScopesForReauth(existing, []string{"publicData", "esi-corporations.read_structures.v1"}),
		[]string{"esi-assets.read_corporation_assets.v1"})
	mustContainOnly(t, missingOptionalScopesForReauth(existing, nil),
		[]string{"esi-assets.read_corporation_assets.v1"})
	if got := missingOptionalScopesForReauth(existing,
		[]string{"publicData", "esi-corporations.read_structures.v1", "esi-assets.read_corporation_assets.v1", corpKillmailScope}); len(got) != 0 {
		t.Fatalf("expected no missing scopes when chain is complete, got %v", got)
	}
	if got := missingOptionalScopesForReauth("publicData esi-corporations.read_structures.v1", nil); len(got) != 0 {
		t.Fatalf("expected no missing scopes when nothing to restore, got %v", got)
	}
}

func mustContainOnly(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
