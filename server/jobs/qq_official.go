package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"
)

func registerQQOfficialCleanup(reg *taskregistry.Registry) {
	reg.Register(taskregistry.TaskDefinition{
		Name: "qq_official_cleanup", Description: "Remove expired QQ Official integration records",
		Category: taskregistry.TaskCategorySystem, Type: taskregistry.TaskTypeRecurring, DefaultCron: "0 17 * * * *",
		RunFunc: func(ctx context.Context) error { return service.CleanupQQOfficial(ctx, global.DB) },
	})
}
