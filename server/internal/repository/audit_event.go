package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type AuditEventRepository struct{}

func NewAuditEventRepository() *AuditEventRepository {
	return &AuditEventRepository{}
}

type AuditEventFilter struct {
	StartDate    *time.Time
	EndDate      *time.Time
	Category     string
	Action       string
	ActorUserID  *uint
	TargetUserID *uint
	Result       string
	RequestID    string
	ResourceID   string
	Keyword      string
}

func applyAuditEventFilter(db *gorm.DB, filter AuditEventFilter) *gorm.DB {
	if filter.StartDate != nil {
		db = db.Where("occurred_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		db = db.Where("occurred_at <= ?", *filter.EndDate)
	}
	if strings.TrimSpace(filter.Category) != "" {
		db = db.Where("category = ?", strings.TrimSpace(filter.Category))
	}
	if strings.TrimSpace(filter.Action) != "" {
		db = db.Where("action = ?", strings.TrimSpace(filter.Action))
	}
	if filter.ActorUserID != nil {
		db = db.Where("actor_user_id = ?", *filter.ActorUserID)
	}
	if filter.TargetUserID != nil {
		db = db.Where("target_user_id = ?", *filter.TargetUserID)
	}
	if strings.TrimSpace(filter.Result) != "" {
		db = db.Where("result = ?", strings.TrimSpace(filter.Result))
	}
	if strings.TrimSpace(filter.RequestID) != "" {
		db = db.Where("request_id = ?", strings.TrimSpace(filter.RequestID))
	}
	if strings.TrimSpace(filter.ResourceID) != "" {
		db = db.Where("resource_id = ?", strings.TrimSpace(filter.ResourceID))
	}
	if strings.TrimSpace(filter.Keyword) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(filter.Keyword)) + "%"
		db = db.Where("LOWER(details_json) LIKE ?", kw)
	}
	return db
}

func (r *AuditEventRepository) CreateTx(tx *gorm.DB, event *model.AuditEvent) error {
	return tx.Create(event).Error
}

func (r *AuditEventRepository) Create(event *model.AuditEvent) error {
	return global.DB.Create(event).Error
}

func (r *AuditEventRepository) List(page, size int, filter AuditEventFilter) ([]model.AuditEvent, error) {
	offset := (page - 1) * size
	records := make([]model.AuditEvent, 0, size)
	db := applyAuditEventFilter(global.DB.Model(&model.AuditEvent{}), filter)
	err := db.Order("occurred_at DESC, id DESC").Offset(offset).Limit(size).Find(&records).Error
	return records, err
}

func (r *AuditEventRepository) Count(filter AuditEventFilter) (int64, error) {
	var total int64
	err := applyAuditEventFilter(global.DB.Model(&model.AuditEvent{}), filter).Count(&total).Error
	return total, err
}

func (r *AuditEventRepository) ListForExport(filter AuditEventFilter, limit int) ([]model.AuditEvent, error) {
	if limit <= 0 {
		limit = 10000
	}
	records := make([]model.AuditEvent, 0, limit)
	db := applyAuditEventFilter(global.DB.Model(&model.AuditEvent{}), filter)
	err := db.Order("occurred_at DESC, id DESC").Limit(limit).Find(&records).Error
	return records, err
}

func (r *AuditEventRepository) ListOlderThan(cutoff time.Time, limit int) ([]model.AuditEvent, error) {
	if limit <= 0 {
		limit = 1000
	}
	records := make([]model.AuditEvent, 0, limit)
	err := global.DB.Model(&model.AuditEvent{}).
		Where("occurred_at < ?", cutoff).
		Order("occurred_at ASC, id ASC").
		Limit(limit).
		Find(&records).Error
	return records, err
}

func (r *AuditEventRepository) DeleteByIDs(ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tx := global.DB.Where("id IN ?", ids).Delete(&model.AuditEvent{})
	return tx.RowsAffected, tx.Error
}

func (r *AuditEventRepository) CreateExportTask(task *model.AuditExportTask) error {
	return global.DB.Create(task).Error
}

func (r *AuditEventRepository) GetExportTaskByTaskID(taskID string) (*model.AuditExportTask, error) {
	var task model.AuditExportTask
	err := global.DB.Where("task_id = ?", strings.TrimSpace(taskID)).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *AuditEventRepository) ListExportTasksByOperator(operatorUserID uint, limit int) ([]model.AuditExportTask, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	tasks := make([]model.AuditExportTask, 0, limit)
	err := global.DB.Where("operator_user_id = ?", operatorUserID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

func (r *AuditEventRepository) UpdateExportTaskStatus(taskID, fromStatus, toStatus string, updates map[string]any) error {
	tx := global.DB.Model(&model.AuditExportTask{}).
		Where("task_id = ? AND status = ?", strings.TrimSpace(taskID), strings.TrimSpace(fromStatus)).
		Updates(mergeStatusUpdates(toStatus, updates))
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *AuditEventRepository) UpdateExportTask(taskID string, updates map[string]any) error {
	return global.DB.Model(&model.AuditExportTask{}).
		Where("task_id = ?", strings.TrimSpace(taskID)).
		Updates(updates).Error
}

func ParseAuditEventFilter(raw string) (AuditEventFilter, error) {
	var f AuditEventFilter
	if strings.TrimSpace(raw) == "" {
		return f, nil
	}
	err := json.Unmarshal([]byte(raw), &f)
	return f, err
}

func EncodeAuditEventFilter(filter AuditEventFilter) (string, error) {
	b, err := json.Marshal(filter)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func mergeStatusUpdates(status string, updates map[string]any) map[string]any {
	merged := map[string]any{"status": status}
	for k, v := range updates {
		merged[k] = v
	}
	return merged
}
