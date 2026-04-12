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
	"testing"

	"github.com/gin-gonic/gin"
)

type stubFuxiAdminService struct {
	getDirectory func() (*service.FuxiAdminDirectoryResponse, error)
	getConfig    func() (*model.FuxiAdminConfig, error)
	updateConfig func(*service.FuxiAdminUpdateConfigRequest) (*model.FuxiAdminConfig, error)
	listTiers    func() ([]model.FuxiAdminTier, error)
	createTier   func(*service.FuxiAdminCreateTierRequest) (*model.FuxiAdminTier, error)
	updateTier   func(uint, *service.FuxiAdminUpdateTierRequest) (*model.FuxiAdminTier, error)
	deleteTier   func(uint) error
	createAdmin  func(*service.FuxiAdminCreateAdminRequest) (*model.FuxiAdmin, error)
	updateAdmin  func(uint, *service.FuxiAdminUpdateAdminRequest) (*model.FuxiAdmin, error)
	deleteAdmin  func(uint) error
}

func (s stubFuxiAdminService) GetDirectory() (*service.FuxiAdminDirectoryResponse, error) {
	if s.getDirectory != nil {
		return s.getDirectory()
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

func decodeFuxiAdminHandlerResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.Response {
	t.Helper()

	var resp response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}
