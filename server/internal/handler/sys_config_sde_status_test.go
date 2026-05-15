package handler

import (
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubSDEStatusService struct {
	status    service.SDEStatus
	checkErr  error
	updateErr error
}

func (s stubSDEStatusService) GetStatus() (service.SDEStatus, error) {
	return s.status, nil
}

func (s stubSDEStatusService) CheckLatestVersion() (service.SDEStatus, error) {
	return s.status, s.checkErr
}

func (s stubSDEStatusService) TriggerManualUpdateWithStatus() (service.SDEStatus, error) {
	return s.status, s.updateErr
}

func TestGetSDEStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/sde-config/status", nil)

	handler := newSysConfigHandlerWithDeps(service.NewSysConfigService(), stubSDEStatusService{
		status: service.SDEStatus{
			CurrentVersion:   "v1",
			LatestVersion:    "v2",
			HasUpdate:        true,
			LastCheckAt:      100,
			LastCheckSuccess: true,
		},
	})
	handler.GetSDEStatus(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp response.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
}

func TestCheckSDEVersionReturnsBizErrorOnFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/system/sde-config/check", nil)

	handler := newSysConfigHandlerWithDeps(service.NewSysConfigService(), stubSDEStatusService{
		checkErr: errors.New("upstream down"),
	})
	handler.CheckSDEVersion(ctx)

	var resp response.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != response.CodeBizError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeBizError)
	}
}

func TestTriggerSDEUpdateReturnsBizErrorOnFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/system/sde-config/update", nil)

	handler := newSysConfigHandlerWithDeps(service.NewSysConfigService(), stubSDEStatusService{
		updateErr: errors.New("import failed"),
	})
	handler.TriggerSDEUpdate(ctx)

	var resp response.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != response.CodeBizError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeBizError)
	}
}
