package model

import "time"

// TaskSchedule stores admin-configured schedule overrides for recurring tasks.
type TaskSchedule struct {
	TaskName  string    `gorm:"column:task_name;primaryKey;size:100" json:"task_name"`
	CronExpr  string    `gorm:"column:cron_expr;size:100;not null" json:"cron_expr"`
	UpdatedBy uint      `gorm:"column:updated_by" json:"updated_by"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (TaskSchedule) TableName() string { return "task_schedules" }

// TaskExecution records a single run of a backend task.
type TaskExecution struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskName    string     `gorm:"column:task_name;size:100;not null;index:idx_task_exec_name_started,priority:1" json:"task_name"`
	Trigger     string     `gorm:"column:trigger;size:20;not null" json:"trigger"`
	TriggeredBy *uint      `gorm:"column:triggered_by" json:"triggered_by,omitempty"`
	Status      string     `gorm:"column:status;size:20;not null" json:"status"`
	StartedAt   time.Time  `gorm:"column:started_at;not null;index:idx_task_exec_name_started,priority:2,sort:desc;index:idx_task_exec_started_at,sort:desc" json:"started_at"`
	FinishedAt  *time.Time `gorm:"column:finished_at" json:"finished_at,omitempty"`
	DurationMs  *int64     `gorm:"column:duration_ms" json:"duration_ms,omitempty"`
	Error       string     `gorm:"column:error;type:text" json:"error,omitempty"`
	Summary     string     `gorm:"column:summary;type:text" json:"summary,omitempty"`
}

func (TaskExecution) TableName() string { return "task_executions" }

// TaskExecutionHistoryItem is the read model returned by task history queries.
// It extends the persisted execution record with the triggerer's nickname.
type TaskExecutionHistoryItem struct {
	TaskExecution
	TriggeredByName string `gorm:"->;column:triggered_by_name" json:"triggered_by_name,omitempty"`
}
