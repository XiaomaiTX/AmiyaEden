package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"amiya-eden/internal/service"

	"github.com/gin-gonic/gin"
)

type fakeNewbroAdminSettingsService struct {
	getSettingsResp service.NewbroSettings
	updateResp      service.NewbroSettings
	updateErr       error
	updateCalls     int
	lastUpdate      service.NewbroSettings
}

func (f *fakeNewbroAdminSettingsService) GetSettings() service.NewbroSettings {
	return f.getSettingsResp
}

func (f *fakeNewbroAdminSettingsService) UpdateSettings(cfg service.NewbroSettings) (service.NewbroSettings, error) {
	f.updateCalls++
	f.lastUpdate = cfg
	if f.updateErr != nil {
		return service.NewbroSettings{}, f.updateErr
	}
	return f.updateResp, nil
}

func TestNewbroAdminGetSettingsReturnsRecruitFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/newbro/settings", nil)

	handler := &NewbroAdminHandler{
		settingsSvc: &fakeNewbroAdminSettingsService{
			getSettingsResp: service.NewbroSettings{
				MaxCharacterSP:          20_000_000,
				MultiCharacterSP:        10_000_000,
				MultiCharacterThreshold: 3,
				RefreshIntervalDays:     7,
				BonusRate:               20,
				RecruitQQURL:            "https://example.com/qq",
				RecruitRewardAmount:     50,
				RecruitCooldownDays:     90,
			},
		},
	}

	handler.GetSettings(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var resp struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data service.NewbroSettings `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 200 {
		t.Fatalf("expected response code 200, got %d", resp.Code)
	}
	if resp.Data.RecruitQQURL != "https://example.com/qq" {
		t.Fatalf("expected recruit_qq_url to round-trip, got %q", resp.Data.RecruitQQURL)
	}
	if resp.Data.RecruitRewardAmount != 50 {
		t.Fatalf("expected recruit_reward_amount 50, got %v", resp.Data.RecruitRewardAmount)
	}
	if resp.Data.RecruitCooldownDays != 90 {
		t.Fatalf("expected recruit_cooldown_days 90, got %d", resp.Data.RecruitCooldownDays)
	}
}

func TestNewbroAdminUpdateSettingsReturnsUpdatedRecruitFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	settingsSvc := &fakeNewbroAdminSettingsService{
		updateResp: service.NewbroSettings{
			MaxCharacterSP:          21_000_000,
			MultiCharacterSP:        11_000_000,
			MultiCharacterThreshold: 4,
			RefreshIntervalDays:     9,
			BonusRate:               0,
			RecruitQQURL:            "https://example.com/new-qq",
			RecruitRewardAmount:     0,
			RecruitCooldownDays:     120,
		},
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/system/newbro/settings",
		bytes.NewBufferString(`{"max_character_sp":21000000,"multi_character_sp":11000000,"multi_character_threshold":4,"refresh_interval_days":9,"bonus_rate":0,"recruit_qq_url":"https://example.com/new-qq","recruit_reward_amount":0,"recruit_cooldown_days":120}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := &NewbroAdminHandler{settingsSvc: settingsSvc}

	handler.UpdateSettings(ctx)

	if settingsSvc.updateCalls != 1 {
		t.Fatalf("expected one update call, got %d", settingsSvc.updateCalls)
	}
	if settingsSvc.lastUpdate.BonusRate != 0 {
		t.Fatalf("expected service to receive bonus_rate 0, got %v", settingsSvc.lastUpdate.BonusRate)
	}
	if settingsSvc.lastUpdate.RecruitRewardAmount != 0 {
		t.Fatalf("expected service to receive recruit_reward_amount 0, got %v", settingsSvc.lastUpdate.RecruitRewardAmount)
	}
	if settingsSvc.lastUpdate.RecruitQQURL != "https://example.com/new-qq" {
		t.Fatalf("expected service to receive recruit_qq_url, got %q", settingsSvc.lastUpdate.RecruitQQURL)
	}
	if settingsSvc.lastUpdate.RecruitCooldownDays != 120 {
		t.Fatalf("expected service to receive recruit_cooldown_days 120, got %d", settingsSvc.lastUpdate.RecruitCooldownDays)
	}

	var resp struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data service.NewbroSettings `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 200 {
		t.Fatalf("expected response code 200, got %d", resp.Code)
	}
	if resp.Data.RecruitRewardAmount != 0 {
		t.Fatalf("expected response recruit_reward_amount 0, got %v", resp.Data.RecruitRewardAmount)
	}
	if resp.Data.RecruitCooldownDays != 120 {
		t.Fatalf("expected response recruit_cooldown_days 120, got %d", resp.Data.RecruitCooldownDays)
	}
}

func TestNewbroAdminUpdateSettingsReturnsBusinessError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/system/newbro/settings",
		bytes.NewBufferString(`{"max_character_sp":21000000,"multi_character_sp":11000000,"multi_character_threshold":4,"refresh_interval_days":9,"bonus_rate":0,"recruit_qq_url":"https://example.com/new-qq","recruit_reward_amount":0,"recruit_cooldown_days":120}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := &NewbroAdminHandler{
		settingsSvc: &fakeNewbroAdminSettingsService{updateErr: errors.New("write failed")},
	}

	handler.UpdateSettings(ctx)

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 500 {
		t.Fatalf("expected response code 500, got %d", resp.Code)
	}
	if resp.Msg != "write failed" {
		t.Fatalf("expected business error message, got %q", resp.Msg)
	}
}
