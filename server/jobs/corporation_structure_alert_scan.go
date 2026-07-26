package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"

	"go.uber.org/zap"
)

const (
	CorporationStructureAlertScanTaskName = "corporation_structure_alert_scan"
	corporationStructureAlertScanCron     = "0 0 * * * *"
)

func registerCorporationStructureAlertScanTask(reg *taskregistry.Registry) {
	reg.Register(taskregistry.TaskDefinition{
		Name:        CorporationStructureAlertScanTaskName,
		Description: "巡查军团建筑燃料与增强状态并发送 QQ 预警",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: corporationStructureAlertScanCron,
		RunFunc: func(ctx context.Context) error {
			if err := service.NewCorporationStructureService().RunAlertScan(ctx); err != nil {
				global.Logger.Error("corporation structure alert scan task failed", zap.Error(err))
				return err
			}
			return nil
		},
	})
}
