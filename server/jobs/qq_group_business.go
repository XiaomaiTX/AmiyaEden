package jobs

import (
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"
	"time"
)

func registerQQGroupBusiness(reg *taskregistry.Registry) {
	reg.Register(taskregistry.TaskDefinition{Name: "qq_group_business", Description: "QQ 群业务复查", Category: taskregistry.TaskCategorySystem, Type: taskregistry.TaskTypeRecurring, DefaultCron: "@every 1m", RunFunc: func(ctx context.Context) error {
		return service.DefaultQQBotService().Groups().Recheck(ctx, time.Now())
	}})
}
