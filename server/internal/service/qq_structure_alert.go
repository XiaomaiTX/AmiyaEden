package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"strconv"
	"time"
)

type QQStructureAlertService struct {
	repo          *repository.QQStructureAlertRepository
	notifications *QQNotificationService
}

func NewQQStructureAlertService(db *gorm.DB, n *QQNotificationService) *QQStructureAlertService {
	return &QQStructureAlertService{repository.NewQQStructureAlertRepository(db), n}
}
func (s *QQStructureAlertService) threshold(ctx context.Context, key string, fallback int) (int, error) {
	row, err := s.repo.Threshold(ctx, key)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fallback, nil
	}
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(row.Value)
	if err != nil || v < 0 || v > 365 {
		return 0, errors.New("invalid structure threshold")
	}
	return v, nil
}
func (s *QQStructureAlertService) Scan(ctx context.Context, now time.Time) error {
	enabled := false
	if err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		c, e := s.notifications.repo.Channel(tx, "structure")
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return nil
		}
		enabled = c.Enabled
		return e
	}); err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	fuel, err := s.threshold(ctx, model.SysConfigDashboardCorpStructuresFuelNoticeThresholdDays, model.SysConfigDefaultDashboardCorpStructuresFuelNoticeThresholdDays)
	if err != nil {
		return err
	}
	timer, err := s.threshold(ctx, model.SysConfigDashboardCorpStructuresTimerNoticeThresholdDays, model.SysConfigDefaultDashboardCorpStructuresTimerNoticeThresholdDays)
	if err != nil {
		return err
	}
	rows, err := s.repo.Structures(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.UpdateAt <= 0 || time.Unix(row.UpdateAt, 0).Before(now.Add(-24*time.Hour)) {
			continue
		}
		for _, kind := range []string{"fuel", "timer"} {
			days, target := fuel, row.FuelExpires
			if kind == "timer" {
				days, target = timer, row.StateTimerEnd
			}
			if err = s.evaluate(ctx, row, kind, days, target, now); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s *QQStructureAlertService) evaluate(ctx context.Context, row model.CorpStructureInfo, kind string, days int, target string, now time.Time) error {
	deadline, known := parseFlexibleTime(target)
	if known {
		target = deadline.UTC().Format(time.RFC3339Nano)
	}
	active := days > 0 && known && !deadline.After(now.Add(time.Duration(days)*24*time.Hour))
	if kind == "timer" && deadline.Before(now) {
		active = false
	}
	// Unknown source values are insufficient evidence of recovery, except an explicit disabled threshold.
	if !known && days > 0 {
		return nil
	}
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		state, err := s.repo.State(tx, row.StructureID, kind)
		if err != nil {
			return err
		}
		if state.Active == active && (!active || kind != "timer" || state.Target == target) {
			return nil
		}
		if state.NotificationID != "" {
			if e := s.notifications.repo.CancelNotification(tx, state.NotificationID); e != nil {
				return e
			}
		}
		state.Active = active
		state.Target = target
		state.Version++
		if active {
			expires := now.Add(6 * time.Hour)
			if kind == "timer" && deadline.Before(expires) {
				expires = deadline
			}
			if !expires.After(time.Now()) {
				return nil
			}
			key := fmt.Sprintf("structure:%d:%s:%d", row.StructureID, kind, state.Version)
			text := fmt.Sprintf("建筑预警 / Structure alert: %s\n%s | %s\n%s", row.Name, row.CorporationName, kind, target)
			targets, e := s.notifications.Submit(ctx, tx, "structure", key, fmt.Sprintf("%d", row.StructureID), text, expires)
			if e != nil {
				return e
			}
			state.NotificationID = targets[0].NotificationID
		}
		return s.repo.Save(tx, &state)
	})
}
