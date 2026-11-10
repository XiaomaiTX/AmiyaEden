package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"
	"time"
)

func registerQQStructureAlert(reg *taskregistry.Registry) {
	reg.Register(taskregistry.TaskDefinition{Name: "qq_structure_alert", Description: "QQ 建筑业务预警", Category: taskregistry.TaskCategorySystem, Type: taskregistry.TaskTypeRecurring, DefaultCron: "0 0 * * * *", RunFunc: func(ctx context.Context) error {
		return service.NewQQStructureAlertService(global.DB, service.DefaultQQBotService().Notifications()).Scan(ctx, time.Now())
	}})
}
