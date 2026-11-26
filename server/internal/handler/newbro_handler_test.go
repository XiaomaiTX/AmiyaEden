package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUpdateNewbroRecruitSettingsRequestAllowsZeroRewardAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &strictRecruitSettingsStub{}
	adapter := apicontract.NewStrictHandler(strictNewbroTestOperations{
		NewbroAdminHandler: &NewbroAdminHandler{settingsSvc: stub},
	})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", uint(42)) })
	registerMigratedOperationRoutes(r, adapter, migratedOperationParamError)

	got := requestStrictNewbro(t, r, http.MethodPut, "/api/v1/system/newbro/recruit-settings", `{
		"recruit_qq_url": "https://example.com/qq",
		"recruit_reward_amount": 0,
		"recruit_cooldown_days": 90
	}`)
	var data apicontract.NewbroRecruitSettings
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatal(err)
	}
	if got.Code != response.CodeOK || data.RecruitQQURL != "https://example.com/qq" || data.RecruitRewardAmount != 0 || data.RecruitCooldownDays != 90 || stub.operator != 42 {
		t.Fatalf("unexpected settings response: code=%d data=%s actor=%d", got.Code, got.Data, stub.operator)
	}
}

type strictNewbroTestOperations struct {
	migratedOperationFallback
	*NewbroAdminHandler
	*CaptainRefundHandler
}

type newbroStrictTestResult struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func requestStrictNewbro(t *testing.T, r *gin.Engine, method, path, body string) newbroStrictTestResult {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	// ShouldBindJSON did not depend on the incoming Content-Type header.
	req.Header.Set("Content-Type", "text/plain")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("HTTP %d %s; want 200", rec.Code, rec.Body)
	}
	var result newbroStrictTestResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

type strictRecruitSettingsStub struct {
	settings service.NewbroRecruitSettings
	operator uint
}

func (s *strictRecruitSettingsStub) GetRecruitSettings() service.NewbroRecruitSettings {
	return s.settings
}

func (s *strictRecruitSettingsStub) UpdateRecruitSettingsByOperator(settings service.NewbroRecruitSettings, operator uint) (service.NewbroRecruitSettings, error) {
	s.settings, s.operator = settings, operator
	return settings, nil
}
