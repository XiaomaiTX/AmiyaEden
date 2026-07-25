package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fakeWebhookConfigStore struct {
	setManyCalls int
	setManyItems []repository.SysConfigUpsertItem
	setManyErr   error
}

func (f *fakeWebhookConfigStore) Get(_ string, defaultVal string) (string, error) {
	return defaultVal, nil
}

func (f *fakeWebhookConfigStore) GetBool(_ string, defaultVal bool) bool {
	return defaultVal
}

func (f *fakeWebhookConfigStore) SetMany(items []repository.SysConfigUpsertItem) error {
	f.setManyCalls++
	f.setManyItems = append([]repository.SysConfigUpsertItem(nil), items...)
	return f.setManyErr
}

func TestWebhookSetConfigPersistsSingleBatch(t *testing.T) {
	store := &fakeWebhookConfigStore{}
	svc := &WebhookService{repo: store, http: &http.Client{}}

	err := svc.SetConfig(&WebhookConfig{
		URL:           "https://discord.com/api/webhooks/1/token",
		Enabled:       true,
		Type:          "discord",
		FleetTemplate: defaultFleetTemplate,
		OBTargetType:  "group",
		OBTargetID:    42,
		OBToken:       "token",
	})
	if err != nil {
		t.Fatalf("expected config update to succeed, got %v", err)
	}
	if store.setManyCalls != 1 {
		t.Fatalf("expected exactly one batch write, got %d", store.setManyCalls)
	}
	if len(store.setManyItems) != 8 {
		t.Fatalf("expected 8 config items, got %d", len(store.setManyItems))
	}

	wantKeys := []string{
		model.SysConfigWebhookURL,
		model.SysConfigWebhookEnabled,
		model.SysConfigWebhookType,
		model.SysConfigWebhookFleetTemplate,
		model.SysConfigWebhookOBTargetType,
		model.SysConfigWebhookOBTargetID,
		model.SysConfigWebhookOBToken,
		model.SysConfigWebhookQQGroupIDs,
	}
	for i, want := range wantKeys {
		if store.setManyItems[i].Key != want {
			t.Fatalf("unexpected key at index %d: got %q want %q", i, store.setManyItems[i].Key, want)
		}
	}
}

func TestWebhookSetConfigPersistsQQGovernanceGroupIDs(t *testing.T) {
	store := &fakeWebhookConfigStore{}
	svc := &WebhookService{repo: store, http: &http.Client{}}

	err := svc.SetConfig(&WebhookConfig{
		Type:                 "qq_governance_onebot",
		Enabled:              true,
		FleetTemplate:        defaultFleetTemplate,
		QQGovernanceGroupIDs: []int64{123456789, 987654321},
	})
	if err != nil {
		t.Fatalf("expected config update to succeed, got %v", err)
	}

	var groupIDsItem *repository.SysConfigUpsertItem
	for i := range store.setManyItems {
		if store.setManyItems[i].Key == model.SysConfigWebhookQQGroupIDs {
			groupIDsItem = &store.setManyItems[i]
			break
		}
	}
	if groupIDsItem == nil {
		t.Fatalf("expected %q in saved items", model.SysConfigWebhookQQGroupIDs)
	}
	want := `[123456789,987654321]`
	if groupIDsItem.Value != want {
		t.Fatalf("group ids JSON = %q, want %q", groupIDsItem.Value, want)
	}
}

func TestWebhookSetConfigByOperatorWritesAuditEvent(t *testing.T) {
	store := &fakeWebhookConfigStore{}
	svc := &WebhookService{repo: store, http: &http.Client{}, auditSvc: NewAuditService()}

	dsn := fmt.Sprintf("file:webhook_audit_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AuditEvent{}); err != nil {
		t.Fatalf("auto migrate audit event: %v", err)
	}
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	err = svc.SetConfigByOperator(&WebhookConfig{
		URL:           "https://discord.com/api/webhooks/1/token",
		Enabled:       true,
		Type:          "discord",
		FleetTemplate: defaultFleetTemplate,
		OBTargetType:  "group",
		OBTargetID:    42,
		OBToken:       "token",
	}, 77)
	if err != nil {
		t.Fatalf("SetConfigByOperator() error = %v", err)
	}

	var events []model.AuditEvent
	if err := db.Where("resource_type = ? AND action = ?", "system_config", "webhook_config_update").Find(&events).Error; err != nil {
		t.Fatalf("load audit events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected webhook_config_update audit event")
	}
	if events[0].Category != "config" || events[0].ActorUserID != 77 || events[0].Result != model.AuditResultSuccess {
		t.Fatalf("unexpected audit event: %+v", events[0])
	}
}

func TestValidateWebhookRequestTargetAcceptsExpectedURLs(t *testing.T) {
	testCases := []WebhookConfig{
		{Type: "discord", URL: "https://discord.com/api/webhooks/1/abc"},
		{Type: "discord", URL: "https://canary.discord.com/api/webhooks/1/abc"},
		{Type: "feishu", URL: "https://open.feishu.cn/open-apis/bot/v2/hook/abc"},
		{Type: "feishu", URL: "https://open.larksuite.com/open-apis/bot/v2/hook/abc"},
		{Type: "dingtalk", URL: "https://oapi.dingtalk.com/robot/send?access_token=abc"},
		{Type: "dingtalk", URL: "https://api.dingtalkapps.com/robot/send?access_token=abc"},
		{Type: "onebot", URL: "http://127.0.0.1:3000", OBTargetType: "group"},
		{Type: "onebot", URL: "https://onebot.example.com", OBTargetType: "private"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Type+"_"+tc.URL, func(t *testing.T) {
			if err := validateWebhookRequestTarget(&tc); err != nil {
				t.Fatalf("expected URL to pass validation, got %v", err)
			}
		})
	}
}

func TestValidateWebhookRequestTargetRejectsInvalidURLs(t *testing.T) {
	testCases := []struct {
		name string
		cfg  WebhookConfig
	}{
		{name: "invalid_url", cfg: WebhookConfig{Type: "discord", URL: "://bad"}},
		{name: "missing_host", cfg: WebhookConfig{Type: "discord", URL: "https:///abc"}},
		{name: "discord_non_allowlisted_host", cfg: WebhookConfig{Type: "discord", URL: "https://example.com/webhook"}},
		{name: "feishu_non_allowlisted_host", cfg: WebhookConfig{Type: "feishu", URL: "https://example.com/webhook"}},
		{name: "dingtalk_non_allowlisted_host", cfg: WebhookConfig{Type: "dingtalk", URL: "https://example.com/webhook"}},
		{name: "onebot_non_http_scheme", cfg: WebhookConfig{Type: "onebot", URL: "ftp://example.com/webhook"}},
		{name: "onebot_invalid_target_type", cfg: WebhookConfig{Type: "onebot", URL: "https://example.com/webhook", OBTargetType: "channel"}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateWebhookRequestTarget(&tc.cfg)
			if err == nil {
				t.Fatal("expected URL validation error, got nil")
			}
			if !strings.Contains(err.Error(), "webhook 配置错误") {
				t.Fatalf("expected normalized config error, got %v", err)
			}
		})
	}
}

func TestValidateQQGovernanceOnebotTarget(t *testing.T) {
	t.Run("accepts_multiple_group_ids_without_url", func(t *testing.T) {
		cfg := &WebhookConfig{Type: "qq_governance_onebot", QQGovernanceGroupIDs: []int64{123456789, 987654321}}
		if err := validateWebhookRequestTarget(cfg); err != nil {
			t.Fatalf("expected validation to pass, got %v", err)
		}
	})
	t.Run("rejects_empty_group_ids", func(t *testing.T) {
		cfg := &WebhookConfig{Type: "qq_governance_onebot"}
		err := validateWebhookRequestTarget(cfg)
		if err == nil || !strings.Contains(err.Error(), "至少配置一个") {
			t.Fatalf("expected empty group ids error, got %v", err)
		}
	})
	t.Run("rejects_zero_group_id", func(t *testing.T) {
		cfg := &WebhookConfig{Type: "qq_governance_onebot", QQGovernanceGroupIDs: []int64{0}}
		err := validateWebhookRequestTarget(cfg)
		if err == nil || !strings.Contains(err.Error(), "正数") {
			t.Fatalf("expected positive-only error, got %v", err)
		}
	})
	t.Run("rejects_duplicate_group_ids", func(t *testing.T) {
		cfg := &WebhookConfig{Type: "qq_governance_onebot", QQGovernanceGroupIDs: []int64{100, 100}}
		err := validateWebhookRequestTarget(cfg)
		if err == nil || !strings.Contains(err.Error(), "重复") {
			t.Fatalf("expected duplicate error, got %v", err)
		}
	})
	t.Run("rejects_private_target_type", func(t *testing.T) {
		cfg := &WebhookConfig{Type: "qq_governance_onebot", OBTargetType: "private", QQGovernanceGroupIDs: []int64{100}}
		err := validateWebhookRequestTarget(cfg)
		if err == nil || !strings.Contains(err.Error(), "群组目标") {
			t.Fatalf("expected group-only error, got %v", err)
		}
	})
}
