package bootstrap

import (
	"amiya-eden/config"
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCronLoggerHandlesNilGlobalLogger(t *testing.T) {
	oldLogger := global.Logger
	global.Logger = nil
	t.Cleanup(func() {
		global.Logger = oldLogger
	})

	logger := newCronLogger()
	logger.Info("cron info")
	logger.Error(errors.New("boom"), "cron error")
}

func newCronBootstrapTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:cron_bootstrap_test?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.TaskSchedule{}, &model.TaskExecution{}, &model.Fleet{}, &model.EveCharacter{}); err != nil {
		t.Fatalf("auto migrate cron bootstrap models: %v", err)
	}
	return db
}

func TestInitCronReturnsTaskServiceAndSchedulesRecurringTasks(t *testing.T) {
	oldConfig := global.Config
	oldDB := global.DB
	oldLogger := global.Logger
	oldCron := global.Cron
	oldRescheduleFn := service.RescheduleFn

	db := newCronBootstrapTestDB(t)
	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.DB = db
	global.Logger = zap.NewNop()
	global.Cron = nil
	service.RescheduleFn = nil

	t.Cleanup(func() {
		if global.Cron != nil {
			ctx := global.Cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(2 * time.Second):
			}
		}
		global.Config = oldConfig
		global.DB = oldDB
		global.Logger = oldLogger
		global.Cron = oldCron
		service.RescheduleFn = oldRescheduleFn
	})

	if err := db.Create(&model.TaskSchedule{TaskName: "mentor_reward", CronExpr: "0 30 3 * * *", UpdatedBy: 1}).Error; err != nil {
		t.Fatalf("seed task schedule override: %v", err)
	}

	taskSvc := InitCron()
	if taskSvc == nil {
		t.Fatal("expected InitCron to return a task service")
	}
	if global.Cron == nil {
		t.Fatal("expected InitCron to initialize the global cron scheduler")
	}
	if service.RescheduleFn == nil {
		t.Fatal("expected InitCron to configure service.RescheduleFn")
	}

	entries := global.Cron.Entries()
	if len(entries) != 8 {
		t.Fatalf("scheduled recurring task count = %d, want 8", len(entries))
	}

	tasks, err := taskSvc.GetTasks()
	if err != nil {
		t.Fatalf("GetTasks returned error: %v", err)
	}

	var mentorRewardCron string
	for _, task := range tasks {
		if task.Name == "mentor_reward" {
			mentorRewardCron = task.CronExpr
			break
		}
	}
	if mentorRewardCron != "0 30 3 * * *" {
		t.Fatalf("mentor_reward cron = %q, want %q", mentorRewardCron, "0 30 3 * * *")
	}

	if err := service.RescheduleFn("mentor_reward", "0 0 4 * * *"); err != nil {
		t.Fatalf("RescheduleFn returned error: %v", err)
	}
}
