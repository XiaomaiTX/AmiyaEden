package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"gorm.io/gorm"
)

func TestCorporationPolicyServiceRejectsInvalidCapability(t *testing.T) {
	db := newCorpPolicyTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	svc := NewCorporationPolicyService()
	err := svc.UpdatePolicies(CorporationPolicyConfig{
		Version:     1,
		DefaultMode: "deny",
		Policies: []CorporationPolicy{{
			CorporationID: 1001,
			Capabilities:  []string{"invalid.capability"},
		}},
	})
	if err == nil {
		t.Fatal("expected invalid capability error, got nil")
	}
}

// Reserved (catalog-only) capabilities must be rejected on write. The whole
// point of Stage 0A is that admins can no longer toggle keys that no backend
// route or service actually checks.
func TestCorporationPolicyServiceRejectsReservedCapability(t *testing.T) {
	db := newCorpPolicyTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	svc := NewCorporationPolicyService()
	err := svc.UpdatePolicies(CorporationPolicyConfig{
		Version:     1,
		DefaultMode: "deny",
		Policies: []CorporationPolicy{{
			CorporationID: 1001,
			Capabilities:  []string{model.CorpCapabilityMenuRole},
		}},
	})
	if err == nil {
		t.Fatalf("expected reserved capability error, got nil")
	}
	if !errors.Is(err, ErrInvalidCorporationAccessPolicy) {
		t.Fatalf("expected ErrInvalidCorporationAccessPolicy, got %v", err)
	}
}

// EffectiveCapabilities expands full-access / default-allow to the entire
// enforced catalog so the frontend can show every menu the backend would
// actually let through.
func TestEffectiveCapabilitiesExpandsFullAccess(t *testing.T) {
	ctx := UserCorpPolicyContext{
		FullAccess:   true,
		Capabilities: []string{model.CorpCapabilitySRPUser},
	}
	got := EffectiveCapabilities(ctx)
	if len(got) != len(model.EnforcedCorpCapabilities()) {
		t.Fatalf("effective capabilities len = %d, want %d", len(got), len(model.EnforcedCorpCapabilities()))
	}
}

func TestEffectiveCapabilitiesPassthroughWhenNotFullAccess(t *testing.T) {
	ctx := UserCorpPolicyContext{
		FullAccess:   false,
		Capabilities: []string{model.CorpCapabilitySRPUser, model.CorpCapabilityInfoWalletRead},
	}
	got := EffectiveCapabilities(ctx)
	if len(got) != 2 {
		t.Fatalf("effective capabilities len = %d, want 2", len(got))
	}
}

// Stored policies containing legacy (no-longer-enforced) capabilities must
// remain readable so admins see what's there. The next save drops them.
func TestCorporationPolicyServiceLoadDropsLegacyCapabilities(t *testing.T) {
	db := newCorpPolicyTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	// Seed a stored blob that contains both an enforced and a reserved key.
	stored := `{"version":1,"default_mode":"deny","policies":[{"corporation_id":1001,"full_access":false,"capabilities":["srp.user","menu.role"],"rules":{}}]}`
	repo := repository.NewSysConfigRepository()
	if err := repo.Set(model.SysConfigCorporationAccessPolicies, stored, ""); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	svc := NewCorporationPolicyService()
	got := svc.GetPolicies()
	if len(got.Policies) != 1 {
		t.Fatalf("policies len = %d, want 1", len(got.Policies))
	}
	if len(got.Policies[0].Capabilities) != 1 || got.Policies[0].Capabilities[0] != model.CorpCapabilitySRPUser {
		t.Fatalf("capabilities = %#v, want [srp.user]", got.Policies[0].Capabilities)
	}

	legacy := svc.StoredLegacyCapabilities()
	if list, ok := legacy[1001]; !ok || len(list) != 1 || list[0] != model.CorpCapabilityMenuRole {
		t.Fatalf("legacy = %#v, want {1001:[menu.role]}", legacy)
	}
}

func TestCorporationPolicyServiceBuildUserPolicyContext(t *testing.T) {
	db := newCorpPolicyTestDB(t)
	seedCorpPolicyUser(t, db, 1, 9001, 1001)

	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	svc := NewCorporationPolicyService()
	if err := svc.UpdatePolicies(CorporationPolicyConfig{
		Version:     2,
		DefaultMode: "deny",
		Policies: []CorporationPolicy{{
			CorporationID: 1001,
			Capabilities:  []string{model.CorpCapabilitySRPUser},
			Rules:         map[string]any{"flag": true},
		}},
	}); err != nil {
		t.Fatalf("update policies: %v", err)
	}

	ctx := svc.BuildUserPolicyContext(1)
	if ctx.PrimaryCorporationID != 1001 {
		t.Fatalf("primary corp = %d, want 1001", ctx.PrimaryCorporationID)
	}
	if len(ctx.Capabilities) != 1 || ctx.Capabilities[0] != model.CorpCapabilitySRPUser {
		t.Fatalf("capabilities = %#v, want [%q]", ctx.Capabilities, model.CorpCapabilitySRPUser)
	}
	if got, ok := ctx.Rules["flag"].(bool); !ok || !got {
		t.Fatalf("rules.flag = %#v, want true", ctx.Rules["flag"])
	}
}

func TestCorporationPolicyServiceReadsFreshConfigAfterUpdate(t *testing.T) {
	db := newCorpPolicyTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	svc := NewCorporationPolicyService()
	if err := svc.UpdatePolicies(CorporationPolicyConfig{
		Version:     1,
		DefaultMode: "deny",
		Policies:    []CorporationPolicy{{CorporationID: 1001}},
	}); err != nil {
		t.Fatalf("update policies: %v", err)
	}

	got := svc.GetPolicies()
	if len(got.Policies) != 1 {
		t.Fatalf("policies len = %d, want 1", len(got.Policies))
	}

	// Direct DB writes must be visible to subsequent reads now that the
	// in-process cache is gone. Cross-instance freshness relies on the shared
	// SysConfig Redis cache + DB fallback.
	if err := db.Exec("UPDATE system_config SET value = ? WHERE key = ?", `{"version":1,"default_mode":"deny","policies":[]}`, model.SysConfigCorporationAccessPolicies).Error; err != nil {
		t.Fatalf("direct update config: %v", err)
	}
	fresh := svc.GetPolicies()
	if len(fresh.Policies) != 0 {
		t.Fatalf("expected fresh policies len 0, got %d", len(fresh.Policies))
	}
}

func TestCorporationPolicyServiceDefaultModeAllowGrantsUnmatchedCorporation(t *testing.T) {
	db := newCorpPolicyTestDB(t)
	seedCorpPolicyUser(t, db, 1, 9001, 1001)

	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	svc := NewCorporationPolicyService()
	if err := svc.UpdatePolicies(CorporationPolicyConfig{
		Version:     1,
		DefaultMode: "allow",
		Policies:    []CorporationPolicy{},
	}); err != nil {
		t.Fatalf("update policies: %v", err)
	}

	if !svc.UserHasCapability(1, []string{model.RoleUser}, model.CorpCapabilityMenuDashboard) {
		t.Fatal("expected unmatched corporation to be allowed when default_mode=allow")
	}
}

func TestCorporationPolicyServiceDefaultModeDenyRejectsUnmatchedCorporation(t *testing.T) {
	db := newCorpPolicyTestDB(t)
	seedCorpPolicyUser(t, db, 1, 9001, 1001)

	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	svc := NewCorporationPolicyService()
	if err := svc.UpdatePolicies(CorporationPolicyConfig{
		Version:     1,
		DefaultMode: "deny",
		Policies:    []CorporationPolicy{},
	}); err != nil {
		t.Fatalf("update policies: %v", err)
	}

	if svc.UserHasCapability(1, []string{model.RoleUser}, model.CorpCapabilityMenuDashboard) {
		t.Fatal("expected unmatched corporation to be denied when default_mode=deny")
	}
}

func TestCorporationPolicyServiceNormalizeDefaultMode(t *testing.T) {
	cfg, err := normalizeCorpPolicyConfig(CorporationPolicyConfig{
		Version: 1,
	})
	if err != nil {
		t.Fatalf("normalize empty mode: %v", err)
	}
	if cfg.DefaultMode != "allow" {
		t.Fatalf("default_mode = %q, want allow", cfg.DefaultMode)
	}

	if _, err := normalizeCorpPolicyConfig(CorporationPolicyConfig{
		Version:     1,
		DefaultMode: "deny",
	}); err != nil {
		t.Fatalf("normalize deny mode: %v", err)
	}

	if _, err := normalizeCorpPolicyConfig(CorporationPolicyConfig{
		Version:     1,
		DefaultMode: "allow",
	}); err != nil {
		t.Fatalf("normalize allow mode: %v", err)
	}

	if _, err := normalizeCorpPolicyConfig(CorporationPolicyConfig{
		Version:     1,
		DefaultMode: "invalid",
	}); err == nil {
		t.Fatal("expected invalid default_mode error")
	}
}

func TestCorporationPolicyServiceGetRuleFloat(t *testing.T) {
	db := newCorpPolicyTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	svc := NewCorporationPolicyService()
	if err := svc.UpdatePolicies(CorporationPolicyConfig{
		Version:     1,
		DefaultMode: "deny",
		Policies: []CorporationPolicy{{
			CorporationID: 1001,
			Rules: map[string]any{
				CorpRuleSRPRecommendationMultiplier: 0.65,
			},
		}},
	}); err != nil {
		t.Fatalf("update policies: %v", err)
	}

	got := svc.GetRuleFloat(1001, CorpRuleSRPRecommendationMultiplier, 1)
	if got != 0.65 {
		t.Fatalf("rule float = %v, want 0.65", got)
	}
	fallback := svc.GetRuleFloat(1002, CorpRuleSRPRecommendationMultiplier, 1)
	if fallback != 1 {
		t.Fatalf("fallback float = %v, want 1", fallback)
	}
}

func TestCorporationPolicyServiceGetRuleIntAndBool(t *testing.T) {
	db := newCorpPolicyTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	svc := NewCorporationPolicyService()
	if err := svc.UpdatePolicies(CorporationPolicyConfig{
		Version:     1,
		DefaultMode: "deny",
		Policies: []CorporationPolicy{{
			CorporationID: 1001,
			Rules: map[string]any{
				"custom.int":  180,
				"custom.bool": false,
			},
		}},
	}); err != nil {
		t.Fatalf("update policies: %v", err)
	}

	if got := svc.GetRuleInt(1001, "custom.int", 365); got != 180 {
		t.Fatalf("GetRuleInt() = %d, want 180", got)
	}
	if got := svc.GetRuleBool(1001, "custom.bool", true); got {
		t.Fatal("GetRuleBool() = true, want false")
	}
	if got := svc.GetRuleBool(1001, "missing.bool", true); !got {
		t.Fatal("GetRuleBool() fallback = false, want true")
	}
}

func newCorpPolicyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newServiceTestDB(t, "corp_policy_test", &model.SystemConfig{}, &model.User{}, &model.EveCharacter{})
}

func seedCorpPolicyUser(t *testing.T, db *gorm.DB, userID uint, characterID int64, corporationID int64) {
	t.Helper()
	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: userID},
		PrimaryCharacterID: characterID,
		Role:               model.RoleUser,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   characterID,
		CharacterName: "Main",
		CorporationID: corporationID,
		UserID:        userID,
	}).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}
}

// clearCorpPolicyCache is a no-op kept for tests that previously reset the
// in-process cache. The cache was removed in Stage 0A; policy reads now go
// straight to the shared SysConfig store so there is nothing to clear.
func clearCorpPolicyCache() {}

// setCorpPolicyCache persists the supplied config directly to system_config,
// replacing the previous in-process cache setter. Tests that want a specific
// policy visible to CorporationPolicyService without going through
// UpdatePolicies validation should use this helper.
func setCorpPolicyCache(cfg CorporationPolicyConfig) {
	if global.DB == nil {
		return
	}
	serialized, err := json.Marshal(cfg)
	if err != nil {
		panic(fmt.Sprintf("marshal corp policy: %v", err))
	}
	repo := repository.NewSysConfigRepository()
	if err := repo.Set(model.SysConfigCorporationAccessPolicies, string(serialized), "军团能力策略配置"); err != nil {
		panic(fmt.Sprintf("set corp policy: %v", err))
	}
}
