package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"
	"time"

	"go.uber.org/zap"
)

func registerMentorRewardTask(reg *taskregistry.Registry) {
	svc := service.NewMentorRewardService()
	reg.Register(taskregistry.TaskDefinition{
		Name:        "mentor_reward",
		Description: "Process mentor reward settlements",
		Category:    taskregistry.TaskCategoryOperation,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 0 3 * * *",
		RunFunc: func(ctx context.Context) error {
			result, err := svc.ProcessRewards(time.Now())
			if err != nil {
				global.Logger.Error("导师奖励处理失败", zap.Error(err))
				return err
			}
			global.Logger.Info("导师奖励处理完成",
				zap.Int("processed_relationships", result.ProcessedRelationships),
				zap.Int("rewards_distributed", result.RewardsDistributed),
				zap.Float64("total_coin_awarded", result.TotalCoinAwarded),
				zap.Int("graduated_count", result.GraduatedCount),
			)
			return nil
		},
	})
	global.Logger.Info("注册导师奖励任务成功", zap.String("task_name", "mentor_reward"))
}
