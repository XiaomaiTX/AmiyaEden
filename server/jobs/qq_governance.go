package jobs

import (
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"
)

const (
	QQGovernanceReconcileTaskName   = "qq_group_governance_reconcile"
	QQGovernanceMaintenanceTaskName = "qq_governance_maintenance"
)

func registerQQGovernanceTasks(reg *taskregistry.Registry) {
	reg.Register(taskregistry.TaskDefinition{
		Name: QQGovernanceReconcileTaskName, Description: "创建 QQ 群治理成员巡检任务", Category: taskregistry.TaskCategorySystem,
		Type: taskregistry.TaskTypeRecurring, DefaultCron: "0 */15 * * * *",
		RunFunc: func(ctx context.Context) error {
			return service.DefaultQQGovernanceService().EnqueueScheduledReconciliations(ctx, 0)
		},
	})
	reg.Register(taskregistry.TaskDefinition{
		Name: QQGovernanceMaintenanceTaskName, Description: "维护 QQ 群治理任务、日志与保留期数据", Category: taskregistry.TaskCategorySystem,
		Type: taskregistry.TaskTypeRecurring, DefaultCron: "0 20 3 * * *",
		RunFunc: func(ctx context.Context) error {
			return service.DefaultQQGovernanceService().RunGovernanceMaintenance(ctx)
		},
	})
}
