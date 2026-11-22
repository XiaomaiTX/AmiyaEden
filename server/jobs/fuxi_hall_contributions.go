package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"
	"errors"
)

const FuxiHallContributionsTaskName = "fuxi_hall_contributions"

func runFuxiHallContributionsTask(ctx context.Context, refresh func(context.Context, uint) error) error {
	err := refresh(ctx, 0)
	if errors.Is(err, service.ErrFuxiHallRefreshRunning) {
		return nil
	}
	return err
}

func registerFuxiHallContributionsTask(reg *taskregistry.Registry) {
	reg.Register(taskregistry.TaskDefinition{
		Name:        FuxiHallContributionsTaskName,
		Description: "Refresh hall of fame contribution snapshots",
		Category:    taskregistry.TaskCategoryOperation,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 0 6 * * *",
		RunFunc: func(ctx context.Context) error {
			return runFuxiHallContributionsTask(ctx, service.NewFuxiHallService().RefreshContributions)
		},
	})
}

func FuxiHallContributionsNeedsFirstRun() bool {
	if global.DB == nil {
		return false
	}
	var card model.FuxiHallCard
	return global.DB.Where("page_key = ? AND contributions_updated_at IS NULL", model.FuxiHallPageFame).First(&card).Error == nil
}
