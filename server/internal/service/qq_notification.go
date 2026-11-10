package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"strings"
	"time"
)

type QQNotificationService struct {
	repo     *repository.QQNotificationRepository
	dispatch *QQBotService
}

func NewQQNotificationService(db *gorm.DB, dispatch *QQBotService) *QQNotificationService {
	return &QQNotificationService{repository.NewQQNotificationRepository(db), dispatch}
}
func validNotificationChannel(name string) bool {
	return name == "general" || name == "fleet" || name == "structure"
}
func (s *QQNotificationService) Channels(ctx context.Context) ([]model.QQNotificationChannel, error) {
	return s.repo.Channels(ctx)
}
func (s *QQNotificationService) Targets(ctx context.Context) ([]model.QQNotificationTarget, error) {
	return s.repo.RecentTargets(ctx)
}
func (s *QQNotificationService) Configure(ctx context.Context, name string, enabled bool, groups []int64) error {
	if !validNotificationChannel(name) || len(groups) > 50 || (enabled && len(groups) == 0) {
		return errors.New("invalid channel")
	}
	seen := map[int64]bool{}
	for _, g := range groups {
		if g <= 0 || seen[g] {
			return errors.New("invalid groups")
		}
		seen[g] = true
	}
	raw, err := json.Marshal(groups)
	if err != nil {
		return err
	}
	// Serialize against execution admission so disable invalidates queued work before returning.
	s.dispatch.stepMu.Lock()
	defer s.dispatch.stepMu.Unlock()
	return s.repo.Transaction(ctx, nil, func(tx *gorm.DB) error {
		if err := s.repo.SaveChannel(tx, &model.QQNotificationChannel{Name: name, Enabled: enabled, GroupIDs: string(raw)}); err != nil {
			return err
		}
		if !enabled {
			return s.repo.CancelPending(tx, name)
		}
		return nil
	})
}

// Submit freezes destinations on the first accepted business intent. Caller transactions
// include both the business record and every target operation; a conflict rolls all back.
func (s *QQNotificationService) Submit(ctx context.Context, tx *gorm.DB, channel, key, correlation, text string, expires time.Time) ([]model.QQNotificationTarget, error) {
	if !validNotificationChannel(channel) || strings.TrimSpace(key) == "" || len(key) > 160 || correlation == "" || len(correlation) > 200 {
		return nil, errors.New("invalid notification identity")
	}
	if _, _, _, err := qqbotProtocol(model.QQBotRequest{Kind: "group_message", GroupID: 1, Text: text}); err != nil {
		return nil, err
	}
	id := qqbotHash([]byte(channel + ":" + key))
	raw, err := json.Marshal([]string{correlation, text, expires.UTC().Format(time.RFC3339Nano)})
	if err != nil {
		return nil, err
	}
	hash := qqbotHash(raw)
	var targets []model.QQNotificationTarget
	err = s.repo.Transaction(ctx, tx, func(tx *gorm.DB) error {
		row := model.QQNotification{ID: id, Channel: channel, BusinessKey: key, CorrelationID: correlation, ContentHash: hash, ExpiresAt: expires}
		inserted, e := s.repo.Insert(tx, &row)
		if e != nil {
			return e
		}
		if !inserted {
			stored, e := s.repo.Get(tx, id)
			if e != nil {
				return e
			}
			if stored.ContentHash != hash {
				return errors.New("notification key conflict")
			}
			targets, e = s.repo.Targets(tx, id)
			return e
		}
		config, e := s.repo.Channel(tx, channel)
		if e != nil {
			return e
		}
		if !config.Enabled {
			return errors.New("notification channel disabled")
		}
		var groups []int64
		if e = json.Unmarshal([]byte(config.GroupIDs), &groups); e != nil {
			return e
		}
		if len(groups) == 0 {
			return errors.New("notification has no destinations")
		}
		for _, g := range groups {
			op, e := s.dispatch.Submit(ctx, tx, fmt.Sprintf("notification:%s:%d", id, g), 1, expires, model.QQBotRequest{Kind: "group_message", GroupID: g, Text: text})
			if e != nil {
				return e
			}
			target := model.QQNotificationTarget{NotificationID: id, Channel: channel, CorrelationID: correlation, BusinessKey: key, GroupID: g, OperationID: op.ID, Status: "pending"}
			if e = s.repo.AddTarget(tx, &target); e != nil {
				return e
			}
			targets = append(targets, target)
		}
		return nil
	})
	return targets, err
}
func (s *QQNotificationService) ConsumeResult(_ context.Context, tx *gorm.DB, payload string) error {
	var op model.QQBotOperation
	if err := json.Unmarshal([]byte(payload), &op); err != nil {
		return err
	}
	if !strings.HasPrefix(op.BusinessKey, "notification:") {
		return nil
	}
	status := ""
	switch op.Status {
	case "succeeded":
		status = "confirmed"
	case "failed":
		status = "failed"
	case "unknown":
		status = "unknown"
	case "expired":
		status = "expired"
	case "cancelled":
		status = "cancelled"
	default:
		return errors.New("nonterminal notification result")
	}
	return s.repo.Complete(tx, op.ID, status, op.Result)
}
func (s *QQNotificationService) Test(ctx context.Context, key string, expires time.Time) ([]model.QQNotificationTarget, error) {
	// Text, channel and validity bound are controlled server-side; clients supply a stable request key.
	if expires.After(time.Now().Add(5 * time.Minute)) {
		return nil, errors.New("invalid test expiry")
	}
	return s.Submit(ctx, nil, "general", "test:"+key, key, "QQ Bot notification test / QQ 机器人通知测试", expires)
}

func (s *QQBotService) Notifications() *QQNotificationService { return s.notifications }

func (s *QQNotificationService) valid(tx *gorm.DB, operationID string) (bool, error) {
	row, err := s.repo.OperationNotification(tx, operationID)
	if err != nil {
		return false, err
	}
	if row == nil {
		return true, nil
	}
	config, err := s.repo.Channel(tx, row.Channel)
	if err != nil {
		return false, err
	}
	return config.Enabled, nil
}
