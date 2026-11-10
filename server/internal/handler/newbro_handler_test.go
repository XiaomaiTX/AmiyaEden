package handler

import (
	"bytes"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseOptionalUintQueryParamRejectsInvalidInput(t *testing.T) {
	if _, err := parseOptionalUintQueryParam("player_user_id", "not-a-number"); err == nil {
		t.Fatal("expected invalid player_user_id to return an error")
	}
}

func TestParseOptionalUintQueryParamRejectsOverflow(t *testing.T) {
	overflow := strconv.FormatUint(uint64(math.MaxUint32)+1, 10)
	if _, err := parseOptionalUintQueryParam("player_user_id", overflow); err == nil {
		t.Fatal("expected overflow player_user_id to return an error")
	}
}

func TestUpdateNewbroRecruitSettingsRequestAllowsZeroRewardAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/newbro/recruit-settings", bytes.NewBufferString(`{
		"recruit_qq_url": "https://example.com/qq",
		"recruit_reward_amount": 0,
		"recruit_cooldown_days": 90
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	var req UpdateNewbroRecruitSettingsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		t.Fatalf("expected zero recruit_reward_amount to bind successfully, got %v", err)
	}
	if req.RecruitRewardAmount == nil || *req.RecruitRewardAmount != 0 {
		t.Fatalf("expected recruit_reward_amount pointer to preserve explicit zero, got %#v", req.RecruitRewardAmount)
	}
	if req.RecruitCooldownDays != 90 {
		t.Fatalf("expected recruit_cooldown_days 90, got %d", req.RecruitCooldownDays)
	}
}
