package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// registerSdeJob 注册每日 20:00 SDE 检查更新任务
func registerSdeJob(c *cron.Cron) {
	global.Logger.Info("SDE 定时任务已禁用")
}

// SdeCheckOnStartup 启动时执行一次 SDE 检查更新（供 main 调用）
func SdeCheckOnStartup() {
	global.Logger.Info("启动时 SDE 自动检查已禁用")
}

// sdeCheckUpdateTask SDE 检查更新任务入口
func sdeCheckUpdateTask() {
	global.Logger.Info("[定时任务] SDE 检查更新中...")
	svc := service.NewSdeService()
	updated, version, err := svc.CheckAndUpdate()
	if err != nil {
		global.Logger.Error("[定时任务] SDE 更新失败", zap.Error(err))
		return
	}
	if updated {
		global.Logger.Info("[定时任务] SDE 更新完成", zap.String("version", version))
	} else {
		global.Logger.Info("[定时任务] SDE 已是最新版本", zap.String("version", version))
	}
}
