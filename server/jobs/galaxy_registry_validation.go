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
		Description: "Validate completed galaxy registry entries against bounty wallet journals",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "@every 1h",
		RunFunc: func(ctx context.Context) error {
			_ = ctx
			count, err := svc.ValidateCompletedEntries(200)
			if err != nil {
				global.Logger.Error("星系登记校验任务失败", zap.Error(err))
				return err
			}
			global.Logger.Info("星系登记校验任务完成", zap.Int("validated_count", count))
			return nil
		},
	})
}
