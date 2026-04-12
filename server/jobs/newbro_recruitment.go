package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"
	"time"

	"go.uber.org/zap"
)

func registerNewbroRecruitmentTask(reg *taskregistry.Registry) {
	svc := service.NewRecruitmentEntryService()

	reg.Register(taskregistry.TaskDefinition{
		Name:        "recruit_link_check",
		Description: "Check recruitment QQ entries and award coins for valid recruitments",
		Category:    taskregistry.TaskCategoryOperation,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 0 2 * * *", // 2 AM daily
		RunFunc: func(ctx context.Context) error {
			result, err := svc.ProcessOngoingEntries(time.Now())
			if err != nil {
				global.Logger.Error("招募链接检查失败", zap.Error(err))
				return err
			}
			global.Logger.Info("招募链接检查完成",
				zap.Int("processed", result.ProcessedCount),
				zap.Int("valid", result.ValidCount),
				zap.Int("stalled", result.StalledCount),
				zap.Float64("total_coin_awarded", result.TotalCoinAwarded),
			)
			return nil
		},
	})
	global.Logger.Info("注册招募链接检查任务成功", zap.String("task_name", "recruit_link_check"))
}
