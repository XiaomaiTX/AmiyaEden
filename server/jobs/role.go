package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"amiya-eden/internal/utils"
	"context"

	"go.uber.org/zap"
)

func registerCorpAccessCheckTask(reg *taskregistry.Registry) {
	reg.Register(taskregistry.TaskDefinition{
		Name:        "corp_access_check",
		Description: "Check corporation access and adjust roles",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 0/5 * * * *",
		RunFunc: func(ctx context.Context) error {
			roleCheckTask()
			return nil
		},
	})
	global.Logger.Info("注册职权检查任务成功", zap.String("task_name", "corp_access_check"))
}

// roleCheckTask 遍历所有用户，根据军团准入列表调整用户权限
func roleCheckTask() {
	// 未配置允许军团列表时跳过
	allowCorps := utils.GetAllowCorporations()
	if len(allowCorps) == 0 {
		return
	}

	ctx := context.Background()
	userRepo := repository.NewUserRepository()
	rollSvc := service.NewRoleService()

	ids, err := userRepo.ListAllIDs()
	if err != nil {
		global.Logger.Error("[CorpCheck] 查询用户 ID 列表失败", zap.Error(err))
		return
	}

	global.Logger.Info("[CorpCheck] 开始军团准入检查", zap.Int("users", len(ids)))
	for _, uid := range ids {
		if err := rollSvc.CheckCorpAccessAndAdjustRole(ctx, uid); err != nil {
			global.Logger.Warn("[CorpCheck] 检查失败",
				zap.Uint("user_id", uid),
				zap.Error(err))
		}
	}
	global.Logger.Info("[CorpCheck] 军团准入检查完成")
}
