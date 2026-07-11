package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"

	"go.uber.org/zap"
)

const (
	StructureFuelRateSyncTaskName = "structure_fuel_rate_sync"
	// 每 10 天同步一次服务模块燃料率（ESI 公开端点，无需认证）。
	structureFuelRateSyncCron = "@every 240h"
)

// registerStructureFuelRateSyncTask 注册建筑服务模块燃料率同步任务。
//
// 任务从 ESI /universe/types/{id}/ 拉取各服务模块的 dogma 属性 2109
// （serviceModuleFuelAmount），刷新 structure_service_fuel_rate 映射表，
// 供军团建筑列表的「预计每小时消耗燃料块 / 到月底需补燃料」两列使用。
//
// 服务实例在 RunFunc 内惰性构造，避免注册阶段读取 global.Config（测试环境下可能为空）。
// 首次同步（映射表为空时）不在注册阶段触发；由 bootstrap 在 TaskService 就绪后
// 经任务生命周期（锁 + 执行历史 + 关停取消）触发，见 bootstrap.InitCron。
func registerStructureFuelRateSyncTask(reg *taskregistry.Registry) {
	reg.Register(taskregistry.TaskDefinition{
		Name:        StructureFuelRateSyncTaskName,
		Description: "同步建筑服务模块每小时燃料消耗率（ESI）",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: structureFuelRateSyncCron,
		RunFunc: func(ctx context.Context) error {
			svc := service.NewStructureFuelRateService()
			if err := svc.SyncFuelRates(ctx); err != nil {
				global.Logger.Error("structure fuel rate sync task failed", zap.Error(err))
				return err
			}
			return nil
		},
	})

	global.Logger.Info("registered structure fuel rate sync task", zap.String("task_name", StructureFuelRateSyncTaskName))
}

// StructureFuelRateSyncNeedsFirstRun 判断燃料率映射表是否为空（需要首次同步）。
// 由 bootstrap 在任务系统就绪后调用。对 global.Config/DB 缺失（如测试环境）返回 false，
// 不在此处 panic。
func StructureFuelRateSyncNeedsFirstRun() bool {
	if global.DB == nil {
		return false
	}
	empty, err := service.NewStructureFuelRateService().IsEmpty()
	if err != nil {
		return false
	}
	return empty
}
