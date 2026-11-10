package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fakeWebhookConfigStore struct {
	values       map[string]string
	enabled      bool
	setManyCalls int
	setManyItems []repository.SysConfigUpsertItem
	setManyErr   error
}

func (f *fakeWebhookConfigStore) Get(key string, defaultVal string) (string, error) {
	if value, ok := f.values[key]; ok {
		return value, nil
	}
	return defaultVal, nil
}

func (f *fakeWebhookConfigStore) GetBool(_ string, defaultVal bool) bool {
	return f.enabled || defaultVal
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
	})
	if err != nil {
		t.Fatalf("expected config update to succeed, got %v", err)
	}
	if store.setManyCalls != 1 {
		t.Fatalf("expected exactly one batch write, got %d", store.setManyCalls)
	}
	if len(store.setManyItems) != 4 {
		t.Fatalf("expected 4 config items, got %d", len(store.setManyItems))
	}

	wantKeys := []string{
		model.SysConfigWebhookURL,
		model.SysConfigWebhookEnabled,
		model.SysConfigWebhookType,
		model.SysConfigWebhookFleetTemplate,
	}
	for i, want := range wantKeys {
		if store.setManyItems[i].Key != want {
			t.Fatalf("unexpected key at index %d: got %q want %q", i, store.setManyItems[i].Key, want)
		}
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

func TestRetiredWebhookTypesRejectWithoutSendingOrSaving(t *testing.T) {
	for _, kind := range []string{"onebot", "qq_governance_onebot"} {
		t.Run(kind, func(t *testing.T) {
			store := &fakeWebhookConfigStore{}
			transport := &recordingWebhookTransport{}
			svc := &WebhookService{repo: store, http: &http.Client{Transport: transport}}
			cfg := &WebhookConfig{Type: kind, URL: "https://discord.com/api/webhooks/1/token", Enabled: true}
			if err := svc.SetConfig(cfg); err == nil {
				t.Fatal("retired type must be rejected")
			}
			if err := svc.SendTest(cfg, "test"); err == nil {
				t.Fatal("retired type must not send a test message")
			}
			if err := svc.sendMessage(cfg, "fleet ping"); err == nil {
				t.Fatal("persisted retired type must not fall through to Discord")
			}
			if store.setManyCalls != 0 || transport.calls != 0 {
				t.Fatal("retired type wrote configuration or sent HTTP request")
			}
		})
	}
}

type recordingWebhookTransport struct {
	calls int
	body  string
}

func (r *recordingWebhookTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.calls++
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	r.body = string(body)
	return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
}

func TestSupportedWebhookProvidersStillSend(t *testing.T) {
	cases := []struct{ kind, url, payload string }{
		{"discord", "https://discord.com/api/webhooks/1/token", `{"content":"hello"}`},
		{"feishu", "https://open.feishu.cn/open-apis/bot/v2/hook/test", `{"content":{"text":"hello"},"msg_type":"text"}`},
		{"dingtalk", "https://oapi.dingtalk.com/robot/send?access_token=test", `{"msgtype":"text","text":{"content":"hello"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			transport := &recordingWebhookTransport{}
			svc := &WebhookService{http: &http.Client{Transport: transport}}
			if err := svc.SendTest(&WebhookConfig{Type: tc.kind, URL: tc.url}, "hello"); err != nil {
				t.Fatal(err)
			}
			if transport.calls != 1 || transport.body != tc.payload {
				t.Fatalf("unexpected delivery: calls=%d payload=%s", transport.calls, transport.body)
			}
		})
	}
}

func TestFleetPingKeepsSupportedDeliveryAndRejectsRetiredConfiguration(t *testing.T) {
	for _, kind := range []string{"discord", "onebot", "qq_governance_onebot"} {
		t.Run(kind, func(t *testing.T) {
			transport := &recordingWebhookTransport{}
			store := &fakeWebhookConfigStore{enabled: true, values: map[string]string{
				model.SysConfigWebhookType:          kind,
				model.SysConfigWebhookURL:           "https://discord.com/api/webhooks/1/token",
				model.SysConfigWebhookFleetTemplate: "{title} {fc_name} {pap_count}",
			}}
			svc := &WebhookService{repo: store, http: &http.Client{Transport: transport}}
			err := svc.SendFleetPing(&model.Fleet{Title: "Operation", FCCharacterName: "Commander", PapCount: 2})
			if kind == "discord" {
				if err != nil || transport.calls != 1 || transport.body != `{"content":"Operation Commander 2"}` {
					t.Fatalf("fleet delivery: err=%v calls=%d body=%s", err, transport.calls, transport.body)
				}
			} else if err == nil || transport.calls != 0 {
				t.Fatalf("retired fleet notification: err=%v calls=%d", err, transport.calls)
			}
		})
	}
}
