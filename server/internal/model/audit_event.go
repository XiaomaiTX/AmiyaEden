package model

import "time"

const (
	AuditResultSuccess = "success"
	AuditResultFailed  = "failed"

	AuditExportStatusPending = "pending"
	AuditExportStatusRunning = "running"
	AuditExportStatusDone    = "done"
	AuditExportStatusFailed  = "failed"
	AuditExportStatusExpired = "expired"
)

// AuditEvent is the normalized business-audit event model.
type AuditEvent struct {
	ID           uint      `gorm:"primarykey"                            json:"id"`
	EventID      string    `gorm:"size:64;not null;uniqueIndex"          json:"event_id"`
	OccurredAt   time.Time `gorm:"not null;index"                        json:"occurred_at"`
	Category     string    `gorm:"size:64;not null;index"                json:"category"`
	Action       string    `gorm:"size:128;not null;index"               json:"action"`
	ActorUserID  uint      `gorm:"default:0;index"                       json:"actor_user_id"`
	TargetUserID uint      `gorm:"default:0;index"                       json:"target_user_id"`
	ResourceType string    `gorm:"size:64;default:''"                     json:"resource_type"`
	ResourceID   string    `gorm:"size:128;default:''"                    json:"resource_id"`
	Result       string    `gorm:"size:16;not null;index"                json:"result"`
	RequestID    string    `gorm:"size:64;default:'';index"              json:"request_id"`
	IP           string    `gorm:"size:64;default:''"                     json:"ip"`
	UserAgent    string    `gorm:"size:256;default:''"                    json:"user_agent"`
	DetailsJSON  string    `gorm:"type:text"                              json:"details_json"`
	CreatedAt    time.Time `gorm:"autoCreateTime"                         json:"created_at"`
}

func (AuditEvent) TableName() string {
	return "audit_event"
}

// AuditExportTask tracks async export jobs for filtered audit events.
type AuditExportTask struct {
	ID             uint       `gorm:"primarykey"                   json:"id"`
	TaskID         string     `gorm:"size:64;not null;uniqueIndex" json:"task_id"`
	OperatorUserID uint       `gorm:"not null;index"               json:"operator_user_id"`
	Format         string     `gorm:"size:16;not null"             json:"format"`
	FilterJSON     string     `gorm:"type:text;not null"           json:"filter_json"`
	Status         string     `gorm:"size:16;not null;index"       json:"status"`
	DownloadURL    string     `gorm:"size:512;default:''"          json:"download_url"`
	ErrorMessage   string     `gorm:"type:text"                    json:"error_message"`
	RowCount       int        `gorm:"default:0"                    json:"row_count"`
	StartedAt      *time.Time `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	ExpireAt       *time.Time `gorm:"index"                        json:"expire_at"`
	CreatedAt      time.Time  `gorm:"autoCreateTime"               json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime"               json:"updated_at"`
}

func (AuditExportTask) TableName() string {
	return "audit_export_task"
}
