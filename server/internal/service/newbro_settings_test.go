package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
	"testing"
	"time"
)

type fakeNewbroSettingsConfigStore struct {
	setManyCalls int
	setManyItems []repository.SysConfigUpsertItem
	setManyErr   error
}

func (f *fakeNewbroSettingsConfigStore) GetInt64(_ string, defaultVal int64) int64 {
	return defaultVal
}

func (f *fakeNewbroSettingsConfigStore) GetInt(_ string, defaultVal int) int {
	return defaultVal
}

func (f *fakeNewbroSettingsConfigStore) GetFloat(_ string, defaultVal float64) float64 {
	return defaultVal
}

func (f *fakeNewbroSettingsConfigStore) SetMany(items []repository.SysConfigUpsertItem) error {
	f.setManyCalls++
	f.setManyItems = append([]repository.SysConfigUpsertItem(nil), items...)
	return f.setManyErr
}

func TestDefaultNewbroSettings(t *testing.T) {
	cfg := DefaultNewbroSettings()

	if cfg.MaxCharacterSP != 20_000_000 {
		t.Fatalf("expected MaxCharacterSP 20000000, got %d", cfg.MaxCharacterSP)
	}
	if cfg.MultiCharacterSP != 10_000_000 {
		t.Fatalf("expected MultiCharacterSP 10000000, got %d", cfg.MultiCharacterSP)
	}
	if cfg.MultiCharacterThreshold != 3 {
		t.Fatalf("expected MultiCharacterThreshold 3, got %d", cfg.MultiCharacterThreshold)
	}
	if cfg.RefreshIntervalDays != 7 {
		t.Fatalf("expected RefreshIntervalDays 7, got %d", cfg.RefreshIntervalDays)
	}
	if cfg.BonusRate != 20 {
		t.Fatalf("expected BonusRate 20, got %v", cfg.BonusRate)
	}
}

func TestNewbroSettingsToEligibilityRules(t *testing.T) {
	cfg := NewbroSettings{
		MaxCharacterSP:          21_000_000,
		MultiCharacterSP:        11_000_000,
		MultiCharacterThreshold: 4,
		RefreshIntervalDays:     9,
		BonusRate:               35,
	}

	rules := cfg.ToEligibilityRules()

	if rules.MaxCharacterSP != 21_000_000 {
		t.Fatalf("expected MaxCharacterSP 21000000, got %d", rules.MaxCharacterSP)
	}
	if rules.MultiCharacterSP != 11_000_000 {
		t.Fatalf("expected MultiCharacterSP 11000000, got %d", rules.MultiCharacterSP)
	}
	if rules.MultiCharacterThreshold != 4 {
		t.Fatalf("expected MultiCharacterThreshold 4, got %d", rules.MultiCharacterThreshold)
	}
	if rules.AttributionLookbackDays != newbroAttributionLookbackDays {
		t.Fatalf("expected AttributionLookbackDays %d, got %d", newbroAttributionLookbackDays, rules.AttributionLookbackDays)
	}
}

func TestNewbroSettingsRefreshInterval(t *testing.T) {
	cfg := NewbroSettings{RefreshIntervalDays: 9}

	if got := cfg.RefreshInterval(); got != 9*24*time.Hour {
		t.Fatalf("expected refresh interval %s, got %s", 9*24*time.Hour, got)
	}
}

func TestValidateNewbroSettings(t *testing.T) {
	valid := DefaultNewbroSettings()
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid defaults, got error %v", err)
	}

	valid.BonusRate = 0
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected zero bonus rate to remain valid, got error %v", err)
	}

	invalidCases := []struct {
		name string
		cfg  NewbroSettings
	}{
		{
			name: "max character sp must be positive",
			cfg: NewbroSettings{
				MaxCharacterSP:          0,
				MultiCharacterSP:        10_000_000,
				MultiCharacterThreshold: 3,
				RefreshIntervalDays:     7,
				BonusRate:               20,
			},
		},
		{
			name: "multi character sp must be positive",
			cfg: NewbroSettings{
				MaxCharacterSP:          20_000_000,
				MultiCharacterSP:        0,
				MultiCharacterThreshold: 3,
				RefreshIntervalDays:     7,
				BonusRate:               20,
			},
		},
		{
			name: "multi character threshold must be positive",
			cfg: NewbroSettings{
				MaxCharacterSP:          20_000_000,
				MultiCharacterSP:        10_000_000,
				MultiCharacterThreshold: 0,
				RefreshIntervalDays:     7,
				BonusRate:               20,
			},
		},
		{
			name: "refresh interval must be positive",
			cfg: NewbroSettings{
				MaxCharacterSP:          20_000_000,
				MultiCharacterSP:        10_000_000,
				MultiCharacterThreshold: 3,
				RefreshIntervalDays:     0,
				BonusRate:               20,
			},
		},
		{
			name: "bonus rate must not be negative",
			cfg: NewbroSettings{
				MaxCharacterSP:          20_000_000,
				MultiCharacterSP:        10_000_000,
				MultiCharacterThreshold: 3,
				RefreshIntervalDays:     7,
				BonusRate:               -1,
			},
		},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestUpdateNewbroSettingsPersistsAllKeysInSingleBatch(t *testing.T) {
	store := &fakeNewbroSettingsConfigStore{}
	svc := &NewbroSettingsService{cfgRepo: store}
	cfg := NewbroSettings{
		MaxCharacterSP:          21_000_000,
		MultiCharacterSP:        11_000_000,
		MultiCharacterThreshold: 4,
		RefreshIntervalDays:     9,
		BonusRate:               35,
	}

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
	if len(store.setManyItems) != 5 {
		t.Fatalf("expected 5 settings entries, got %d", len(store.setManyItems))
	}

	gotKeys := []string{
		store.setManyItems[0].Key,
		store.setManyItems[1].Key,
		store.setManyItems[2].Key,
		store.setManyItems[3].Key,
		store.setManyItems[4].Key,
	}
	wantKeys := []string{
		model.SysConfigNewbroMaxCharacterSP,
		model.SysConfigNewbroMultiCharacterSP,
		model.SysConfigNewbroMultiCharacterThreshold,
		model.SysConfigNewbroRefreshIntervalDays,
		model.SysConfigNewbroBonusRate,
	}
	for i := range wantKeys {
		if gotKeys[i] != wantKeys[i] {
			t.Fatalf("unexpected key at index %d: got %q want %q", i, gotKeys[i], wantKeys[i])
		}
	}
}

func TestUpdateNewbroSettingsReturnsBatchWriteError(t *testing.T) {
	store := &fakeNewbroSettingsConfigStore{setManyErr: errors.New("write failed")}
	svc := &NewbroSettingsService{cfgRepo: store}

	_, err := svc.UpdateSettings(DefaultNewbroSettings())
	if err == nil {
		t.Fatal("expected batch write error")
	}
	if store.setManyCalls != 1 {
		t.Fatalf("expected one batch write attempt, got %d", store.setManyCalls)
	}
}
