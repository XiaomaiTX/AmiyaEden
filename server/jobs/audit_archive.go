package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"context"

	"go.uber.org/zap"
)

const (
	auditArchiveTaskName = "audit_archive_daily"
	// Run every day at 05:00:00 local time.
	auditArchiveTaskCron = "0 0 5 * * *"
)

func registerAuditArchiveTask(reg *taskregistry.Registry) {
	archiveSvc := service.NewAuditArchiveService()

	reg.Register(taskregistry.TaskDefinition{
		Name:        auditArchiveTaskName,
		Description: "Archive and purge audit events older than retention window",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: auditArchiveTaskCron,
		RunFunc: func(ctx context.Context) error {
			summary, err := archiveSvc.RunDailyArchive(ctx)
			if err != nil {
				global.Logger.Error("audit archive task failed", zap.Error(err))
				return err
			}

			global.Logger.Info(
				"audit archive task completed",
				zap.Int("archived_batches", summary.Batches),
				zap.Int("archived_rows", summary.ArchivedRows),
				zap.Int("purged_rows", summary.PurgedRows),
				zap.String("archive_file", summary.FilePath),
				zap.String("cutoff_at", summary.CutoffAt.Format("2006-01-02T15:04:05Z07:00")),
			)
			return nil
		},
	})

	global.Logger.Info("registered audit archive task", zap.String("task_name", auditArchiveTaskName))
}
