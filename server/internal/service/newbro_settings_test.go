package service

import (
	"testing"
	"time"
)

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
