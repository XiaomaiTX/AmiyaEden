package service

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"amiya-eden/internal/model"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Explicit test-only endpoints; never use the application's production configuration.
func TestQQBotPostgresRedisBoundaries(t *testing.T) {
	dsn := os.Getenv("QQBOT_TEST_POSTGRES")
	addr := os.Getenv("QQBOT_TEST_REDIS")
	if dsn == "" || addr == "" {
		t.Skip("set QQBOT_TEST_POSTGRES and QQBOT_TEST_REDIS for isolated integration boundaries")
	}
	db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	schema := "qqbot_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if e = db.Exec("CREATE SCHEMA " + schema).Error; e != nil {
		t.Fatal(e)
	}
	defer func() {
		if err := db.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Error(err)
		}
		sql, _ := db.DB()
		_ = sql.Close()
	}()
	scoped, e := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	sql, e := scoped.DB()
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = sql.Close() }()
	if e = scoped.AutoMigrate(&model.QQBotOperation{}, &model.QQBotAttempt{}, &model.QQBotInbox{}, &model.QQBotDelivery{}, &model.QQBotCircuit{}, &model.QQBotSetting{}); e != nil {
		t.Fatal(e)
	}
	r := redis.NewClient(&redis.Options{Addr: addr})
	defer func() { _ = r.Close() }()
	s := NewQQBotService(scoped, r)
	defer s.Close()
	token := "secret"
	if e = s.SetSetting(context.Background(), model.QQBotSetting{Enabled: true, BotID: 42, CIDRs: "127.0.0.1/32", IntervalMS: 100}, &token); e != nil {
		t.Fatal(e)
	}
	// Concurrent stable-key submission through real unique constraints.
	var wg sync.WaitGroup
	ids := make(chan string, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o, err := s.Submit(context.Background(), nil, "stable-postgres", 0, time.Now().Add(time.Hour), model.QQBotRequest{Kind: "group_info", GroupID: 8})
			if err != nil {
				t.Error(err)
				return
			}
			ids <- o.ID
		}()
	}
	wg.Wait()
	close(ids)
	unique := map[string]bool{}
	for id := range ids {
		unique[id] = true
	}
	if len(unique) != 1 {
		t.Fatal(unique)
	}
	claimed := make(chan string, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o, err := s.repo.Claim(context.Background(), time.Now())
			if err != nil {
				t.Error(err)
			}
			if o != nil {
				claimed <- o.ID
			}
		}()
	}
	wg.Wait()
	close(claimed)
	n := 0
	for range claimed {
		n++
	}
	if n != 1 {
		t.Fatalf("concurrent claims: %d", n)
	}
	// Global Lua admission atomicity, TTL recovery, and actual network failure.
	if reason, e := s.admit(context.Background(), 8, 100); e != nil || reason != "" {
		t.Fatal(reason, e)
	}
	if reason, e := s.admit(context.Background(), 9, 100); e != nil || reason != "rate_limited" {
		t.Fatal(reason, e)
	}
	time.Sleep(120 * time.Millisecond)
	if reason, e := s.admit(context.Background(), 9, 100); e != nil || reason != "" {
		t.Fatal(reason, e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if reason, e := s.admit(context.Background(), 8, 100); e != nil || reason != "redis_unavailable" {
		t.Fatal(reason, e)
	}
}
