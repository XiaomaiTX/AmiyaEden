package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"amiya-eden/pkg/eve/esi"
	"context"

	"go.uber.org/zap"
)

// esiQueue 全局 ESI 刷新队列实例
var esiQueue *esi.Queue

var newESIQueueForJobs = func() *esi.Queue {
	return esi.NewQueue(
		service.NewEveSSOService(),
		repository.NewEveCharacterRepository(),
	)
}

var startInitialESIQueueRun = func(queue *esi.Queue) {
	go queue.Run()
}

// GetESIQueue 获取 ESI 刷新队列实例（供 handler 层使用）
func GetESIQueue() *esi.Queue {
	return esiQueue
}

// SetTestESIQueue 设置测试用的 ESI 队列实例（仅用于测试）
func SetTestESIQueue(queue *esi.Queue) {
	esiQueue = queue
}

// registerESIRefreshTask 注册 ESI 数据刷新任务定义。
func registerESIRefreshTask(reg *taskregistry.Registry) {
	esiQueue = newESIQueueForJobs()

	rollSvc := service.NewRoleService()
	autoRoleSvc := service.NewAutoRoleService()

	runSigninSecuritySync := func(characterID int64, userID uint) {
		ctx := context.Background()

		// 先刷新 affiliation，确保 corporation_id 是当前值。
		if err := esiQueue.RunTask("character_affiliation", characterID); err != nil {
			global.Logger.Warn("[ESI SyncHook] affiliation 任务执行失败",
				zap.Int64("character_id", characterID),
				zap.Error(err),
			)
		}

		// 再刷新 corp roles，让 allow_corporations 过滤作用在最新军团上。
		if err := esiQueue.RunTask("character_corp_roles", characterID); err != nil {
			global.Logger.Warn("[ESI SyncHook] corp roles 任务执行失败",
				zap.Int64("character_id", characterID),
				zap.Error(err),
			)
		}

		if err := rollSvc.CheckCorpAccessAndAdjustRole(ctx, userID); err != nil {
			global.Logger.Warn("[ESI SyncHook] 权限检查失败",
				zap.Int64("character_id", characterID),
				zap.Uint("user_id", userID),
				zap.Error(err),
			)
		}
		if err := autoRoleSvc.SyncUserAutoRoles(ctx, userID); err != nil {
			global.Logger.Warn("[ESI SyncHook] 自动权限同步失败",
				zap.Int64("character_id", characterID),
				zap.Uint("user_id", userID),
				zap.Error(err),
			)
		}
	}

	// 注入同步钩子：在 JWT 生成前同步拉取最小安全数据并重算权限
	service.OnNewCharacterSyncFunc = func(characterID int64, userID uint) {
		runSigninSecuritySync(characterID, userID)
	}

	// 注入新人物全量刷新钩子：SSO 回调完成后后台异步执行，跑全部 ESI 任务，完成后补一次军团准入检查 + 自动权限同步
	service.OnNewCharacterFunc = func(characterID int64, userID uint) {
		ctx := context.Background()
		esiQueue.RunAllForCharacter(ctx, characterID)
		if err := rollSvc.CheckCorpAccessAndAdjustRole(ctx, userID); err != nil {
			global.Logger.Warn("[ESI FullRefreshHook] 权限检查失败",
				zap.Int64("character_id", characterID),
				zap.Uint("user_id", userID),
				zap.Error(err),
			)
		}
		// ESI 全量刷新完成后同步自动权限（corp_roles + titles 已入库）
		_ = autoRoleSvc.SyncUserAutoRoles(ctx, userID)
	}

	// 注入已有人物绑定/重登录同步钩子：JWT 生成前先刷新 affiliation / corp roles，再重算权限
	service.OnExistingCharacterSyncFunc = func(characterID int64, userID uint) {
		runSigninSecuritySync(characterID, userID)
	}

	reg.Register(taskregistry.TaskDefinition{
		Name:        "esi_refresh",
		Description: "Refresh ESI data for registered characters",
		Category:    taskregistry.TaskCategoryESI,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			esiQueue.Run()
			return nil
		},
	})
	global.Logger.Info("注册 ESI 刷新任务成功", zap.String("task_name", "esi_refresh"))
	startInitialESIQueueRun(esiQueue)
}
