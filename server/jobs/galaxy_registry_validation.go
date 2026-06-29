package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"

	"go.uber.org/zap"
)

func registerGalaxyRegistryValidationTask(reg *taskregistry.Registry) {
	svc := service.NewGalaxyRegistryService()

	reg.Register(taskregistry.TaskDefinition{
		Name:        "galaxy_registry_validation",
		Description: "Auto-end overdue galaxy registry entries and settle pending entries through ESI wallet queue",
		Category:    taskregistry.TaskCategoryESI,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "@every 5m",
		RunFunc: func(ctx context.Context) error {
			ended, err := svc.AutoEndOverdueEntries(200)
			if err != nil {
				global.Logger.Error("星系登记自动下线任务失败", zap.Error(err))
				return err
			}
			queue := GetESIQueue()
			if queue == nil {
				global.Logger.Warn("星系登记结算任务跳过：ESI 队列未初始化")
				return nil
			}
			settled, err := svc.SettlePendingEntriesWithWalletRefresh(ctx, queue, 200)
			if err != nil {
				global.Logger.Error("星系登记结算任务失败", zap.Error(err))
				return err
			}
			global.Logger.Info("星系登记后台任务完成",
				zap.Int("auto_ended_count", ended),
				zap.Int("settled_count", settled),
			)
			return nil
		},
	})
}
