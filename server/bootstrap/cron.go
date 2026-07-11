package bootstrap

import (
	"amiya-eden/global"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"amiya-eden/jobs"
	"amiya-eden/pkg/background"
	"context"
	"sync"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var nopCronLogger = zap.NewNop()

// InitCron 初始化并启动定时任务调度器
func InitCron() *service.TaskService {
	c := cron.New(
		cron.WithSeconds(), // 支持秒级精度
		cron.WithChain(cron.Recover(cron.DefaultLogger)), // panic 恢复
		cron.WithLogger(newCronLogger()),
	)

	reg := taskregistry.New()
	jobs.RegisterAll(reg)

	taskRepo := repository.NewTaskRepository()
	schedules, err := taskRepo.ListAllSchedules()
	if err != nil {
		cronZapLogger().Warn("[Cron] 加载任务调度配置失败，使用默认值", zap.Error(err))
	}
	scheduleMap := make(map[string]string, len(schedules))
	for _, schedule := range schedules {
		scheduleMap[schedule.TaskName] = schedule.CronExpr
	}

	var entryMu sync.Mutex
	entryIDs := make(map[string]cron.EntryID)
	var taskSvc *service.TaskService
	rescheduleTask := func(taskName, cronExpr string) error {
		entryMu.Lock()
		defer entryMu.Unlock()

		if oldID, ok := entryIDs[taskName]; ok {
			c.Remove(oldID)
		}

		newID, err := c.AddFunc(cronExpr, func() {
			taskSvc.RunTaskFromCron(taskName)
		})
		if err != nil {
			cronZapLogger().Error("[Cron] 重新调度任务失败", zap.String("task", taskName), zap.Error(err))
			return err
		}

		entryIDs[taskName] = newID
		cronZapLogger().Info("[Cron] 重新调度任务成功", zap.String("task", taskName), zap.String("cron", cronExpr))
		return nil
	}
	taskSvc = service.NewTaskService(reg, taskRepo, rescheduleTask)

	for _, definition := range reg.All() {
		if definition.Type != taskregistry.TaskTypeRecurring || definition.DefaultCron == "" {
			continue
		}

		cronExpr := definition.DefaultCron
		if override, ok := scheduleMap[definition.Name]; ok {
			cronExpr = override
		}

		taskName := definition.Name
		entryID, err := c.AddFunc(cronExpr, func() {
			taskSvc.RunTaskFromCron(taskName)
		})
		if err != nil {
			cronZapLogger().Error("[Cron] 注册任务失败", zap.String("task", taskName), zap.Error(err))
			continue
		}
		entryIDs[taskName] = entryID
		cronZapLogger().Info("[Cron] 注册任务成功", zap.String("task", taskName), zap.String("cron", cronExpr))
	}

	c.Start()
	global.Cron = c
	cronZapLogger().Info("定时任务调度器已启动")

	// 启动时触发需要首次补跑的任务（经任务生命周期：锁 + 执行历史 + 关停取消）。
	triggerStartupOneShotTasks(taskSvc)

	return taskSvc
}

// triggerStartupOneShotTasks 在任务系统就绪后触发需要首次执行的任务。
// 仅当对应数据为空时触发，避免每次启动都跑。通过后台任务管理器派发，
// 确保进程关停时能取消并等待。
func triggerStartupOneShotTasks(taskSvc *service.TaskService) {
	// 燃料率映射表为空时，首次同步 structure_fuel_rate_sync。
	if jobs.StructureFuelRateSyncNeedsFirstRun() {
		taskName := jobs.StructureFuelRateSyncTaskName
		if err := background.RunOrSchedule(global.BackgroundContext(), global.EnsureBackgroundTaskManager(), "structure_fuel_rate_sync_initial", func(ctx context.Context) error {
			return taskSvc.RunTask(ctx, taskName, nil)
		}); err != nil {
			cronZapLogger().Warn("[Cron] 触发燃料率首次同步失败", zap.String("task", taskName), zap.Error(err))
		}
	}
}

// cronLogger 适配 zap 到 cron.Logger 接口
type cronLogger struct{}

func newCronLogger() cron.Logger {
	return &cronLogger{}
}

func (l *cronLogger) Info(msg string, keysAndValues ...interface{}) {
	cronZapLogger().Sugar().Infow("[Cron] "+msg, keysAndValues...)
}

func (l *cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	cronZapLogger().Sugar().Errorw("[Cron] "+msg, append([]interface{}{"error", err}, keysAndValues...)...)
}

func cronZapLogger() *zap.Logger {
	if logger := global.CurrentLogger(); logger != nil {
		return logger
	}
	return nopCronLogger
}
