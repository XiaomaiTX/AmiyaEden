package service

import (
	"amiya-eden/internal/model"
	"context"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestQQOfficialPauseRetainsButDisableNeverReplays(t *testing.T) {
	s, _, now := qqFixture(t)
	qqEnable(t, s)
	group := qqGroupFixture(t, s, 1)
	ctx := t.Context()
	first := qqSubmitTest(t, s, group)
	paused := true
	if _, err := s.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{Paused: &paused}); err != nil {
		t.Fatal(err)
	}
	if permit, err := s.claim(ctx, qqToken(*now), 0); permit != nil || err != nil {
		t.Fatalf("paused claimed: %+v %v", permit, err)
	}
	result, _, err := s.Task(ctx, first.ID)
	if err != nil || result.Status != model.QQQueued {
		t.Fatalf("paused task %+v: %v", result, err)
	}
	enabled := false
	if _, err = s.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	paused = false
	if _, err = s.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{Paused: &paused}); err != nil {
		t.Fatal(err)
	}
	qqEnable(t, s)
	if permit, err := s.claim(ctx, qqToken(*now), 0); permit != nil || err != nil {
		t.Fatalf("disabled task replayed: %+v %v", permit, err)
	}
	result, _, err = s.Task(ctx, first.ID)
	if err != nil || result.Status != model.QQCancelled || result.Text != "" {
		t.Fatalf("disabled task %+v: %v", result, err)
	}
}

func TestQQOfficialCleanupWithoutInstalledCredentials(t *testing.T) {
	s, db, now := qqFixture(t)
	// Align this fixture with the real maintenance clock even when a fixed
	// virtual test clock is used by queue behavior tests.
	*now = time.Now().UTC().AddDate(0, 0, -8)
	qqEnable(t, s)
	group := qqGroupFixture(t, s, 1)
	task := qqSubmitTest(t, s, group)
	ctx := t.Context()
	if err := s.locked(ctx, nil, func(tx *gorm.DB, _ *model.QQOfficialApp, at time.Time) error {
		_, err := s.repo.InsertEvent(ctx, tx, &model.QQOfficialEvent{AppID: s.appID, EventKey: qqHash("old-event"), Kind: "GROUP_ADD_ROBOT", GroupOpenID: group.OpenID, EventAt: at, CreatedAt: at})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	stopped, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := CleanupQQOfficial(stopped, db); err != nil {
		t.Fatal(err)
	}
	if n, err := s.repo.EventBacklog(ctx, nil, s.appID); err != nil || n != 0 {
		t.Fatalf("expired event still in backlog: %d %v", n, err)
	}
	result, _, err := s.Task(ctx, task.ID)
	if err != nil || result.Status != model.QQExpired || result.Text != "" {
		t.Fatalf("stale payload not erased: %s %v", result.Status, err)
	}
}
