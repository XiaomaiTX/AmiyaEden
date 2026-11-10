package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"errors"
)

type NewbroRecruitSettings struct {
	RecruitQQURL        string  `json:"recruit_qq_url"`
	RecruitRewardAmount float64 `json:"recruit_reward_amount"`
	RecruitCooldownDays int     `json:"recruit_cooldown_days"`
}

func DefaultNewbroRecruitSettings() NewbroRecruitSettings {
	return NewbroRecruitSettings{
		RecruitQQURL:        "",
		RecruitRewardAmount: model.SysConfigDefaultNewbroRecruitRewardAmount,
		RecruitCooldownDays: model.SysConfigDefaultNewbroRecruitCooldownDays,
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

type NewbroSettingsService struct {
	cfgRepo  *repository.SysConfigRepository
	auditSvc *AuditService
}

func NewNewbroSettingsService() *NewbroSettingsService {
	return &NewbroSettingsService{repository.NewSysConfigRepository(), NewAuditService()}
}
func (s *NewbroSettingsService) GetSettings() NewbroRecruitSettings { return s.GetRecruitSettings() }
func (s *NewbroSettingsService) GetRecruitSettings() NewbroRecruitSettings {
	defaults := DefaultNewbroRecruitSettings()
	return NewbroRecruitSettings{
		RecruitQQURL:        s.cfgRepo.GetString(model.SysConfigNewbroRecruitQQURL, defaults.RecruitQQURL),
		RecruitRewardAmount: s.cfgRepo.GetFloat(model.SysConfigNewbroRecruitRewardAmount, defaults.RecruitRewardAmount),
		RecruitCooldownDays: s.cfgRepo.GetInt(model.SysConfigNewbroRecruitCooldownDays, defaults.RecruitCooldownDays),
	}
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
