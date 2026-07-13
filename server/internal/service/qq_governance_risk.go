package service

import (
	"amiya-eden/internal/model"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

func (s *QQGovernanceService) riskWait(task *model.QQGovernanceActionTask) (time.Duration, error) {
	if task.Source != "automatic" && task.Source != "cron" && task.Source != "reconcile" {
		return 0, nil
	}
	botQQ := NewSysConfigService().GetOneBotConfig().BotQQ
	if botQQ <= 0 {
		return 0, nil
	}
	state, err := s.repo.GetRiskState(botQQ)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	now := s.now()
	if state.OpenUntil != nil && state.OpenUntil.After(now) {
		switch state.Level {
		case model.QQGovernanceRiskLevelThree:
			if task.ActionType != model.QQGovernanceActionScan && task.ActionType != model.QQGovernanceActionRecheck {
				return state.OpenUntil.Sub(now), nil
			}
		case model.QQGovernanceRiskLevelTwo:
			if task.ActionType == model.QQGovernanceActionKick || task.ActionType == model.QQGovernanceActionSetCard {
				return state.OpenUntil.Sub(now), nil
			}
		case model.QQGovernanceRiskLevelOne:
			if task.ActionType == model.QQGovernanceActionSetCard || task.ActionType == model.QQGovernanceActionScan {
				return 30 * time.Second, nil
			}
		}
	}
	return 0, nil
}

func (s *QQGovernanceService) recordRiskOutcome(task *model.QQGovernanceActionTask, success bool, cause error) {
	botQQ := NewSysConfigService().GetOneBotConfig().BotQQ
	if botQQ <= 0 {
		return
	}
	now := s.now()
	total, failed, err := s.repo.CountActionLogsSince(now.Add(-5 * time.Minute))
	if err != nil {
		return
	}
	rate := 0.0
	if total > 0 {
		rate = float64(failed) / float64(total)
	}
	level := model.QQGovernanceRiskNormal
	if cause != nil && strings.Contains(cause.Error(), "风控") {
		level = model.QQGovernanceRiskLevelTwo
	} else if total >= 5 && rate > 0.4 {
		level = model.QQGovernanceRiskLevelTwo
	} else if total >= 5 && rate > 0.2 {
		level = model.QQGovernanceRiskLevelOne
	}
	if !success && failed >= 10 {
		level = model.QQGovernanceRiskLevelThree
	}
	state, err := s.repo.GetRiskState(botQQ)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		state = &model.QQGovernanceRiskControlState{BotQQ: botQQ}
	} else if err != nil {
		return
	}
	if level == model.QQGovernanceRiskNormal {
		if state.OpenUntil != nil && !state.OpenUntil.After(now) {
			state.Level = model.QQGovernanceRiskNormal
			state.OpenUntil = nil
			state.OpenedAt = nil
			state.HalfOpenLeft = 0
			_ = s.repo.SaveRiskState(state)
			_ = s.repo.ResolveAlert("circuit", now)
		}
		return
	}
	if level < state.Level && state.OpenUntil != nil && state.OpenUntil.After(now) {
		return
	}
	state.Level = level
	state.OpenedAt = &now
	until := now.Add(30 * time.Minute)
	if level == model.QQGovernanceRiskLevelOne {
		until = now.Add(5 * time.Minute)
	}
	state.OpenUntil = &until
	state.HalfOpenLeft = 3
	_ = s.repo.SaveRiskState(state)
	_, _ = s.createAlert("circuit", 0, 0, 0, fmt.Sprintf("OneBot 风险控制已进入第 %d 级", level))
}

func (s *QQGovernanceService) ResetRiskControl(operator uint) error {
	botQQ := NewSysConfigService().GetOneBotConfig().BotQQ
	if botQQ <= 0 {
		return errors.New("OneBot 机器人配置不可用")
	}
	now := s.now()
	state, err := s.repo.GetRiskState(botQQ)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	state.Level = model.QQGovernanceRiskNormal
	state.OpenedAt = nil
	state.OpenUntil = nil
	state.HalfOpenLeft = 0
	state.UpdatedBy = operator
	if err := s.repo.SaveRiskState(state); err != nil {
		return err
	}
	return s.repo.ResolveAlert("circuit", now)
}

func (s *QQGovernanceService) createAlert(kind string, groupID, qq int64, taskID uint, message string) (bool, error) {
	key := kind
	if groupID > 0 || qq > 0 {
		key = fmt.Sprintf("%s:%d:%d", kind, groupID, qq)
	}
	if taskID > 0 {
		key = fmt.Sprintf("%s:%d", kind, taskID)
	}
	return s.repo.CreateAlertIfAbsent(&model.QQGovernanceAlert{AlertKey: key, Kind: kind, GroupID: groupID, QQ: qq, TaskID: taskID, Status: model.QQGovernanceAlertOpen, Message: message})
}
