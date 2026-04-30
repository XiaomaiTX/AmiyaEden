package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"fmt"
	"net/http"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
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
		URL:           "https://example.test/webhook",
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
	if len(store.setManyItems) != 7 {
		t.Fatalf("expected 7 config items, got %d", len(store.setManyItems))
	}

	wantKeys := []string{
		model.SysConfigWebhookURL,
		model.SysConfigWebhookEnabled,
		model.SysConfigWebhookType,
		model.SysConfigWebhookFleetTemplate,
		model.SysConfigWebhookOBTargetType,
		model.SysConfigWebhookOBTargetID,
		model.SysConfigWebhookOBToken,
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
		URL:           "https://example.test/webhook",
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
