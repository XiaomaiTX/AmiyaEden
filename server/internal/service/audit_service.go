package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuditRecordInput struct {
	Category     string
	Action       string
	ActorUserID  uint
	TargetUserID uint
	ResourceType string
	ResourceID   string
	Result       string
	RequestID    string
	IP           string
	UserAgent    string
	Details      map[string]any
}

type AuditService struct {
	repo *repository.AuditEventRepository
}

const (
	auditExportMaxRows      = 20000
	auditExportExpireHours  = 24
	auditExportStorageDir   = "uploads/audit-exports"
	auditExportActionCreate = "audit_export_create"
	auditExportActionReady  = "audit_export_generated"
)

func NewAuditService() *AuditService {
	return &AuditService{repo: repository.NewAuditEventRepository()}
}

func (s *AuditService) RecordEventTx(tx *gorm.DB, in AuditRecordInput) error {
	details, _ := json.Marshal(in.Details)
	event := &model.AuditEvent{
		EventID:      uuid.NewString(),
		OccurredAt:   time.Now(),
		Category:     in.Category,
		Action:       in.Action,
		ActorUserID:  in.ActorUserID,
		TargetUserID: in.TargetUserID,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		Result:       in.Result,
		RequestID:    in.RequestID,
		IP:           in.IP,
		UserAgent:    in.UserAgent,
		DetailsJSON:  string(details),
	}
	if err := s.repo.CreateTx(tx, event); err != nil {
		if isAuditTableMissingError(err) {
			return nil
		}
		return err
	}
	return nil
}

func (s *AuditService) RecordEvent(_ context.Context, in AuditRecordInput) error {
	details, _ := json.Marshal(in.Details)
	event := &model.AuditEvent{
		EventID:      uuid.NewString(),
		OccurredAt:   time.Now(),
		Category:     in.Category,
		Action:       in.Action,
		ActorUserID:  in.ActorUserID,
		TargetUserID: in.TargetUserID,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		Result:       in.Result,
		RequestID:    in.RequestID,
		IP:           in.IP,
		UserAgent:    in.UserAgent,
		DetailsJSON:  string(details),
	}
	if err := s.repo.Create(event); err != nil {
		if isAuditTableMissingError(err) {
			return nil
		}
		return err
	}
	return nil
}

func (s *AuditService) AdminListAuditEvents(page, size int, filter repository.AuditEventFilter) ([]model.AuditEvent, int64, error) {
	page = normalizePage(page)
	size = normalizeLedgerPageSize(size)

	records, err := s.repo.List(page, size, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(filter)
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func isAuditTableMissingError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table: audit_event") ||
		strings.Contains(msg, "relation \"audit_event\" does not exist")
}

type AuditExportTaskCreateInput struct {
	OperatorUserID uint
	Format         string
	Filter         repository.AuditEventFilter
	RequestID      string
	IP             string
	UserAgent      string
}

type AuditExportTaskStatus struct {
	TaskID       string     `json:"task_id"`
	Status       string     `json:"status"`
	Format       string     `json:"format,omitempty"`
	DownloadURL  string     `json:"download_url,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	ExpireAt     *time.Time `json:"expire_at,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

func (s *AuditService) CreateExportTask(ctx context.Context, in AuditExportTaskCreateInput) (*AuditExportTaskStatus, error) {
	format := strings.ToLower(strings.TrimSpace(in.Format))
	if format != "csv" && format != "json" {
		return nil, fmt.Errorf("unsupported format: %s", in.Format)
	}

	filterJSON, err := repository.EncodeAuditEventFilter(in.Filter)
	if err != nil {
		return nil, err
	}

	taskID := uuid.NewString()
	task := &model.AuditExportTask{
		TaskID:         taskID,
		OperatorUserID: in.OperatorUserID,
		Format:         format,
		FilterJSON:     filterJSON,
		Status:         model.AuditExportStatusPending,
	}
	if err := s.repo.CreateExportTask(task); err != nil {
		return nil, err
	}

	_ = s.RecordEvent(ctx, AuditRecordInput{
		Category:     "task_ops",
		Action:       auditExportActionCreate,
		ActorUserID:  in.OperatorUserID,
		ResourceType: "audit_export_task",
		ResourceID:   taskID,
		Result:       model.AuditResultSuccess,
		RequestID:    in.RequestID,
		IP:           in.IP,
		UserAgent:    in.UserAgent,
		Details: map[string]any{
			"format": format,
			"filter": in.Filter,
		},
	})

	go s.runExportTaskInBackground(context.Background(), taskID)

	return &AuditExportTaskStatus{TaskID: taskID, Status: model.AuditExportStatusPending}, nil
}

func (s *AuditService) GetExportTaskStatus(taskID string) (*AuditExportTaskStatus, error) {
	task, err := s.repo.GetExportTaskByTaskID(taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, gorm.ErrRecordNotFound
	}

	if task.Status == model.AuditExportStatusDone && task.ExpireAt != nil && task.ExpireAt.Before(time.Now()) {
		_ = s.repo.UpdateExportTask(task.TaskID, map[string]any{"status": model.AuditExportStatusExpired})
		task.Status = model.AuditExportStatusExpired
	}

	status := &AuditExportTaskStatus{
		TaskID:       task.TaskID,
		Status:       task.Status,
		Format:       task.Format,
		DownloadURL:  task.DownloadURL,
		ErrorMessage: task.ErrorMessage,
		ExpireAt:     task.ExpireAt,
		CreatedAt:    &task.CreatedAt,
		FinishedAt:   task.FinishedAt,
	}
	return status, nil
}

func (s *AuditService) ListExportTaskStatuses(operatorUserID uint, limit int) ([]AuditExportTaskStatus, error) {
	tasks, err := s.repo.ListExportTasksByOperator(operatorUserID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]AuditExportTaskStatus, 0, len(tasks))
	now := time.Now()
	for i := range tasks {
		task := tasks[i]
		status := task.Status
		if status == model.AuditExportStatusDone && task.ExpireAt != nil && task.ExpireAt.Before(now) {
			_ = s.repo.UpdateExportTask(task.TaskID, map[string]any{"status": model.AuditExportStatusExpired})
			status = model.AuditExportStatusExpired
		}
		createdAt := task.CreatedAt
		out = append(out, AuditExportTaskStatus{
			TaskID:       task.TaskID,
			Status:       status,
			Format:       task.Format,
			DownloadURL:  task.DownloadURL,
			ErrorMessage: task.ErrorMessage,
			ExpireAt:     task.ExpireAt,
			CreatedAt:    &createdAt,
			FinishedAt:   task.FinishedAt,
		})
	}
	return out, nil
}

func (s *AuditService) runExportTaskInBackground(ctx context.Context, taskID string) {
	now := time.Now()
	err := s.repo.UpdateExportTaskStatus(taskID, model.AuditExportStatusPending, model.AuditExportStatusRunning, map[string]any{
		"started_at": now,
	})
	if err != nil {
		return
	}
	if err := s.generateExportFile(ctx, taskID); err != nil {
		_ = s.repo.UpdateExportTask(taskID, map[string]any{
			"status":        model.AuditExportStatusFailed,
			"error_message": err.Error(),
			"finished_at":   time.Now(),
		})
	}
}

func (s *AuditService) generateExportFile(ctx context.Context, taskID string) error {
	task, err := s.repo.GetExportTaskByTaskID(taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return gorm.ErrRecordNotFound
	}
	if task.Status != model.AuditExportStatusRunning {
		return nil
	}

	filter, err := repository.ParseAuditEventFilter(task.FilterJSON)
	if err != nil {
		return err
	}
	rows, err := s.repo.ListForExport(filter, auditExportMaxRows)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(auditExportStorageDir, 0o755); err != nil {
		return err
	}

	filename := fmt.Sprintf("audit-%s.%s", task.TaskID, task.Format)
	fullPath := filepath.Join(auditExportStorageDir, filename)
	switch task.Format {
	case "csv":
		err = writeAuditRowsCSV(fullPath, rows)
	case "json":
		err = writeAuditRowsJSON(fullPath, rows)
	default:
		err = fmt.Errorf("unsupported format: %s", task.Format)
	}
	if err != nil {
		return err
	}

	expireAt := time.Now().Add(auditExportExpireHours * time.Hour)
	downloadURL := "/uploads/audit-exports/" + filename
	if err := s.repo.UpdateExportTask(task.TaskID, map[string]any{
		"status":        model.AuditExportStatusDone,
		"download_url":  downloadURL,
		"row_count":     len(rows),
		"error_message": "",
		"finished_at":   time.Now(),
		"expire_at":     expireAt,
	}); err != nil {
		return err
	}

	_ = s.RecordEvent(ctx, AuditRecordInput{
		Category:     "task_ops",
		Action:       auditExportActionReady,
		ActorUserID:  task.OperatorUserID,
		ResourceType: "audit_export_task",
		ResourceID:   task.TaskID,
		Result:       model.AuditResultSuccess,
		Details: map[string]any{
			"format":       task.Format,
			"row_count":    len(rows),
			"download_url": downloadURL,
		},
	})
	return nil
}

func writeAuditRowsJSON(path string, rows []model.AuditEvent) error {
	data, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func writeAuditRowsCSV(path string, rows []model.AuditEvent) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"event_id", "occurred_at", "category", "action", "actor_user_id", "target_user_id",
		"resource_type", "resource_id", "result", "request_id", "ip", "user_agent", "details_json",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for i := range rows {
		row := rows[i]
		record := []string{
			row.EventID,
			row.OccurredAt.Format(time.RFC3339),
			row.Category,
			row.Action,
			fmt.Sprintf("%d", row.ActorUserID),
			fmt.Sprintf("%d", row.TargetUserID),
			row.ResourceType,
			row.ResourceID,
			row.Result,
			row.RequestID,
			row.IP,
			row.UserAgent,
			row.DetailsJSON,
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	if err := w.Error(); err != nil {
		return err
	}
	return nil
}
