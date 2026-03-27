package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
	"fmt"
	"time"
)

type NewbroSettings struct {
	MaxCharacterSP          int64   `json:"max_character_sp"`
	MultiCharacterSP        int64   `json:"multi_character_sp"`
	MultiCharacterThreshold int     `json:"multi_character_threshold"`
	RefreshIntervalDays     int     `json:"refresh_interval_days"`
	BonusRate               float64 `json:"bonus_rate"`
}

func DefaultNewbroSettings() NewbroSettings {
	return NewbroSettings{
		MaxCharacterSP:          model.SysConfigDefaultNewbroMaxCharacterSP,
		MultiCharacterSP:        model.SysConfigDefaultNewbroMultiCharacterSP,
		MultiCharacterThreshold: model.SysConfigDefaultNewbroMultiCharacterThreshold,
		RefreshIntervalDays:     model.SysConfigDefaultNewbroRefreshIntervalDays,
		BonusRate:               model.SysConfigDefaultNewbroBonusRate,
	}
}

func (s NewbroSettings) Validate() error {
	switch {
	case s.MaxCharacterSP <= 0:
		return errors.New("单角色技能点阈值必须大于 0")
	case s.MultiCharacterSP <= 0:
		return errors.New("多角色技能点阈值必须大于 0")
	case s.MultiCharacterThreshold <= 0:
		return errors.New("多角色计数阈值必须大于 0")
	case s.RefreshIntervalDays <= 0:
		return errors.New("资格快照刷新间隔必须大于 0")
	case s.BonusRate < 0:
		return errors.New("队长奖励比例不能小于 0")
	default:
		return nil
	}
}

func (s NewbroSettings) ToEligibilityRules() NewbroEligibilityRules {
	return NewbroEligibilityRules{
		MaxCharacterSP:          s.MaxCharacterSP,
		MultiCharacterSP:        s.MultiCharacterSP,
		MultiCharacterThreshold: s.MultiCharacterThreshold,
		AttributionLookbackDays: newbroAttributionLookbackDays,
	}
}

func (s NewbroSettings) RefreshInterval() time.Duration {
	return time.Duration(s.RefreshIntervalDays) * 24 * time.Hour
}

type NewbroSettingsService struct {
	cfgRepo *repository.SysConfigRepository
}

func NewNewbroSettingsService() *NewbroSettingsService {
	return &NewbroSettingsService{
		cfgRepo: repository.NewSysConfigRepository(),
	}
}

func (s *NewbroSettingsService) GetSettings() NewbroSettings {
	defaults := DefaultNewbroSettings()
	return NewbroSettings{
		MaxCharacterSP:          s.cfgRepo.GetInt64(model.SysConfigNewbroMaxCharacterSP, defaults.MaxCharacterSP),
		MultiCharacterSP:        s.cfgRepo.GetInt64(model.SysConfigNewbroMultiCharacterSP, defaults.MultiCharacterSP),
		MultiCharacterThreshold: s.cfgRepo.GetInt(model.SysConfigNewbroMultiCharacterThreshold, defaults.MultiCharacterThreshold),
		RefreshIntervalDays:     s.cfgRepo.GetInt(model.SysConfigNewbroRefreshIntervalDays, defaults.RefreshIntervalDays),
		BonusRate:               s.cfgRepo.GetFloat(model.SysConfigNewbroBonusRate, defaults.BonusRate),
	}
}

func (s *NewbroSettingsService) UpdateSettings(cfg NewbroSettings) (NewbroSettings, error) {
	if err := cfg.Validate(); err != nil {
		return NewbroSettings{}, err
	}

	items := []struct {
		key   string
		value string
		desc  string
	}{
		{
			key:   model.SysConfigNewbroMaxCharacterSP,
			value: fmt.Sprintf("%d", cfg.MaxCharacterSP),
			desc:  "新人资格：单角色技能点阈值",
		},
		{
			key:   model.SysConfigNewbroMultiCharacterSP,
			value: fmt.Sprintf("%d", cfg.MultiCharacterSP),
			desc:  "新人资格：多角色技能点阈值",
		},
		{
			key:   model.SysConfigNewbroMultiCharacterThreshold,
			value: fmt.Sprintf("%d", cfg.MultiCharacterThreshold),
			desc:  "新人资格：达到多角色阈值的角色数量",
		},
		{
			key:   model.SysConfigNewbroRefreshIntervalDays,
			value: fmt.Sprintf("%d", cfg.RefreshIntervalDays),
			desc:  "新人资格快照刷新间隔（天）",
		},
		{
			key:   model.SysConfigNewbroBonusRate,
			value: fmt.Sprintf("%g", cfg.BonusRate),
			desc:  "队长奖励比例（百分比）",
		},
	}

	for _, item := range items {
		if err := s.cfgRepo.Set(item.key, item.value, item.desc); err != nil {
			return NewbroSettings{}, err
		}
	}

	return cfg, nil
}
