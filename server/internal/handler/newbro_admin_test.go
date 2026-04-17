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
	getSupportResp     service.NewbroSupportSettings
	getRecruitResp     service.NewbroRecruitSettings
	updateSupportResp  service.NewbroSupportSettings
	updateRecruitResp  service.NewbroRecruitSettings
	updateErr          error
	supportUpdateCalls int
	recruitUpdateCalls int
	lastSupportUpdate  service.NewbroSupportSettings
	lastRecruitUpdate  service.NewbroRecruitSettings
}

func (f *fakeNewbroAdminSettingsService) GetSupportSettings() service.NewbroSupportSettings {
	return f.getSupportResp
}

func (f *fakeNewbroAdminSettingsService) UpdateSupportSettings(cfg service.NewbroSupportSettings) (service.NewbroSupportSettings, error) {
	f.supportUpdateCalls++
	f.lastSupportUpdate = cfg
	if f.updateErr != nil {
		return service.NewbroSupportSettings{}, f.updateErr
	}
	return f.updateSupportResp, nil
}

func (f *fakeNewbroAdminSettingsService) GetRecruitSettings() service.NewbroRecruitSettings {
	return f.getRecruitResp
}

func (f *fakeNewbroAdminSettingsService) UpdateRecruitSettings(cfg service.NewbroRecruitSettings) (service.NewbroRecruitSettings, error) {
	f.recruitUpdateCalls++
	f.lastRecruitUpdate = cfg
	if f.updateErr != nil {
		return service.NewbroRecruitSettings{}, f.updateErr
	}
	return f.updateRecruitResp, nil
}

func TestNewbroAdminGetSupportSettingsReturnsSupportFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/newbro/support-settings", nil)

	handler := &NewbroAdminHandler{
		settingsSvc: &fakeNewbroAdminSettingsService{
			getSupportResp: service.NewbroSupportSettings{
				MaxCharacterSP:          20_000_000,
				MultiCharacterSP:        10_000_000,
				MultiCharacterThreshold: 3,
				RefreshIntervalDays:     7,
				BonusRate:               20,
			},
		},
	}

	handler.GetSupportSettings(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var resp struct {
		Code int                           `json:"code"`
		Msg  string                        `json:"msg"`
		Data service.NewbroSupportSettings `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 200 {
		t.Fatalf("expected response code 200, got %d", resp.Code)
	}
	if resp.Data.MaxCharacterSP != 20_000_000 {
		t.Fatalf("expected max_character_sp 20000000, got %d", resp.Data.MaxCharacterSP)
	}
	if resp.Data.BonusRate != 20 {
		t.Fatalf("expected bonus_rate 20, got %v", resp.Data.BonusRate)
	}
	if resp.Data.RefreshIntervalDays != 7 {
		t.Fatalf("expected refresh_interval_days 7, got %d", resp.Data.RefreshIntervalDays)
	}
}

func TestNewbroAdminGetRecruitSettingsReturnsRecruitFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/newbro/recruit-settings", nil)

	handler := &NewbroAdminHandler{
		settingsSvc: &fakeNewbroAdminSettingsService{
			getRecruitResp: service.NewbroRecruitSettings{
				RecruitQQURL:        "https://example.com/qq",
				RecruitRewardAmount: 50,
				RecruitCooldownDays: 90,
			},
		},
	}

	handler.GetRecruitSettings(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var resp struct {
		Code int                           `json:"code"`
		Msg  string                        `json:"msg"`
		Data service.NewbroRecruitSettings `json:"data"`
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

func TestNewbroAdminUpdateSupportSettingsReturnsUpdatedSupportFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	settingsSvc := &fakeNewbroAdminSettingsService{
		updateSupportResp: service.NewbroSupportSettings{
			MaxCharacterSP:          21_000_000,
			MultiCharacterSP:        11_000_000,
			MultiCharacterThreshold: 4,
			RefreshIntervalDays:     9,
			BonusRate:               0,
		},
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/system/newbro/support-settings",
		bytes.NewBufferString(`{"max_character_sp":21000000,"multi_character_sp":11000000,"multi_character_threshold":4,"refresh_interval_days":9,"bonus_rate":0}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := &NewbroAdminHandler{settingsSvc: settingsSvc}

	handler.UpdateSupportSettings(ctx)

	if settingsSvc.supportUpdateCalls != 1 {
		t.Fatalf("expected one support update call, got %d", settingsSvc.supportUpdateCalls)
	}
	if settingsSvc.lastSupportUpdate.BonusRate != 0 {
		t.Fatalf("expected service to receive bonus_rate 0, got %v", settingsSvc.lastSupportUpdate.BonusRate)
	}
	if settingsSvc.lastSupportUpdate.RefreshIntervalDays != 9 {
		t.Fatalf("expected service to receive refresh_interval_days 9, got %d", settingsSvc.lastSupportUpdate.RefreshIntervalDays)
	}
	if settingsSvc.lastSupportUpdate.MultiCharacterThreshold != 4 {
		t.Fatalf("expected service to receive multi_character_threshold 4, got %d", settingsSvc.lastSupportUpdate.MultiCharacterThreshold)
	}

	var resp struct {
		Code int                           `json:"code"`
		Msg  string                        `json:"msg"`
		Data service.NewbroSupportSettings `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 200 {
		t.Fatalf("expected response code 200, got %d", resp.Code)
	}
	if resp.Data.BonusRate != 0 {
		t.Fatalf("expected response bonus_rate 0, got %v", resp.Data.BonusRate)
	}
	if resp.Data.RefreshIntervalDays != 9 {
		t.Fatalf("expected response refresh_interval_days 9, got %d", resp.Data.RefreshIntervalDays)
	}
}

func TestNewbroAdminUpdateRecruitSettingsReturnsUpdatedRecruitFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	settingsSvc := &fakeNewbroAdminSettingsService{
		updateRecruitResp: service.NewbroRecruitSettings{
			RecruitQQURL:        "https://example.com/new-qq",
			RecruitRewardAmount: 0,
			RecruitCooldownDays: 120,
		},
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/system/newbro/recruit-settings",
		bytes.NewBufferString(`{"recruit_qq_url":"https://example.com/new-qq","recruit_reward_amount":0,"recruit_cooldown_days":120}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := &NewbroAdminHandler{settingsSvc: settingsSvc}

	handler.UpdateRecruitSettings(ctx)

	if settingsSvc.recruitUpdateCalls != 1 {
		t.Fatalf("expected one recruit update call, got %d", settingsSvc.recruitUpdateCalls)
	}
	if settingsSvc.lastRecruitUpdate.RecruitRewardAmount != 0 {
		t.Fatalf("expected service to receive recruit_reward_amount 0, got %v", settingsSvc.lastRecruitUpdate.RecruitRewardAmount)
	}
	if settingsSvc.lastRecruitUpdate.RecruitQQURL != "https://example.com/new-qq" {
		t.Fatalf("expected service to receive recruit_qq_url, got %q", settingsSvc.lastRecruitUpdate.RecruitQQURL)
	}
	if settingsSvc.lastRecruitUpdate.RecruitCooldownDays != 120 {
		t.Fatalf("expected service to receive recruit_cooldown_days 120, got %d", settingsSvc.lastRecruitUpdate.RecruitCooldownDays)
	}

	var resp struct {
		Code int                           `json:"code"`
		Msg  string                        `json:"msg"`
		Data service.NewbroRecruitSettings `json:"data"`
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

func TestNewbroAdminUpdateRecruitSettingsReturnsBusinessError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/system/newbro/recruit-settings",
		bytes.NewBufferString(`{"recruit_qq_url":"https://example.com/new-qq","recruit_reward_amount":0,"recruit_cooldown_days":120}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := &NewbroAdminHandler{
		settingsSvc: &fakeNewbroAdminSettingsService{updateErr: errors.New("write failed")},
	}

	handler.UpdateRecruitSettings(ctx)

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
