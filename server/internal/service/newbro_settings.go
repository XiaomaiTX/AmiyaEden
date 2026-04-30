package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"errors"
	"time"
)

type NewbroSupportSettings struct {
	MaxCharacterSP          int64   `json:"max_character_sp"`
	MultiCharacterSP        int64   `json:"multi_character_sp"`
	MultiCharacterThreshold int     `json:"multi_character_threshold"`
	RefreshIntervalDays     int     `json:"refresh_interval_days"`
	BonusRate               float64 `json:"bonus_rate"`
}

type NewbroRecruitSettings struct {
	RecruitQQURL        string  `json:"recruit_qq_url"`
	RecruitRewardAmount float64 `json:"recruit_reward_amount"`
	RecruitCooldownDays int     `json:"recruit_cooldown_days"`
}

type NewbroSettings struct {
	MaxCharacterSP          int64   `json:"max_character_sp"`
	MultiCharacterSP        int64   `json:"multi_character_sp"`
	MultiCharacterThreshold int     `json:"multi_character_threshold"`
	RefreshIntervalDays     int     `json:"refresh_interval_days"`
	BonusRate               float64 `json:"bonus_rate"`
	RecruitQQURL            string  `json:"recruit_qq_url"`
	RecruitRewardAmount     float64 `json:"recruit_reward_amount"`
	RecruitCooldownDays     int     `json:"recruit_cooldown_days"`
}

func DefaultNewbroSupportSettings() NewbroSupportSettings {
	return NewbroSupportSettings{
		MaxCharacterSP:          model.SysConfigDefaultNewbroMaxCharacterSP,
		MultiCharacterSP:        model.SysConfigDefaultNewbroMultiCharacterSP,
		MultiCharacterThreshold: model.SysConfigDefaultNewbroMultiCharacterThreshold,
		RefreshIntervalDays:     model.SysConfigDefaultNewbroRefreshIntervalDays,
		BonusRate:               model.SysConfigDefaultNewbroBonusRate,
	}
}

func DefaultNewbroRecruitSettings() NewbroRecruitSettings {
	return NewbroRecruitSettings{
		RecruitQQURL:        "",
		RecruitRewardAmount: model.SysConfigDefaultNewbroRecruitRewardAmount,
		RecruitCooldownDays: model.SysConfigDefaultNewbroRecruitCooldownDays,
	}
}

func DefaultNewbroSettings() NewbroSettings {
	support := DefaultNewbroSupportSettings()
	recruit := DefaultNewbroRecruitSettings()
	return NewbroSettings{
		MaxCharacterSP:          support.MaxCharacterSP,
		MultiCharacterSP:        support.MultiCharacterSP,
		MultiCharacterThreshold: support.MultiCharacterThreshold,
		RefreshIntervalDays:     support.RefreshIntervalDays,
		BonusRate:               support.BonusRate,
		RecruitQQURL:            recruit.RecruitQQURL,
		RecruitRewardAmount:     recruit.RecruitRewardAmount,
		RecruitCooldownDays:     recruit.RecruitCooldownDays,
	}
}

func (s NewbroSettings) SupportSettings() NewbroSupportSettings {
	return NewbroSupportSettings{
		MaxCharacterSP:          s.MaxCharacterSP,
		MultiCharacterSP:        s.MultiCharacterSP,
		MultiCharacterThreshold: s.MultiCharacterThreshold,
		RefreshIntervalDays:     s.RefreshIntervalDays,
		BonusRate:               s.BonusRate,
	}
}

func (s NewbroSettings) RecruitSettings() NewbroRecruitSettings {
	return NewbroRecruitSettings{
		RecruitQQURL:        s.RecruitQQURL,
		RecruitRewardAmount: s.RecruitRewardAmount,
		RecruitCooldownDays: s.RecruitCooldownDays,
	}
}

func (s NewbroSupportSettings) Validate() error {
	switch {
	case s.MaxCharacterSP <= 0:
		return errors.New("单人物技能点阈值必须大于 0")
	case s.MultiCharacterSP <= 0:
		return errors.New("多人物技能点阈值必须大于 0")
	case s.MultiCharacterThreshold <= 0:
		return errors.New("多人物计数阈值必须大于 0")
	case s.RefreshIntervalDays <= 0:
		return errors.New("资格快照刷新间隔必须大于 0")
	case s.BonusRate < 0:
		return errors.New("队长奖励比例不能小于 0")
	default:
		return nil
	}
}

func (s NewbroRecruitSettings) Validate() error {
	switch {
	case s.RecruitRewardAmount < 0:
		return errors.New("招募奖励金额不能小于 0")
	case s.RecruitCooldownDays <= 0:
		return errors.New("招募链接冷却天数必须大于 0")
	default:
		return nil
	}
}

func (s NewbroSettings) Validate() error {
	if err := s.SupportSettings().Validate(); err != nil {
		return err
	}
	return s.RecruitSettings().Validate()
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
	cfgRepo  newbroSettingsConfigStore
	auditSvc *AuditService
}

type newbroSettingsConfigStore interface {
	GetInt64(key string, defaultVal int64) int64
	GetInt(key string, defaultVal int) int
	GetFloat(key string, defaultVal float64) float64
	GetString(key string, defaultVal string) string
	SetMany(items []repository.SysConfigUpsertItem) error
}

func NewNewbroSettingsService() *NewbroSettingsService {
	return &NewbroSettingsService{
		cfgRepo:  repository.NewSysConfigRepository(),
		auditSvc: NewAuditService(),
	}
}

func (s *NewbroSettingsService) GetSupportSettings() NewbroSupportSettings {
	defaults := DefaultNewbroSupportSettings()
	return NewbroSupportSettings{
		MaxCharacterSP:          s.cfgRepo.GetInt64(model.SysConfigNewbroMaxCharacterSP, defaults.MaxCharacterSP),
		MultiCharacterSP:        s.cfgRepo.GetInt64(model.SysConfigNewbroMultiCharacterSP, defaults.MultiCharacterSP),
		MultiCharacterThreshold: s.cfgRepo.GetInt(model.SysConfigNewbroMultiCharacterThreshold, defaults.MultiCharacterThreshold),
		RefreshIntervalDays:     s.cfgRepo.GetInt(model.SysConfigNewbroRefreshIntervalDays, defaults.RefreshIntervalDays),
		BonusRate:               s.cfgRepo.GetFloat(model.SysConfigNewbroBonusRate, defaults.BonusRate),
	}
}

func (s *NewbroSettingsService) GetRecruitSettings() NewbroRecruitSettings {
	defaults := DefaultNewbroRecruitSettings()
	return NewbroRecruitSettings{
		RecruitQQURL:        s.cfgRepo.GetString(model.SysConfigNewbroRecruitQQURL, defaults.RecruitQQURL),
		RecruitRewardAmount: s.cfgRepo.GetFloat(model.SysConfigNewbroRecruitRewardAmount, defaults.RecruitRewardAmount),
		RecruitCooldownDays: s.cfgRepo.GetInt(model.SysConfigNewbroRecruitCooldownDays, defaults.RecruitCooldownDays),
	}
}

func (s *NewbroSettingsService) GetSettings() NewbroSettings {
	support := s.GetSupportSettings()
	recruit := s.GetRecruitSettings()
	return NewbroSettings{
		MaxCharacterSP:          support.MaxCharacterSP,
		MultiCharacterSP:        support.MultiCharacterSP,
		MultiCharacterThreshold: support.MultiCharacterThreshold,
		RefreshIntervalDays:     support.RefreshIntervalDays,
		BonusRate:               support.BonusRate,
		RecruitQQURL:            recruit.RecruitQQURL,
		RecruitRewardAmount:     recruit.RecruitRewardAmount,
		RecruitCooldownDays:     recruit.RecruitCooldownDays,
	}
}

func (s *NewbroSettingsService) UpdateSupportSettings(cfg NewbroSupportSettings) (NewbroSupportSettings, error) {
	if err := cfg.Validate(); err != nil {
		return NewbroSupportSettings{}, err
	}

	items := newSysConfigBatch(5).
		AddInt64(model.SysConfigNewbroMaxCharacterSP, cfg.MaxCharacterSP, "新人资格：单人物技能点阈值").
		AddInt64(model.SysConfigNewbroMultiCharacterSP, cfg.MultiCharacterSP, "新人资格：多人物技能点阈值").
		AddInt(model.SysConfigNewbroMultiCharacterThreshold, cfg.MultiCharacterThreshold, "新人资格：达到多人物阈值的人物数量").
		AddInt(model.SysConfigNewbroRefreshIntervalDays, cfg.RefreshIntervalDays, "新人资格快照刷新间隔（天）").
		AddFloat64(model.SysConfigNewbroBonusRate, cfg.BonusRate, "队长奖励比例（百分比）").
		Items()

	if err := s.cfgRepo.SetMany(items); err != nil {
		return NewbroSupportSettings{}, err
	}

	return cfg, nil
}

func (s *NewbroSettingsService) UpdateSupportSettingsByOperator(cfg NewbroSupportSettings, operatorID uint) (NewbroSupportSettings, error) {
	updated, err := s.UpdateSupportSettings(cfg)
	if err != nil {
		return NewbroSupportSettings{}, err
	}
	if s.auditSvc != nil {
		_ = s.auditSvc.RecordEvent(context.Background(), AuditRecordInput{
			Category:     "config",
			Action:       "newbro_support_settings_update",
			ActorUserID:  operatorID,
			ResourceType: "system_config",
			ResourceID:   model.SysConfigNewbroMaxCharacterSP,
			Result:       model.AuditResultSuccess,
			Details: map[string]any{
				"max_character_sp":          updated.MaxCharacterSP,
				"multi_character_sp":        updated.MultiCharacterSP,
				"multi_character_threshold": updated.MultiCharacterThreshold,
				"refresh_interval_days":     updated.RefreshIntervalDays,
				"bonus_rate":                updated.BonusRate,
			},
		})
	}
	return updated, nil
}

func (s *NewbroSettingsService) UpdateRecruitSettings(cfg NewbroRecruitSettings) (NewbroRecruitSettings, error) {
	if err := cfg.Validate(); err != nil {
		return NewbroRecruitSettings{}, err
	}

	items := newSysConfigBatch(3).
		AddString(model.SysConfigNewbroRecruitQQURL, cfg.RecruitQQURL, "招募链接 QQ 群邀请地址").
		AddFloat64(model.SysConfigNewbroRecruitRewardAmount, cfg.RecruitRewardAmount, "招募链接有效奖励（伏羲币）").
		AddInt(model.SysConfigNewbroRecruitCooldownDays, cfg.RecruitCooldownDays, "招募链接冷却天数").
		Items()

	if err := s.cfgRepo.SetMany(items); err != nil {
		return NewbroRecruitSettings{}, err
	}

	return cfg, nil
}

func (s *NewbroSettingsService) UpdateRecruitSettingsByOperator(cfg NewbroRecruitSettings, operatorID uint) (NewbroRecruitSettings, error) {
	updated, err := s.UpdateRecruitSettings(cfg)
	if err != nil {
		return NewbroRecruitSettings{}, err
	}
	if s.auditSvc != nil {
		_ = s.auditSvc.RecordEvent(context.Background(), AuditRecordInput{
			Category:     "config",
			Action:       "newbro_recruit_settings_update",
			ActorUserID:  operatorID,
			ResourceType: "system_config",
			ResourceID:   model.SysConfigNewbroRecruitQQURL,
			Result:       model.AuditResultSuccess,
			Details: map[string]any{
				"recruit_qq_url":        updated.RecruitQQURL,
				"recruit_reward_amount": updated.RecruitRewardAmount,
				"recruit_cooldown_days": updated.RecruitCooldownDays,
			},
		})
	}
	return updated, nil
}

func (s *NewbroSettingsService) UpdateSettings(cfg NewbroSettings) (NewbroSettings, error) {
	if err := cfg.Validate(); err != nil {
		return NewbroSettings{}, err
	}

	items := newSysConfigBatch(8).
		AddInt64(model.SysConfigNewbroMaxCharacterSP, cfg.MaxCharacterSP, "新人资格：单人物技能点阈值").
		AddInt64(model.SysConfigNewbroMultiCharacterSP, cfg.MultiCharacterSP, "新人资格：多人物技能点阈值").
		AddInt(model.SysConfigNewbroMultiCharacterThreshold, cfg.MultiCharacterThreshold, "新人资格：达到多人物阈值的人物数量").
		AddInt(model.SysConfigNewbroRefreshIntervalDays, cfg.RefreshIntervalDays, "新人资格快照刷新间隔（天）").
		AddFloat64(model.SysConfigNewbroBonusRate, cfg.BonusRate, "队长奖励比例（百分比）").
		AddString(model.SysConfigNewbroRecruitQQURL, cfg.RecruitQQURL, "招募链接 QQ 群邀请地址").
		AddFloat64(model.SysConfigNewbroRecruitRewardAmount, cfg.RecruitRewardAmount, "招募链接有效奖励（伏羲币）").
		AddInt(model.SysConfigNewbroRecruitCooldownDays, cfg.RecruitCooldownDays, "招募链接冷却天数").
		Items()

	if err := s.cfgRepo.SetMany(items); err != nil {
		return NewbroSettings{}, err
	}

	return cfg, nil
}
