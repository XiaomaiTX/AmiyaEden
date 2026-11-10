package service

import (
	"amiya-eden/internal/model"
	"context"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestQQBusinessPostgresConcurrency(t *testing.T) {
	dsn := os.Getenv("QQBOT_TEST_POSTGRES")
	if dsn == "" {
		t.Skip("set isolated QQBOT_TEST_POSTGRES")
	}
	root, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	sql, e := root.DB()
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = sql.Close() }()
	schema := "qqbusiness_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if e = root.Exec("CREATE SCHEMA " + schema).Error; e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := root.Exec("DROP SCHEMA " + schema + " CASCADE").Error; e != nil {
			t.Error(e)
		}
	}()
	db, e := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	conn, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = conn.Close() }()
	if e = db.AutoMigrate(&model.QQBotOperation{}, &model.QQNotificationChannel{}, &model.QQNotification{}, &model.QQNotificationTarget{}, &model.QQGroupRule{}, &model.QQGroupApplication{}, &model.QQGroupMember{}, &model.QQGroupAppliedEvent{}, &model.QQGroupInspection{}, &model.QQGroupInspectionItem{}, &model.User{}, &model.EveCharacter{}, &model.UserRole{}); e != nil {
		t.Fatal(e)
	}
	d := NewQQBotService(db, nil)
	n := NewQQNotificationService(db, d)
	g := NewQQGroupBusinessService(db, d)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if e = n.Configure(ctx, "general", true, []int64{8, 9}); e != nil {
		t.Fatal(e)
	}
	expiry := time.Now().Add(time.Hour)
	errs := make(chan error, 8)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := n.Submit(ctx, nil, "general", "concurrent", "business-concurrent", "hello", expiry)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 2 {
		t.Fatal("concurrent intent duplicated", count)
	}
	seedQQQualification(t, db)
	rule := model.QQGroupRule{GroupID: 8, InspectionEnabled: true, MismatchConfirmations: 3, ObservationHours: 24, CorporationIDs: []int64{300}}
	if e = g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	run := inspectionRound(t, g, db, time.Now(), true)
	errs = make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- g.judgeMember(ctx, run.ID, 9, time.Now()) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var m model.QQGroupMember
	if e = db.First(&m, "group_id = ? AND qq = ?", 8, 9).Error; e != nil {
		t.Fatal(e)
	}
	if m.Confirmations != 1 {
		t.Fatal("concurrent round counted more than once", m.Confirmations)
	}
}
