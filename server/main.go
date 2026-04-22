package main

import (
	"amiya-eden/bootstrap"
	"amiya-eden/global"
	"amiya-eden/pkg/background"
	"amiya-eden/pkg/jwt"
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	// 初始化配置
	bootstrap.InitConfig()

	// 初始化日志
	bootstrap.InitLogger()
	global.SetBackgroundTaskManager(background.New(context.Background(), func() *zap.Logger {
		return global.Logger
	}))

	// 初始化 JWT 密钥
	jwt.Init(global.Config.JWT.Secret)

	// 初始化数据库
	bootstrap.InitDB()

	// 初始化 Redis
	bootstrap.InitRedis()

	// 初始化定时任务
	taskSvc := bootstrap.InitCron()

	// 将 ESI 任务模块的 scope 注册到 SSO 服务
	bootstrap.InitScopes()

	// 初始化路由
	r := bootstrap.InitRouter(taskSvc)

	// 启动 HTTP 服务
	srv := &http.Server{
		Addr:    ":" + global.Config.Server.Port,
		Handler: r,
	}

	go func() {
		global.Logger.Info("服务启动", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			global.Logger.Fatal("服务启动失败", zap.Error(err))
		}
	}()

	// 优雅关停
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	global.Logger.Info("正在关闭服务...")
	shutdownServer(srv, stopCronScheduler(), global.BackgroundTaskManager(), 5*time.Second)
	global.Logger.Info("服务已退出")
}

func stopCronScheduler() context.Context {
	if global.Cron == nil {
		return nil
	}
	return global.Cron.Stop()
}

func shutdownServer(srv *http.Server, cronStopCtx context.Context, backgroundTaskManager *background.Manager, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	shutdownDone := make(chan error, 1)
	go func() {
		if srv == nil {
			shutdownDone <- nil
			return
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			shutdownDone <- context.DeadlineExceeded
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), remaining)
		defer cancel()
		shutdownDone <- srv.Shutdown(ctx)
	}()

	if err := <-shutdownDone; err != nil && !errors.Is(err, http.ErrServerClosed) {
		global.Logger.Error("服务关闭异常", zap.Error(err))
	}
	if err := waitForContext(cronStopCtx, time.Until(deadline)); err != nil {
		global.Logger.Warn("定时任务关闭超时", zap.Error(err))
	}

	if backgroundTaskManager != nil {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			remaining = time.Nanosecond
		}
		if err := backgroundTaskManager.Shutdown(remaining); err != nil {
			global.Logger.Warn("后台任务关闭超时", zap.Error(err))
		}
	}
}

func waitForContext(ctx context.Context, timeout time.Duration) error {
	if ctx == nil {
		return nil
	}
	if timeout <= 0 {
		return context.DeadlineExceeded
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil
	case <-timer.C:
		return context.DeadlineExceeded
	}
}
