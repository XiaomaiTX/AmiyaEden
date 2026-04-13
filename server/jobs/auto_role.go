package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"

	"go.uber.org/zap"
)

// registerAutoRoleSyncTask 注册自动权限同步任务。
func registerAutoRoleSyncTask(reg *taskregistry.Registry) {
	reg.Register(taskregistry.TaskDefinition{
		Name:        "auto_role_sync",
		Description: "Synchronize auto roles for all users",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 2/10 * * * *",
		RunFunc: func(ctx context.Context) error {
			autoRoleSyncTask(ctx)
			return nil
		},
	})
	global.Logger.Info("注册自动权限同步任务成功", zap.String("task_name", "auto_role_sync"))
}

// autoRoleSyncTask 根据 ESI 军团职权 + 头衔映射，自动同步所有用户权限
func autoRoleSyncTask(ctx context.Context) {
	autoRoleSvc := service.NewAutoRoleService()
	autoRoleSvc.SyncAllUsersAutoRoles(ctx)
}
