package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/service/internal/qqapi"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestQQOfficialQuotaSurvivesRestartAndSkipsBlockedGroup(t *testing.T) {
	s, db, now := qqFixture(t)
	qqEnable(t, s)
	group := qqGroupFixture(t, s, 1)
	other := qqGroupFixture(t, s, 2)
	blocked := qqSubmitTest(t, s, group)
	ready := qqSubmitTest(t, s, other)
	ctx := t.Context()
	if err := s.locked(ctx, nil, func(tx *gorm.DB, app *model.QQOfficialApp, at time.Time) error {
		app.LastRegularGroup = other.ID
		if err := s.repo.SaveApp(ctx, tx, app); err != nil {
			return err
		}
		for i := range 12 {
			row := model.QQOfficialAttempt{AppID: s.appID, ID: uuid.NewString(), GroupID: group.ID, Kind: qqapi.SendText, AdmittedAt: at.Add(time.Duration(-50+i) * time.Second)}
			if err := s.repo.SaveAttempt(ctx, tx, &row); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// A separate worker has no in-memory quota data or RR cursor.
	restarted := NewQQOfficialService(db, s.appID, "sample-secret", true)
	restarted.now = s.now
	p, err := restarted.claim(ctx, qqToken(*now), 0)
	if err != nil || p == nil || p.attempt.TaskID != ready.ID {
		t.Fatalf("expected another group, got %+v %v", p, err)
	}
	row, _, err := s.Task(ctx, blocked.ID)
	if err != nil || row.Reason != "rate_limited" || row.NextRunAt.Before(now.Add(15*time.Second)) {
		t.Fatalf("blocked task %+v %v", row, err)
	}
}

func TestQQOfficialGlobalAndDailyQuotasUseAllAttempts(t *testing.T) {
	s, _, now := qqFixture(t)
	group := qqGroupFixture(t, s, 1)
	ctx := t.Context()
	if err := s.locked(ctx, nil, func(tx *gorm.DB, _ *model.QQOfficialApp, at time.Time) error {
		for i := range 800 {
			row := model.QQOfficialAttempt{AppID: s.appID, ID: uuid.NewString(), GroupID: group.ID, Kind: qqapi.SendText, AdmittedAt: at.Add(time.Duration(i-850) * time.Second), Outcome: "lease_expired"}
			if err := s.repo.SaveAttempt(ctx, tx, &row); err != nil {
				return err
			}
		}
		for i := range 48 {
			row := model.QQOfficialAttempt{AppID: s.appID, ID: uuid.NewString(), Kind: qqTokenKind, AdmittedAt: at.Add(time.Duration(i-50) * time.Second)}
			if err := s.repo.SaveAttempt(ctx, tx, &row); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	global, err := s.window(ctx, nil, "", "", *now, 48, time.Minute)
	if err != nil || !global.Equal(now.Add(15*time.Second)) {
		t.Fatalf("global %v %v", global, err)
	}
	daily, reason, err := s.nextEligible(ctx, nil, &model.QQOfficialApp{}, nil, qqapi.SendText, group.ID, *now)
	expected := now.Add(24*time.Hour - 845*time.Second)
	if err != nil || reason != "rate_limited" || !daily.Equal(expected) {
		t.Fatalf("daily %v %s %v want %v", daily, reason, err, expected)
	}
}

func TestQQOfficialCircuitAndFairness(t *testing.T) {
	s, _, now := qqFixture(t)
	ctx := t.Context()
	if err := s.locked(ctx, nil, func(tx *gorm.DB, _ *model.QQOfficialApp, at time.Time) error {
		for range 3 {
			if err := s.updateCooldown(ctx, tx, model.QQOfficialAttempt{Kind: qqapi.JoinList}, &qqapi.OperationError{Kind: qqapi.TemporaryNotApplied}, at); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	view, err := s.Status(ctx)
	if err != nil || len(view.Cooldowns) != 1 || view.Cooldowns[0].Scope != "read" || !view.Cooldowns[0].Until.Equal(now.Add(time.Minute)) {
		t.Fatalf("circuit %+v %v", view, err)
	}
	rows := make([]model.QQOfficialTask, 0, 8)
	for i := range 4 {
		id := fmt.Sprint(i)
		rows = append(rows, model.QQOfficialTask{GroupID: id, Urgent: true}, model.QQOfficialTask{GroupID: id})
	}
	qqOrderCandidates(rows, &model.QQOfficialApp{LastRegularGroup: "1", LastUrgentGroup: "2", UrgentStreak: 3})
	if rows[0].Urgent || rows[0].GroupID != "2" || !rows[4].Urgent || rows[4].GroupID != "3" {
		t.Fatalf("fair order %s/%t %s/%t", rows[0].GroupID, rows[0].Urgent, rows[4].GroupID, rows[4].Urgent)
	}
	if err = s.locked(ctx, nil, func(tx *gorm.DB, _ *model.QQOfficialApp, at time.Time) error {
		return s.updateCooldown(ctx, tx, model.QQOfficialAttempt{Kind: qqapi.JoinList}, nil, at)
	}); err != nil {
		t.Fatal(err)
	}
	view, err = s.Status(ctx)
	if err != nil || len(view.Cooldowns) != 0 {
		t.Fatalf("clear %+v %v", view, err)
	}
}
