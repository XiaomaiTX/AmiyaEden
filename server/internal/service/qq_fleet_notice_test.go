package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"errors"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestFleetNoticeAtomicAndStable(t *testing.T) {
	_, n, db := notificationTestService(t)
	ctx := context.Background()
	if err := db.AutoMigrate(&model.Fleet{}); err != nil {
		t.Fatal(err)
	}
	if err := n.Configure(ctx, "fleet", true, []int64{8, 9}); err != nil {
		t.Fatal(err)
	}
	fleet := model.Fleet{ID: "fleet-1", Title: "Fleet", CreatedAt: time.Now().UTC().Truncate(time.Microsecond), StartAt: time.Now().Add(time.Hour), EndAt: time.Now().Add(2 * time.Hour)}
	rollback := errors.New("rollback")
	err := db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(&fleet).Error; e != nil {
			return e
		}
		if e := n.SubmitFleetCreated(ctx, tx, &fleet); e != nil {
			return e
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 0 {
		t.Fatal("notification leaked rollback")
	}
	if err = db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(&fleet).Error; e != nil {
			return e
		}
		return n.SubmitFleetCreated(ctx, tx, &fleet)
	}); err != nil {
		t.Fatal(err)
	}
	if err = db.Transaction(func(tx *gorm.DB) error { return n.SubmitFleetCreated(ctx, tx, &fleet) }); err != nil {
		t.Fatal(err)
	}
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 2 {
		t.Fatal(count)
	}
	targets, err := n.Targets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Transaction(func(tx *gorm.DB) error {
		return n.repo.Complete(tx, targets[0].OperationID, "failed", "platform rejected")
	}); err != nil {
		t.Fatal(err)
	}
	db.Model(&model.Fleet{}).Count(&count)
	if count != 1 {
		t.Fatal("notification failure undid fleet")
	}
	var op model.QQBotOperation
	db.First(&op)
	if !op.ExpiresAt.Equal(fleet.CreatedAt.Add(30 * time.Minute)) {
		t.Fatal(op.ExpiresAt)
	}
}
func TestFleetNoticeExpiredAndDisabled(t *testing.T) {
	_, n, db := notificationTestService(t)
	ctx := context.Background()
	fleet := model.Fleet{ID: "expired", CreatedAt: time.Now(), StartAt: time.Now().Add(-time.Minute)}
	if err := n.Configure(ctx, "fleet", true, []int64{8}); err != nil {
		t.Fatal(err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return n.SubmitFleetCreated(ctx, tx, &fleet) }); err != nil {
		t.Fatal(err)
	}
	fleet.StartAt = time.Now().Add(time.Hour)
	if err := n.Configure(ctx, "fleet", false, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return n.SubmitFleetCreated(ctx, tx, &fleet) }); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 0 {
		t.Fatal("expired/disabled fleet queued")
	}
}

func TestFleetCreationRollsBackWhenIntentCannotCommit(t *testing.T) {
	_, n, db := notificationTestService(t)
	ctx := context.Background()
	old := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = old })
	if err := db.AutoMigrate(&model.Fleet{}); err != nil {
		t.Fatal(err)
	}
	if err := n.Configure(ctx, "fleet", true, []int64{8}); err != nil {
		t.Fatal(err)
	}
	svc := &FleetService{repo: repository.NewFleetRepository(), notifications: n}
	fleet := model.Fleet{ID: "atomic", Title: "valid", StartAt: time.Now().Add(time.Hour), EndAt: time.Now().Add(2 * time.Hour)}
	if err := db.Model(&model.QQNotificationChannel{}).Where("name = ?", "fleet").Update("group_ids", "[]").Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.createFleetWithNotice(ctx, &fleet, true); err == nil {
		t.Fatal("invalid notification committed")
	}
	var count int64
	db.Model(&model.Fleet{}).Count(&count)
	if count != 0 {
		t.Fatal("fleet survived failed transaction")
	}
	if err := n.Configure(ctx, "fleet", true, []int64{8}); err != nil {
		t.Fatal(err)
	}
	if err := svc.createFleetWithNotice(ctx, &fleet, true); err != nil {
		t.Fatal(err)
	}
	db.Model(&model.Fleet{}).Count(&count)
	if count != 1 {
		t.Fatal(count)
	}
}
