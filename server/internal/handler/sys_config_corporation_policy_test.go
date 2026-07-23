package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/pkg/response"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUpdateCorporationAccessPoliciesRejectsInvalidCapability(t *testing.T) {
	db := newSysConfigHandlerTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	recorder := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/system/basic-config/corporation-access-policies", bytes.NewBufferString(`{
		"version":1,
		"default_mode":"allow",
		"policies":[{"corporation_id":1001,"capabilities":["invalid.cap"],"rules":{}}]
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	NewSysConfigHandler().UpdateCorporationAccessPolicies(ctx)

	var resp response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != response.CodeParamError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeParamError)
	}
}

func TestGetCorporationAccessPoliciesReturnsDefaultWhenMissing(t *testing.T) {
	db := newSysConfigHandlerTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	recorder := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/basic-config/corporation-access-policies", nil)

	NewSysConfigHandler().GetCorporationAccessPolicies(ctx)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Version     int                      `json:"version"`
			DefaultMode string                   `json:"default_mode"`
			Policies    []map[string]interface{} `json:"policies"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
	if resp.Data.Version != 1 {
		t.Fatalf("version = %d, want 1", resp.Data.Version)
	}
	if resp.Data.DefaultMode != "allow" {
		t.Fatalf("default_mode = %q, want allow", resp.Data.DefaultMode)
	}
	if len(resp.Data.Policies) != 0 {
		t.Fatalf("expected empty policies, got %#v", resp.Data.Policies)
	}
}

func TestUpdateCorporationAccessPoliciesPersistsConfig(t *testing.T) {
	db := newSysConfigHandlerTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	recorder := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(recorder)

	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: 42},
		PrimaryCharacterID: 900042,
		Role:               model.RoleUser,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   900042,
		CharacterName: "wallet user",
		UserID:        42,
		CorporationID: 1001,
	}).Error; err != nil {
		t.Fatalf("seed character: %v", err)
	}
	if err := db.Create(&model.SystemWallet{
		UserID:  42,
		Balance: 88.8,
	}).Error; err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/system/basic-config/corporation-access-policies", bytes.NewBufferString(`{
		"version":2,
		"default_mode":"allow",
		"policies":[{"corporation_id":1001,"full_access":false,"capabilities":["srp.user"],"rules":{"x":1}}]
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	NewSysConfigHandler().UpdateCorporationAccessPolicies(ctx)

	var resp response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}

	var stored model.SystemConfig
	if err := db.Where("key = ?", model.SysConfigCorporationAccessPolicies).First(&stored).Error; err != nil {
		t.Fatalf("query stored config: %v", err)
	}
	if stored.Value == "" {
		t.Fatal("expected stored policy value to be non-empty")
	}

	var wallet model.SystemWallet
	if err := db.Where("user_id = ?", 42).First(&wallet).Error; err != nil {
		t.Fatalf("query wallet: %v", err)
	}
	if wallet.Balance != 0 {
		t.Fatalf("wallet balance = %f, want 0", wallet.Balance)
	}
}

func TestUpdateCorporationAccessPoliciesRejectsReservedCapability(t *testing.T) {
	db := newSysConfigHandlerTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	recorder := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/system/basic-config/corporation-access-policies", bytes.NewBufferString(`{
		"version":1,
		"default_mode":"allow",
		"policies":[{"corporation_id":1001,"capabilities":["menu.role"],"rules":{}}]
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	NewSysConfigHandler().UpdateCorporationAccessPolicies(ctx)

	var resp response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != response.CodeParamError {
		t.Fatalf("response code = %d, want %d (reserved capability should be rejected)", resp.Code, response.CodeParamError)
	}
}

func TestGetCorporationAccessPoliciesExposesEnforcedAndLegacyCapabilities(t *testing.T) {
	db := newSysConfigHandlerTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	// Seed stored policy with one enforced + one legacy capability to verify
	// the GET handler surfaces both lists to the config UI.
	stored := `{"version":1,"default_mode":"deny","policies":[{"corporation_id":1001,"full_access":false,"capabilities":["srp.user","menu.role"],"rules":{}}]}`
	if err := db.Where("key = ?", model.SysConfigCorporationAccessPolicies).
		Assign(model.SystemConfig{Key: model.SysConfigCorporationAccessPolicies, Value: stored}).
		FirstOrCreate(&model.SystemConfig{}).Error; err != nil {
		t.Fatalf("seed config: %v", err)
	}

	recorder := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/basic-config/corporation-access-policies", nil)

	NewSysConfigHandler().GetCorporationAccessPolicies(ctx)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Version              int                `json:"version"`
			DefaultMode          string             `json:"default_mode"`
			Policies             []map[string]any   `json:"policies"`
			EnforcedCapabilities []string           `json:"enforced_capabilities"`
			LegacyCapabilities   map[int64][]string `json:"legacy_capabilities"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
	if len(resp.Data.EnforcedCapabilities) == 0 {
		t.Fatal("expected enforced_capabilities to be populated")
	}
	// Sanity: srp.user is in the enforced set, menu.role is not.
	enforced := make(map[string]struct{}, len(resp.Data.EnforcedCapabilities))
	for _, key := range resp.Data.EnforcedCapabilities {
		enforced[key] = struct{}{}
	}
	if _, ok := enforced["srp.user"]; !ok {
		t.Errorf("enforced set missing srp.user; got %#v", resp.Data.EnforcedCapabilities)
	}
	if _, ok := enforced["menu.role"]; ok {
		t.Errorf("enforced set should not contain reserved menu.role")
	}
	if list, ok := resp.Data.LegacyCapabilities[1001]; !ok || len(list) != 1 || list[0] != "menu.role" {
		t.Errorf("legacy_capabilities = %#v, want {1001:[menu.role]}", resp.Data.LegacyCapabilities)
	}
	// Stored policy must be readable with the legacy key dropped.
	if len(resp.Data.Policies) != 1 {
		t.Fatalf("policies len = %d, want 1", len(resp.Data.Policies))
	}
	caps, _ := resp.Data.Policies[0]["capabilities"].([]any)
	if len(caps) != 1 || caps[0] != "srp.user" {
		t.Errorf("stored capabilities = %#v, want [srp.user]", caps)
	}
}

func TestUpdateCorporationAccessPoliciesRecordsAudit(t *testing.T) {
	db := newSysConfigHandlerTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	if err := db.AutoMigrate(&model.AuditEvent{}); err != nil {
		t.Fatalf("migrate audit events: %v", err)
	}

	recorder := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("userID", uint(7))
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/system/basic-config/corporation-access-policies", bytes.NewBufferString(`{
		"version":2,
		"default_mode":"allow",
		"policies":[{"corporation_id":1001,"full_access":false,"capabilities":["srp.user"],"rules":{}}]
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	NewSysConfigHandler().UpdateCorporationAccessPolicies(ctx)

	var resp response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}

	var event model.AuditEvent
	if err := db.Where("action = ?", "corporation_access_policy_update").First(&event).Error; err != nil {
		t.Fatalf("expected audit event for corporation_access_policy_update: %v", err)
	}
	if event.Category != "config" {
		t.Fatalf("audit category = %q, want %q", event.Category, "config")
	}
	if event.ActorUserID != 7 {
		t.Fatalf("audit actor = %d, want 7", event.ActorUserID)
	}
	if event.ResourceID != model.SysConfigCorporationAccessPolicies {
		t.Fatalf("audit resource id = %q, want %q", event.ResourceID, model.SysConfigCorporationAccessPolicies)
	}
}
