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
		"default_mode":"deny",
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
	if resp.Data.DefaultMode != "deny" {
		t.Fatalf("default_mode = %q, want deny", resp.Data.DefaultMode)
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
		"default_mode":"deny",
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
