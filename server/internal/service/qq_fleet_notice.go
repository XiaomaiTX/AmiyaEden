package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"time"
	"unicode/utf8"
)

// SubmitFleetCreated is called only inside the fleet creation transaction.
func (s *QQNotificationService) SubmitFleetCreated(ctx context.Context, tx *gorm.DB, fleet *model.Fleet) error {
	config, err := s.repo.Channel(tx, "fleet")
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !config.Enabled {
		return nil
	}
	expires := fleet.CreatedAt.Add(30 * time.Minute)
	if fleet.StartAt.Before(expires) {
		expires = fleet.StartAt
	}
	if !expires.After(time.Now()) {
		return nil
	}
	text := fmt.Sprintf("舰队 / Fleet: %s\nFC: %s\n%s ~ %s\n%s | PAP: %g\n%s", fleet.Title, fleet.FCCharacterName, fleet.StartAt.UTC().Format(time.RFC3339), fleet.EndAt.UTC().Format(time.RFC3339), fleet.Importance, fleet.PapCount, fleet.Description)
	if len(text) > 16000 {
		text = text[:15997]
		for !utf8.ValidString(text) {
			text = text[:len(text)-1]
		}
		text += "..."
	}
	_, err = s.Submit(ctx, tx, "fleet", "fleet:"+fleet.ID+":created", fleet.ID, text, expires)
	return err
}
func (s *FleetService) createFleetWithNotice(ctx context.Context, fleet *model.Fleet, send bool) error {
	notifications := s.notifications
	if notifications == nil {
		notifications = NewQQNotificationService(global.DB, DefaultQQBotService())
	}
	fleet.CreatedAt = time.Now().UTC().Truncate(time.Microsecond)
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.repo.CreateTx(tx, fleet); err != nil {
			return err
		}
		if send {
			return notifications.SubmitFleetCreated(ctx, tx, fleet)
		}
		return nil
	})
}
