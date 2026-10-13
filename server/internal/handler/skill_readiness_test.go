package handler

import (
	"amiya-eden/internal/service"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSkillReadinessHandlerRejectsMalformedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &SkillReadinessHandler{}
	router := gin.New()
	router.POST("/query", h.Query)
	router.POST("/users/:user_id/characters", h.UserCharacters)
	router.POST("/characters/:character_id/skills", h.CharacterSkills)
	for _, tc := range []struct{ path, body string }{
		{"/query", `{"plan_id":-1}`}, {"/query", `{"plan_id":"bad"}`}, {"/query", `{`},
		{"/users/0/characters", `{}`}, {"/users/-1/characters", `{}`}, {"/characters/0/skills", `{}`}, {"/characters/9223372036854775808/skills", `{}`},
	} {
		req := httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var body struct{ Code int }
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Code != 400 {
			t.Fatalf("%s = %s", tc.path, rec.Body.String())
		}
	}
}

func TestSkillReadinessErrorsDoNotExposeInternals(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	readinessError(ctx, errors.New("secret postgres password"))
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatal("internal error leaked")
	}
	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	readinessError(ctx, service.ErrSkillReadinessPlan)
	var body struct{ Code int }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Code != 500 {
		t.Fatal(rec.Body.String())
	}
}
