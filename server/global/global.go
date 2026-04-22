package global

import (
	"amiya-eden/config"
	"amiya-eden/pkg/background"
	"context"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	// Config 全局配置
	Config *config.Config

	// Logger 全局日志
	Logger *zap.Logger

	// DB 全局数据库连接
	DB *gorm.DB

	// Redis 全局 Redis 客户端
	Redis *redis.Client

	// Cron 全局定时任务调度器
	Cron *cron.Cron

	loggerMu                sync.RWMutex
	backgroundTaskManager   *background.Manager
	backgroundTaskManagerMu sync.RWMutex
)

func CurrentLogger() *zap.Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return Logger
}

func SetLogger(logger *zap.Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	Logger = logger
}

func BackgroundTaskManager() *background.Manager {
	backgroundTaskManagerMu.RLock()
	defer backgroundTaskManagerMu.RUnlock()
	return backgroundTaskManager
}

func SetBackgroundTaskManager(manager *background.Manager) {
	backgroundTaskManagerMu.Lock()
	defer backgroundTaskManagerMu.Unlock()
	backgroundTaskManager = manager
}

func EnsureBackgroundTaskManager() *background.Manager {
	backgroundTaskManagerMu.Lock()
	defer backgroundTaskManagerMu.Unlock()

	if backgroundTaskManager == nil {
		backgroundTaskManager = background.New(context.Background(), func() *zap.Logger {
			return CurrentLogger()
		})
	}

	return backgroundTaskManager
}

func BackgroundContext() context.Context {
	if manager := BackgroundTaskManager(); manager != nil {
		return manager.Context()
	}
	return context.Background()
}
