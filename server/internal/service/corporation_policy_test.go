package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
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

func TestCorporationPolicyServiceCacheInvalidation(t *testing.T) {
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

	if err := db.Exec("UPDATE system_config SET value = ? WHERE key = ?", `{"version":1,"default_mode":"deny","policies":[]}`, model.SysConfigCorporationAccessPolicies).Error; err != nil {
		t.Fatalf("direct update config: %v", err)
	}
	stale := svc.GetPolicies()
	if len(stale.Policies) != 1 {
		t.Fatalf("expected cached policies len 1, got %d", len(stale.Policies))
	}
	svc.InvalidateCache()
	fresh := svc.GetPolicies()
	if len(fresh.Policies) != 0 {
		t.Fatalf("expected fresh policies len 0, got %d", len(fresh.Policies))
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
	dsn := "file:corp_policy_test_" + time.Now().Format("150405.000000000") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.User{}, &model.EveCharacter{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
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
