package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"
	"time"

	"go.uber.org/zap"
)

// registerAlliancePAPTasks 注册联盟 PAP 任务定义:
//   - 每小时整点刷新当月数据
//   - 每月第一天 01:00 补拉上月数据并归档
func registerAlliancePAPTasks(reg *taskregistry.Registry) {
	svc := service.NewAlliancePAPService()
	papRepo := repository.NewAlliancePAPRepository()

	reg.Register(taskregistry.TaskDefinition{
		Name:        "alliance_pap_hourly",
		Description: "Refresh current-month alliance PAP data",
		Category:    taskregistry.TaskCategoryOperation,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 0 * * * *",
		RunFunc: func(ctx context.Context) error {
			now := time.Now()
			global.Logger.Info("开始联盟 PAP 小时刷新", zap.Int("year", now.Year()), zap.Int("month", int(now.Month())))
			svc.FetchAllUsers(now.Year(), int(now.Month()))
			return nil
		},
	})
	global.Logger.Info("注册联盟 PAP 小时任务成功", zap.String("task_name", "alliance_pap_hourly"))

	reg.Register(taskregistry.TaskDefinition{
		Name:        "alliance_pap_archive",
		Description: "Archive prior-month alliance PAP data",
		Category:    taskregistry.TaskCategoryOperation,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 0 1 1 * *",
		RunFunc: func(ctx context.Context) error {
			now := time.Now()
			lastMonth := now.AddDate(0, -1, 0)
			year := lastMonth.Year()
			month := int(lastMonth.Month())

			global.Logger.Info("开始联盟 PAP 月度归档", zap.Int("year", year), zap.Int("month", month))
			svc.FetchAllUsers(year, month)

			if err := papRepo.MarkArchived(year, month); err != nil {
				global.Logger.Error("联盟 PAP 归档标记失败", zap.Error(err))
				return err
			}
			global.Logger.Info("联盟 PAP 月度归档完成", zap.Int("year", year), zap.Int("month", month))
			return nil
		},
	})
	global.Logger.Info("注册联盟 PAP 月度归档任务成功", zap.String("task_name", "alliance_pap_archive"))
}
