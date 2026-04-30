package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
	"testing"
)

type fakeWelfareSettingsConfigStore struct {
	threshold    int
	hasThreshold bool
	setManyCalls int
	setManyItems []repository.SysConfigUpsertItem
	setManyErr   error
}

func (f *fakeWelfareSettingsConfigStore) GetInt(key string, defaultVal int) int {
	if key == model.SysConfigWelfareAutoApproveFuxiCoinThreshold && f.hasThreshold {
		return f.threshold
	}
	return defaultVal
}

func (f *fakeWelfareSettingsConfigStore) SetMany(items []repository.SysConfigUpsertItem) error {
	if f.setManyErr != nil {
		return f.setManyErr
	}

	f.setManyCalls++
	f.setManyItems = append([]repository.SysConfigUpsertItem(nil), items...)
	for _, item := range items {
		if item.Key == model.SysConfigWelfareAutoApproveFuxiCoinThreshold {
			f.hasThreshold = true
		}
	}
	return nil
}

func TestDefaultWelfareSettings(t *testing.T) {
	cfg := DefaultWelfareSettings()

	if cfg.AutoApproveFuxiCoinThreshold != model.SysConfigDefaultWelfareAutoApproveFuxiCoinThreshold {
		t.Fatalf(
			"expected AutoApproveFuxiCoinThreshold %d, got %d",
			model.SysConfigDefaultWelfareAutoApproveFuxiCoinThreshold,
			cfg.AutoApproveFuxiCoinThreshold,
		)
	}
}

func TestValidateWelfareSettings(t *testing.T) {
	valid := DefaultWelfareSettings()
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid defaults, got %v", err)
	}

	valid.AutoApproveFuxiCoinThreshold = 0
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected zero threshold to be valid, got %v", err)
	}

	invalid := DefaultWelfareSettings()
	invalid.AutoApproveFuxiCoinThreshold = -1
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected negative threshold to be rejected")
	}
}

func TestWelfareSettingsGetSettingsUsesConfigOverride(t *testing.T) {
	store := &fakeWelfareSettingsConfigStore{threshold: 350, hasThreshold: true}
	svc := &WelfareSettingsService{cfgRepo: store}

	got := svc.GetSettings()

	if got.AutoApproveFuxiCoinThreshold != 350 {
		t.Fatalf("expected threshold 350, got %d", got.AutoApproveFuxiCoinThreshold)
	}
}

func TestUpdateWelfareSettingsPersistsSingleBatch(t *testing.T) {
	store := &fakeWelfareSettingsConfigStore{}
	svc := &WelfareSettingsService{cfgRepo: store}
	cfg := WelfareSettings{AutoApproveFuxiCoinThreshold: 275}

	updated, err := svc.UpdateSettings(cfg)
	if err != nil {
		t.Fatalf("expected update to succeed, got %v", err)
	}
	if updated != cfg {
		t.Fatalf("expected updated settings %v, got %v", cfg, updated)
	}
	if store.setManyCalls != 1 {
		t.Fatalf("expected exactly one batch write, got %d", store.setManyCalls)
	}
	if len(store.setManyItems) != 1 {
		t.Fatalf("expected 1 settings entry, got %d", len(store.setManyItems))
	}
	if store.setManyItems[0].Key != model.SysConfigWelfareAutoApproveFuxiCoinThreshold {
		t.Fatalf("unexpected key %q", store.setManyItems[0].Key)
	}
}

func TestUpdateWelfareSettingsReturnsBatchWriteError(t *testing.T) {
	store := &fakeWelfareSettingsConfigStore{setManyErr: errors.New("write failed")}
	svc := &WelfareSettingsService{cfgRepo: store}

	_, err := svc.UpdateSettings(DefaultWelfareSettings())
	if err == nil {
		t.Fatal("expected batch write error")
	}
	if store.setManyCalls != 0 {
		t.Fatalf("expected no successful batch writes, got %d", store.setManyCalls)
	}
}

func TestUpdateWelfareSettingsAcceptsZeroThreshold(t *testing.T) {
	store := &fakeWelfareSettingsConfigStore{}
	svc := &WelfareSettingsService{cfgRepo: store}
	cfg := WelfareSettings{AutoApproveFuxiCoinThreshold: 0}

	updated, err := svc.UpdateSettings(cfg)
	if err != nil {
		t.Fatalf("expected zero threshold to be accepted, got %v", err)
	}
	if updated.AutoApproveFuxiCoinThreshold != 0 {
		t.Fatalf("expected threshold 0, got %d", updated.AutoApproveFuxiCoinThreshold)
	}
	if store.setManyCalls != 1 {
		t.Fatalf("expected one batch write, got %d", store.setManyCalls)
	}
}

func TestUpdateWelfareSettingsByOperatorWritesAuditEvent(t *testing.T) {
	db := newServiceTestDB(t, "welfare_settings_audit", &model.AuditEvent{})
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	store := &fakeWelfareSettingsConfigStore{}
	svc := &WelfareSettingsService{cfgRepo: store, auditSvc: NewAuditService()}

	_, err := svc.UpdateSettingsByOperator(WelfareSettings{AutoApproveFuxiCoinThreshold: 333}, 66)
	if err != nil {
		t.Fatalf("UpdateSettingsByOperator() error = %v", err)
	}

	var events []model.AuditEvent
	if err := db.Where("resource_type = ? AND action = ?", "system_config", "welfare_settings_update").Find(&events).Error; err != nil {
		t.Fatalf("load audit events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected welfare_settings_update audit event")
	}
	if events[0].Category != "config" || events[0].ActorUserID != 66 || events[0].Result != model.AuditResultSuccess {
		t.Fatalf("unexpected audit event: %+v", events[0])
	}
}
