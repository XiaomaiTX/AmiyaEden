package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"
	"time"

	"go.uber.org/zap"
)

func registerAdmissionEvidenceCleanup(reg *taskregistry.Registry) {
	reg.Register(taskregistry.TaskDefinition{
		Name: "admission_evidence_cleanup", Description: "Delete expired private admission evidence", Category: taskregistry.TaskCategorySystem, Type: taskregistry.TaskTypeRecurring, DefaultCron: "0 30 5 * * *",
		RunFunc: func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			count, err := service.NewAdmissionService().CleanupEvidence(time.Now())
			global.Logger.Info("admission evidence cleanup", zap.Int("applications", count), zap.Error(err))
			return err
		},
	})
}
