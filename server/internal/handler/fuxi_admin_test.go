package handler

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubFuxiAdminService struct {
	getDirectory       func() (*service.FuxiAdminDirectoryResponse, error)
	getManageDirectory func() (*service.FuxiAdminManageDirectoryResponse, error)
	getManageAdmin     func(uint) (*service.FuxiAdminManageAdmin, error)
	getConfig          func() (*model.FuxiAdminConfig, error)
	updateConfig       func(*service.FuxiAdminUpdateConfigRequest) (*model.FuxiAdminConfig, error)
	listTiers          func() ([]model.FuxiAdminTier, error)
	createTier         func(*service.FuxiAdminCreateTierRequest) (*model.FuxiAdminTier, error)
	updateTier         func(uint, *service.FuxiAdminUpdateTierRequest) (*model.FuxiAdminTier, error)
	deleteTier         func(uint) error
	createAdmin        func(*service.FuxiAdminCreateAdminRequest) (*model.FuxiAdmin, error)
	updateAdmin        func(uint, *service.FuxiAdminUpdateAdminRequest) (*model.FuxiAdmin, error)
	deleteAdmin        func(uint) error
}

func (s stubFuxiAdminService) GetDirectory() (*service.FuxiAdminDirectoryResponse, error) {
	if s.getDirectory != nil {
		return s.getDirectory()
	}
	return nil, nil
}

func (s stubFuxiAdminService) GetManageDirectory() (*service.FuxiAdminManageDirectoryResponse, error) {
	if s.getManageDirectory != nil {
		return s.getManageDirectory()
	}
	return nil, nil
}

func (s stubFuxiAdminService) GetManageAdmin(id uint) (*service.FuxiAdminManageAdmin, error) {
	if s.getManageAdmin != nil {
		return s.getManageAdmin(id)
	}
	return nil, nil
}

func (s stubFuxiAdminService) GetConfig() (*model.FuxiAdminConfig, error) {
	if s.getConfig != nil {
		return s.getConfig()
	}
	return nil, nil
}

func (s stubFuxiAdminService) UpdateConfig(req *service.FuxiAdminUpdateConfigRequest) (*model.FuxiAdminConfig, error) {
	if s.updateConfig != nil {
		return s.updateConfig(req)
	}
	return nil, nil
}

func (s stubFuxiAdminService) ListTiers() ([]model.FuxiAdminTier, error) {
	if s.listTiers != nil {
		return s.listTiers()
	}
	return nil, nil
}

func (s stubFuxiAdminService) CreateTier(req *service.FuxiAdminCreateTierRequest) (*model.FuxiAdminTier, error) {
	if s.createTier != nil {
		return s.createTier(req)
	}
	return nil, nil
}

func (s stubFuxiAdminService) UpdateTier(id uint, req *service.FuxiAdminUpdateTierRequest) (*model.FuxiAdminTier, error) {
	if s.updateTier != nil {
		return s.updateTier(id, req)
	}
	return nil, nil
}

func (s stubFuxiAdminService) DeleteTier(id uint) error {
	if s.deleteTier != nil {
		return s.deleteTier(id)
	}
	return nil
}

func (s stubFuxiAdminService) CreateAdmin(req *service.FuxiAdminCreateAdminRequest) (*model.FuxiAdmin, error) {
	if s.createAdmin != nil {
		return s.createAdmin(req)
	}
	return nil, nil
}

func (s stubFuxiAdminService) UpdateAdmin(id uint, req *service.FuxiAdminUpdateAdminRequest) (*model.FuxiAdmin, error) {
	if s.updateAdmin != nil {
		return s.updateAdmin(id, req)
	}
	return nil, nil
}

func (s stubFuxiAdminService) DeleteAdmin(id uint) error {
	if s.deleteAdmin != nil {
		return s.deleteAdmin(id)
	}
	return nil
}

func TestFuxiAdminHandlerGetDirectoryReturnsSafeErrorMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/fuxi-admins", nil)

	h := &FuxiAdminHandler{svc: stubFuxiAdminService{
		getDirectory: func() (*service.FuxiAdminDirectoryResponse, error) {
			return nil, errors.New("sql: database is closed")
		},
	}}

	h.GetDirectory(ctx)

	resp := decodeFuxiAdminHandlerResponse(t, recorder)
	if resp.Code != response.CodeBizError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeBizError)
	}
	if resp.Msg != "获取伏羲管理名录失败" {
		t.Fatalf("response msg = %q, want transport-safe error", resp.Msg)
	}
}

func TestFuxiAdminHandlerUpdateConfigPreservesUserVisibleErrorMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/system/fuxi-admins/config",
		bytes.NewBufferString(`{"page_background_color":"rgba(16,36,58,0.5)"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	h := &FuxiAdminHandler{svc: stubFuxiAdminService{
		updateConfig: func(*service.FuxiAdminUpdateConfigRequest) (*model.FuxiAdminConfig, error) {
			return nil, service.NewUserVisibleError("页面背景色必须是十六进制颜色值")
		},
	}}

	h.UpdateConfig(ctx)

	resp := decodeFuxiAdminHandlerResponse(t, recorder)
	if resp.Code != response.CodeBizError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeBizError)
	}
	if resp.Msg != "页面背景色必须是十六进制颜色值" {
		t.Fatalf("response msg = %q, want user-visible validation error", resp.Msg)
	}
}

func TestFuxiAdminHandlerGetDirectoryDoesNotExposeWelfareDeliveryOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/fuxi-admins", nil)

	h := &FuxiAdminHandler{svc: stubFuxiAdminService{
		getDirectory: func() (*service.FuxiAdminDirectoryResponse, error) {
			return &service.FuxiAdminDirectoryResponse{
				Config: model.DefaultFuxiAdminConfig(),
				Tiers: []service.FuxiAdminTierWithAdmins{{
					FuxiAdminTier: model.FuxiAdminTier{ID: 1, Name: "Ops"},
					Admins:        []model.FuxiAdmin{{BaseModel: model.BaseModel{ID: 1}, TierID: 1, Nickname: "Alpha", WelfareDeliveryOffset: 7}},
				}},
			}, nil
		},
	}}

	h.GetDirectory(ctx)

	if strings.Contains(recorder.Body.String(), "welfare_delivery_offset") {
		t.Fatalf("expected public directory response to omit welfare_delivery_offset, body = %s", recorder.Body.String())
	}
}

func TestFuxiAdminHandlerGetManageDirectoryReturnsManageFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/fuxi-admins/manage-directory", nil)

	h := &FuxiAdminHandler{svc: stubFuxiAdminService{
		getManageDirectory: func() (*service.FuxiAdminManageDirectoryResponse, error) {
			return &service.FuxiAdminManageDirectoryResponse{
				Config: model.DefaultFuxiAdminConfig(),
				Tiers: []service.FuxiAdminManageTierWithAdmins{{
					FuxiAdminTier: model.FuxiAdminTier{ID: 1, Name: "Ops"},
					Admins: []service.FuxiAdminManageAdmin{{
						FuxiAdmin:             model.FuxiAdmin{BaseModel: model.BaseModel{ID: 1}, TierID: 1, Nickname: "Alpha"},
						WelfareDeliveryOffset: 7,
						FleetLedCount:         2,
						WelfareDeliveryCount:  5,
					}},
				}},
			}, nil
		},
	}}

	h.GetManageDirectory(ctx)

	body := recorder.Body.String()
	if !strings.Contains(body, "welfare_delivery_offset") || !strings.Contains(body, "fleet_led_count") || !strings.Contains(body, "welfare_delivery_count") {
		t.Fatalf("expected manage directory response to include count fields, body = %s", body)
	}
}

func TestFuxiAdminHandlerCreateAdminReturnsManageFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/system/fuxi-admins",
		bytes.NewBufferString(`{"tier_id":1,"nickname":"Alpha"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	created := model.FuxiAdmin{BaseModel: model.BaseModel{ID: 7}, TierID: 1, Nickname: "Alpha"}
	var gotManageAdminID uint
	h := &FuxiAdminHandler{svc: stubFuxiAdminService{
		createAdmin: func(*service.FuxiAdminCreateAdminRequest) (*model.FuxiAdmin, error) {
			return &created, nil
		},
		getManageAdmin: func(id uint) (*service.FuxiAdminManageAdmin, error) {
			gotManageAdminID = id
			return &service.FuxiAdminManageAdmin{
				FuxiAdmin:             created,
				WelfareDeliveryOffset: 2,
				FleetLedCount:         3,
				WelfareDeliveryCount:  5,
			}, nil
		},
	}}

	h.CreateAdmin(ctx)

	if gotManageAdminID != created.ID {
		t.Fatalf("expected GetManageAdmin to be called with %d, got %d", created.ID, gotManageAdminID)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "fleet_led_count") || !strings.Contains(body, "welfare_delivery_count") {
		t.Fatalf("expected create response to include manage fields, body = %s", body)
	}
}

func TestFuxiAdminHandlerCreateAdminFallsBackToBaseManagePayloadWhenEnrichmentFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/system/fuxi-admins",
		bytes.NewBufferString(`{"tier_id":1,"nickname":"Alpha"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	created := model.FuxiAdmin{
		BaseModel:             model.BaseModel{ID: 8},
		TierID:                1,
		Nickname:              "Alpha",
		WelfareDeliveryOffset: 4,
	}
	h := &FuxiAdminHandler{svc: stubFuxiAdminService{
		createAdmin: func(*service.FuxiAdminCreateAdminRequest) (*model.FuxiAdmin, error) {
			return &created, nil
		},
		getManageAdmin: func(uint) (*service.FuxiAdminManageAdmin, error) {
			return nil, errors.New("count query failed")
		},
	}}

	h.CreateAdmin(ctx)

	resp := decodeFuxiAdminHandlerResponse(t, recorder)
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"fleet_led_count":0`) || !strings.Contains(body, `"welfare_delivery_count":4`) {
		t.Fatalf("expected fallback manage payload in body, got %s", body)
	}
}

func TestFuxiAdminHandlerUpdateAdminReturnsManageFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/system/fuxi-admins/7",
		bytes.NewBufferString(`{"nickname":"Alpha"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "7"}}
	ctx.Set("roles", []string{model.RoleAdmin})

	updated := model.FuxiAdmin{BaseModel: model.BaseModel{ID: 7}, TierID: 1, Nickname: "Alpha"}
	var gotManageAdminID uint
	h := &FuxiAdminHandler{svc: stubFuxiAdminService{
		updateAdmin: func(uint, *service.FuxiAdminUpdateAdminRequest) (*model.FuxiAdmin, error) {
			return &updated, nil
		},
		getManageAdmin: func(id uint) (*service.FuxiAdminManageAdmin, error) {
			gotManageAdminID = id
			return &service.FuxiAdminManageAdmin{
				FuxiAdmin:             updated,
				WelfareDeliveryOffset: 1,
				FleetLedCount:         4,
				WelfareDeliveryCount:  9,
			}, nil
		},
	}}

	h.UpdateAdmin(ctx)

	if gotManageAdminID != updated.ID {
		t.Fatalf("expected GetManageAdmin to be called with %d, got %d", updated.ID, gotManageAdminID)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "fleet_led_count") || !strings.Contains(body, "welfare_delivery_count") {
		t.Fatalf("expected update response to include manage fields, body = %s", body)
	}
}

func TestFuxiAdminHandlerUpdateAdminFallsBackToBaseManagePayloadWhenEnrichmentFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/system/fuxi-admins/8",
		bytes.NewBufferString(`{"nickname":"Alpha"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "8"}}
	ctx.Set("roles", []string{model.RoleAdmin})

	updated := model.FuxiAdmin{
		BaseModel:             model.BaseModel{ID: 8},
		TierID:                1,
		Nickname:              "Alpha",
		WelfareDeliveryOffset: 3,
	}
	h := &FuxiAdminHandler{svc: stubFuxiAdminService{
		updateAdmin: func(uint, *service.FuxiAdminUpdateAdminRequest) (*model.FuxiAdmin, error) {
			return &updated, nil
		},
		getManageAdmin: func(uint) (*service.FuxiAdminManageAdmin, error) {
			return nil, errors.New("count query failed")
		},
	}}

	h.UpdateAdmin(ctx)

	resp := decodeFuxiAdminHandlerResponse(t, recorder)
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"fleet_led_count":0`) || !strings.Contains(body, `"welfare_delivery_count":3`) {
		t.Fatalf("expected fallback manage payload in body, got %s", body)
	}
}

func TestFuxiAdminHandlerUpdateAdminRejectsWelfareDeliveryOffsetForNonSuperAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/system/fuxi-admins/1",
		bytes.NewBufferString(`{"welfare_delivery_offset":5}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	ctx.Set("roles", []string{model.RoleAdmin})

	called := false
	h := &FuxiAdminHandler{svc: stubFuxiAdminService{
		updateAdmin: func(uint, *service.FuxiAdminUpdateAdminRequest) (*model.FuxiAdmin, error) {
			called = true
			return &model.FuxiAdmin{}, nil
		},
	}}

	h.UpdateAdmin(ctx)

	resp := decodeFuxiAdminHandlerResponse(t, recorder)
	if resp.Code != response.CodeForbidden {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeForbidden)
	}
	if resp.Msg != "仅超级管理员可修改福利发放次数偏移" {
		t.Fatalf("response msg = %q, want super-admin-only message", resp.Msg)
	}
	if called {
		t.Fatal("expected UpdateAdmin service not to be called")
	}
}

func TestFuxiAdminHandlerUpdateAdminAllowsWelfareDeliveryOffsetForSuperAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/system/fuxi-admins/1",
		bytes.NewBufferString(`{"welfare_delivery_offset":5}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	ctx.Set("roles", []string{model.RoleSuperAdmin})

	called := false
	h := &FuxiAdminHandler{svc: stubFuxiAdminService{
		updateAdmin: func(id uint, req *service.FuxiAdminUpdateAdminRequest) (*model.FuxiAdmin, error) {
			called = true
			if req.WelfareDeliveryOffset == nil || *req.WelfareDeliveryOffset != 5 {
				t.Fatalf("expected welfare delivery offset 5, got %+v", req.WelfareDeliveryOffset)
			}
			return &model.FuxiAdmin{BaseModel: model.BaseModel{ID: id}, WelfareDeliveryOffset: *req.WelfareDeliveryOffset}, nil
		},
	}}

	h.UpdateAdmin(ctx)

	resp := decodeFuxiAdminHandlerResponse(t, recorder)
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
	if !called {
		t.Fatal("expected UpdateAdmin service to be called")
	}
}

func decodeFuxiAdminHandlerResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.Response {
	t.Helper()

	var resp response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}
