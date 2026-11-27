package jobs

import (
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"
)

const QQOfficialStructureScanTaskName = "qq_official_structure_scan"

func registerQQOfficialStructureScan(reg *taskregistry.Registry) {
	reg.Register(taskregistry.TaskDefinition{
		Name:        QQOfficialStructureScanTaskName,
		Description: "扫描已授权军团的建筑快照，提交 QQ 官方提醒",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "@every 1h",
		RunFunc: func(ctx context.Context) error {
			qq, err := service.DefaultQQOfficialRuntime().Active(ctx, nil)
			if err != nil {
				return service.QQOfficialPublicError(err)
			}
			if qq == nil {
				return nil
			}
			return qq.ScanStructureAlerts(ctx)
		},
	})
}
