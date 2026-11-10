package repository

import (
	"context"
	"testing"
	"time"

	"amiya-eden/internal/model"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestQQBotClaimFencingAndPhysicalRetention(t *testing.T) {
	db, e := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	sql, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = sql.Close() }()
	if e = db.AutoMigrate(&model.QQBotOperation{}, &model.QQBotAttempt{}, &model.QQBotDelivery{}, &model.QQBotInbox{}); e != nil {
		t.Fatal(e)
	}
	r := NewQQBotRepository(db)
	now := time.Now()
	old := now.Add(-100 * 24 * time.Hour)
	o := model.QQBotOperation{ID: "old", BusinessKey: "old", Status: "queued", Priority: 0, CreatedAt: now.Add(-3 * time.Minute), AvailableAt: now, ExpiresAt: now.Add(time.Hour)}
	high := model.QQBotOperation{ID: "high", BusinessKey: "high", Status: "queued", Priority: 2, CreatedAt: now, AvailableAt: now, ExpiresAt: now.Add(time.Hour)}
	if e = db.Create(&o).Error; e != nil {
		t.Fatal(e)
	}
	if e = db.Create(&high).Error; e != nil {
		t.Fatal(e)
	}
	claimed, e := r.Claim(context.Background(), now)
	if e != nil || claimed == nil || claimed.ID != "old" {
		t.Fatal(claimed, e)
	}
	stale := *claimed
	stale.LeaseToken = "wrong"
	stale.Status = "succeeded"
	if e = r.UpdateOwned(db, &stale); e == nil {
		t.Fatal("stale lease updated operation")
	}
	claimed.Status = "succeeded"
	claimed.TerminalAt = &old
	if e = r.UpdateOwned(db, claimed); e != nil {
		t.Fatal(e)
	}
	if e = db.Create(&model.QQBotAttempt{ID: "attempt", OperationID: o.ID, StartedAt: old}).Error; e != nil {
		t.Fatal(e)
	}
	if e = db.Create(&model.QQBotInbox{ID: "inbox", CreatedAt: old}).Error; e != nil {
		t.Fatal(e)
	}
	if e = db.Create(&model.QQBotDelivery{ID: "delivery", Topic: "event", ItemID: "inbox", Consumer: "c", CreatedAt: old, Status: "dead"}).Error; e != nil {
		t.Fatal(e)
	}
	if e = r.Cleanup(context.Background(), now); e != nil {
		t.Fatal(e)
	}
	var count int64
	for _, table := range []interface{}{&model.QQBotAttempt{}, &model.QQBotDelivery{}, &model.QQBotInbox{}} {
		if e = db.Model(table).Count(&count).Error; e != nil || count != 0 {
			t.Fatal(count, e)
		}
	}
	if e = db.Model(&model.QQBotOperation{}).Count(&count).Error; e != nil || count != 1 {
		t.Fatal(count, e)
	}
}
