package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"
	"time"

	"go.uber.org/zap"
)

func registerNewbroSupportTasks(reg *taskregistry.Registry) {
	attributionSvc := service.NewCaptainBountySyncService()
	rewardSvc := service.NewCaptainRewardProcessingService()

	reg.Register(taskregistry.TaskDefinition{
		Name:        "captain_attribution_sync",
		Description: "Synchronize captain bounty attributions",
		Category:    taskregistry.TaskCategoryOperation,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "@every 13h",
		RunFunc: func(ctx context.Context) error {
			result, err := attributionSvc.RunSync(time.Now())
			if err != nil {
				global.Logger.Error("队长归因同步失败", zap.Error(err))
				return err
			}
			global.Logger.Info(
				"队长归因同步完成",
				zap.Int("processed_count", result.ProcessedCount),
				zap.Int("inserted_count", result.InsertedCount),
				zap.Int("skipped_count", result.SkippedCount),
				zap.Int64("last_wallet_journal_id", result.LastWalletJournalID),
			)
			return nil
		},
	})
	global.Logger.Info("注册队长归因同步任务成功", zap.String("task_name", "captain_attribution_sync"))

	reg.Register(taskregistry.TaskDefinition{
		Name:        "captain_reward_processing",
		Description: "Process captain attribution rewards",
		Category:    taskregistry.TaskCategoryOperation,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "@every 100h",
		RunFunc: func(ctx context.Context) error {
			result, err := rewardSvc.Run(time.Now())
			if err != nil {
				global.Logger.Error("队长奖励处理失败", zap.Error(err))
				return err
			}
			global.Logger.Info(
				"队长奖励处理完成",
				zap.Int("processed_captain_count", result.ProcessedCaptainCount),
				zap.Int("processed_attribution_count", result.ProcessedAttributionCount),
				zap.Int("settlement_count", result.SettlementCount),
				zap.Float64("total_credited_value", result.TotalCreditedValue),
			)
			return nil
		},
	})
	global.Logger.Info("注册队长奖励处理任务成功", zap.String("task_name", "captain_reward_processing"))
}
