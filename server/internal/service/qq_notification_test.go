package service

import (
	"amiya-eden/internal/model"
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"testing"
	"time"
)

func notificationTestService(t *testing.T) (*QQBotService, *QQNotificationService, *gorm.DB) {
	t.Helper()
	s, db, _ := qqbotTestService(t)
	if err := db.AutoMigrate(&model.QQNotificationChannel{}, &model.QQNotification{}, &model.QQNotificationTarget{}); err != nil {
		t.Fatal(err)
	}
	n := NewQQNotificationService(db, s)
	s.notifications = n
	if err := s.RegisterConsumer("result", "notifications", n.ConsumeResult); err != nil {
		t.Fatal(err)
	}
	if err := n.Configure(context.Background(), "general", true, []int64{8, 9}); err != nil {
		t.Fatal(err)
	}
	return s, n, db
}
func TestNotificationAtomicIdempotentTargets(t *testing.T) {
	_, n, db := notificationTestService(t)
	ctx := context.Background()
	expires := time.Now().Add(time.Hour)
	submit := func(tx *gorm.DB, text string) ([]model.QQNotificationTarget, error) {
		return n.Submit(ctx, tx, "general", "stable", "business-1", text, expires)
	}
	sentinel := errors.New("rollback")
	if err := db.Transaction(func(tx *gorm.DB) error {
		if _, err := submit(tx, "hello"); err != nil {
			return err
		}
		return sentinel
	}); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 0 {
		t.Fatal("rollback leaked operations")
	}
	a, err := submit(nil, "hello")
	if err != nil {
		t.Fatal(err)
	}
	b, err := submit(nil, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 2 || len(b) != 2 || a[0].OperationID != b[0].OperationID {
		t.Fatal("duplicate intent")
	}
	if _, err = submit(nil, "changed"); err == nil {
		t.Fatal("conflicting key accepted")
	}
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 2 {
		t.Fatal(count)
	}
	if err = n.Configure(ctx, "general", true, []int64{10}); err != nil {
		t.Fatal(err)
	}
	b, err = submit(nil, "hello")
	if err != nil || b[0].GroupID != 8 {
		t.Fatal("destinations not frozen", err)
	}
}
func TestNotificationResultConsumerAndDisable(t *testing.T) {
	s, n, db := notificationTestService(t)
	ctx := context.Background()
	targets, err := n.Submit(ctx, nil, "general", "result", "business-2", "hello", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	complete := func(id, status string) {
		t.Helper()
		raw, _ := json.Marshal(model.QQBotOperation{ID: id, BusinessKey: "notification:test", Status: status})
		if e := db.Transaction(func(tx *gorm.DB) error { return n.ConsumeResult(ctx, tx, string(raw)) }); e != nil {
			t.Fatal(e)
		}
	}
	complete(targets[0].OperationID, "succeeded")
	complete(targets[0].OperationID, "succeeded")
	complete(targets[1].OperationID, "expired")
	rows, err := n.Targets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	states := map[int64]string{}
	for _, r := range rows {
		states[r.GroupID] = r.Status
	}
	if states[8] != "confirmed" || states[9] != "expired" {
		t.Fatal(states)
	}
	if err = n.Configure(ctx, "general", false, nil); err != nil {
		t.Fatal(err)
	}
	if err = db.Transaction(func(tx *gorm.DB) error {
		valid, e := s.notifications.valid(tx, targets[0].OperationID)
		if valid {
			t.Fatal("disabled intent valid")
		}
		return e
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = n.Submit(ctx, nil, "general", "new", "business-3", "hello", time.Now().Add(time.Hour)); err == nil {
		t.Fatal("disabled submission")
	}
}

func TestNotificationExpiredIntentUsesDurableResult(t *testing.T) {
	s, n, db := notificationTestService(t)
	ctx := context.Background()
	expiry := time.Now().Add(time.Minute)
	rows, err := n.Submit(ctx, nil, "general", "expiry", "business-expiry", "hello", expiry)
	if err != nil {
		t.Fatal(err)
	}
	for range rows {
		if err = s.RunOnce(ctx, expiry.Add(time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	before, err := n.Targets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range before {
		if r.Status != "pending" {
			t.Fatal("enqueue or terminal operation marked business result before consumption")
		}
	}
	for range rows {
		if err = s.consumeOnce(ctx, s.consumers[0]); err != nil {
			t.Fatal(err)
		}
	}
	after, err := n.Targets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range after {
		if r.Status != "expired" {
			t.Fatal(r.Status)
		}
	}
	var count int64
	db.Model(&model.QQBotAttempt{}).Count(&count)
	if count != 0 {
		t.Fatal("expired intent attempted")
	}
}

func TestNotificationDisablePersistsAcrossReenable(t *testing.T) {
	s, n, _ := notificationTestService(t)
	ctx := context.Background()
	rows, err := n.Submit(ctx, nil, "general", "disable", "business-disable", "hello", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err = n.Configure(ctx, "general", false, nil); err != nil {
		t.Fatal(err)
	}
	if err = n.Configure(ctx, "general", true, []int64{8, 9}); err != nil {
		t.Fatal(err)
	}
	for range rows {
		if err = s.RunOnce(ctx, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range rows {
		op, e := s.GetOperation(ctx, r.OperationID)
		if e != nil || op.Status != "cancelled" {
			t.Fatal(op.Status, e)
		}
	}
}
