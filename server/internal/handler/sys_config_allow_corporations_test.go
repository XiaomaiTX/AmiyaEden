package handler

import (
	"amiya-eden/config"
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/utils"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetAllowCorporationsReturnsNames(t *testing.T) {
	db := newSysConfigHandlerTestDB(t)
	originalDB := global.DB
	originalConfig := global.Config
	global.DB = db
	defer func() {
		global.DB = originalDB
		global.Config = originalConfig
		utils.InvalidateAllowCorporationsCache()
	}()

	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigAllowCorporations,
		Value: `[98185110,98000001]`,
		Desc:  "test",
	}).Error; err != nil {
		t.Fatalf("create allow corporations config: %v", err)
	}
	utils.InvalidateAllowCorporationsCache()

	esiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/universe/names" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":98185110,"name":"Fuxi Legion"},
			{"id":98000001,"name":"Test Corp"}
		]`))
	}))
	defer esiServer.Close()

	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Config.EveSSO.ESIBaseURL = esiServer.URL
	global.Config.EveSSO.ESIAPIPrefix = ""

	recorder := httptest.NewRecorder()
	ctx, _ := newJSONContext(http.MethodGet, "/api/v1/system/basic-config/allow-corporations", nil, recorder)

	NewSysConfigHandler().GetAllowCorporations(ctx)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			AllowCorporations []int64 `json:"allow_corporations"`
			Corporations      []struct {
				CorporationID   int64  `json:"corporation_id"`
				CorporationName string `json:"corporation_name"`
			} `json:"corporations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 200 {
		t.Fatalf("response code = %d, want 200", resp.Code)
	}
	if len(resp.Data.AllowCorporations) != 2 {
		t.Fatalf("allow_corporations length = %d, want 2", len(resp.Data.AllowCorporations))
	}
	if len(resp.Data.Corporations) != 2 {
		t.Fatalf("corporations length = %d, want 2", len(resp.Data.Corporations))
	}
	if resp.Data.Corporations[0].CorporationName != "Fuxi Legion" {
		t.Fatalf("first corporation name = %q, want %q", resp.Data.Corporations[0].CorporationName, "Fuxi Legion")
	}
}

func TestGetAllowCorporationsFallsBackToIDWhenNameLookupFails(t *testing.T) {
	db := newSysConfigHandlerTestDB(t)
	originalDB := global.DB
	originalConfig := global.Config
	global.DB = db
	defer func() {
		global.DB = originalDB
		global.Config = originalConfig
		utils.InvalidateAllowCorporationsCache()
	}()

	if err := db.Where("key = ?", model.SysConfigAllowCorporations).Delete(&model.SystemConfig{}).Error; err != nil {
		t.Fatalf("delete allow corporations config: %v", err)
	}
	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigAllowCorporations,
		Value: `[98185110]`,
		Desc:  "test",
	}).Error; err != nil {
		t.Fatalf("create allow corporations config: %v", err)
	}
	utils.InvalidateAllowCorporationsCache()

	esiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer esiServer.Close()

	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Config.EveSSO.ESIBaseURL = esiServer.URL
	global.Config.EveSSO.ESIAPIPrefix = ""

	recorder := httptest.NewRecorder()
	ctx, _ := newJSONContext(http.MethodGet, "/api/v1/system/basic-config/allow-corporations", nil, recorder)

	NewSysConfigHandler().GetAllowCorporations(ctx)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Corporations []struct {
				CorporationID   int64  `json:"corporation_id"`
				CorporationName string `json:"corporation_name"`
			} `json:"corporations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 200 {
		t.Fatalf("response code = %d, want 200", resp.Code)
	}
	if len(resp.Data.Corporations) != 1 {
		t.Fatalf("corporations length = %d, want 1", len(resp.Data.Corporations))
	}
	if resp.Data.Corporations[0].CorporationName != "" {
		t.Fatalf("corporation_name = %q, want empty", resp.Data.Corporations[0].CorporationName)
	}
}

func newJSONContext(method, target string, body []byte, recorder *httptest.ResponseRecorder) (*gin.Context, *httptest.ResponseRecorder) {
	if recorder == nil {
		recorder = httptest.NewRecorder()
	}
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(recorder)
	var reqBodyReader *bytes.Reader
	if body == nil {
		reqBodyReader = bytes.NewReader([]byte{})
	} else {
		reqBodyReader = bytes.NewReader(body)
	}
	ctx.Request = httptest.NewRequest(method, target, reqBodyReader)
	ctx.Request.Header.Set("Content-Type", "application/json")
	return ctx, recorder
}
