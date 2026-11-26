package handler

import (
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSkillReadinessHandlerRejectsMalformedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newMigratedOperationAdapter(migratedOperations{SkillReadinessHandler: NewSkillReadinessHandler()})
	router := gin.New()
	registerMigratedOperationRoutes(router, h, func(c *gin.Context, _ string, _ error, _ int) {
		c.Abort()
		response.Fail(c, response.CodeParamError, "查询参数无效")
	})
	for _, tc := range []struct{ path, body string }{
		{"/query", `{"plan_id":-1}`}, {"/query", `{"plan_id":"bad"}`}, {"/query", `{`},
		{"/users/0/characters", `{}`}, {"/users/-1/characters", `{}`}, {"/users/18446744073709551616/characters", `{}`},
		{"/characters/0/skills", `{}`}, {"/characters/9223372036854775808/skills", `{}`},
	} {
		req := httptest.NewRequest("POST", "/api/v1/dashboard/skill-readiness"+tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var body struct {
			Code int
			Msg  string
			Data json.RawMessage
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != 200 || body.Code != 400 || body.Msg != "查询参数无效" || string(body.Data) != "null" {
			t.Fatalf("%s = HTTP %d %s", tc.path, rec.Code, rec.Body.String())
		}
	}
}

func TestSkillReadinessErrorsDoNotExposeInternals(t *testing.T) {
	got := readinessBusinessError(errors.New("secret postgres password"))
	if got.Code != response.CodeBizError || strings.Contains(got.Msg, "secret") {
		t.Fatalf("unexpected internal error response: %+v", got)
	}
	if got := readinessBusinessError(service.ErrSkillReadinessPlan); got.Code != response.CodeBizError {
		t.Fatalf("unexpected plan error response: %+v", got)
	}
}
