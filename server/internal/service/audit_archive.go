package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	auditArchiveOnlineRetentionDays = 90
	auditArchiveBatchSize           = 1000
	auditArchiveAction              = "audit_archive_daily"
)

var auditArchiveStorageDir = "uploads/audit-archive"

type AuditArchiveSummary struct {
	CutoffAt    time.Time
	FilePath    string
	Batches     int
	ArchivedRows int
	PurgedRows  int
}

type AuditArchiveService struct {
	repo     *repository.AuditEventRepository
	auditSvc *AuditService
	nowFn    func() time.Time
}

func NewAuditArchiveService() *AuditArchiveService {
	return &AuditArchiveService{
		repo:     repository.NewAuditEventRepository(),
		auditSvc: NewAuditService(),
		nowFn:    time.Now,
	}
}

func (s *AuditArchiveService) RunDailyArchive(ctx context.Context) (*AuditArchiveSummary, error) {
	now := s.nowFn()
	cutoff := now.AddDate(0, 0, -auditArchiveOnlineRetentionDays)
	filename := fmt.Sprintf("audit-archive-%s.jsonl", now.UTC().Format("20060102-150405"))

	if err := os.MkdirAll(auditArchiveStorageDir, 0o755); err != nil {
		return nil, err
	}
	filePath := filepath.Join(auditArchiveStorageDir, filename)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	summary := &AuditArchiveSummary{
		CutoffAt: cutoff,
		FilePath: filePath,
	}

	for {
		rows, err := s.repo.ListOlderThan(cutoff, auditArchiveBatchSize)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			break
		}

		ids := make([]uint, 0, len(rows))
		for i := range rows {
			payload, err := json.Marshal(rows[i])
			if err != nil {
				return nil, err
			}
			if _, err := file.Write(append(payload, '\n')); err != nil {
				return nil, err
			}
			ids = append(ids, rows[i].ID)
		}

		purged, err := s.repo.DeleteByIDs(ids)
		if err != nil {
			return nil, err
		}

		summary.Batches++
		summary.ArchivedRows += len(rows)
		summary.PurgedRows += int(purged)
	}

	if summary.ArchivedRows == 0 {
		_ = os.Remove(filePath)
		summary.FilePath = ""
	}

	_ = s.auditSvc.RecordEvent(ctx, AuditRecordInput{
		Category:     "task_ops",
		Action:       auditArchiveAction,
		ResourceType: "audit_event",
		Result:       model.AuditResultSuccess,
		Details: map[string]any{
			"cutoff_at":     cutoff.Format(time.RFC3339),
			"archive_file":  summary.FilePath,
			"archived_rows": summary.ArchivedRows,
			"purged_rows":   summary.PurgedRows,
			"batches":       summary.Batches,
		},
	})

	return summary, nil
}
