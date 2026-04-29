package model

import "time"

const (
	TicketStatusPending    = "pending"
	TicketStatusInProgress = "in_progress"
	TicketStatusCompleted  = "completed"
)

const (
	TicketPriorityLow    = "low"
	TicketPriorityMedium = "medium"
	TicketPriorityHigh   = "high"
)

type Ticket struct {
	BaseModel
	UserID      uint       `gorm:"not null;index" json:"user_id"`
	CategoryID  uint       `gorm:"not null;index" json:"category_id"`
	Title       string     `gorm:"size:200;not null" json:"title"`
	Description string     `gorm:"type:text;not null" json:"description"`
	Status      string     `gorm:"size:20;not null;default:'pending';index" json:"status"`
	Priority    string     `gorm:"size:20;not null;default:'medium'" json:"priority"`
	HandledBy   *uint      `gorm:"index" json:"handled_by,omitempty"`
	HandledAt   *time.Time `json:"handled_at,omitempty"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
}

func (Ticket) TableName() string {
	return "ticket"
}

type TicketCategory struct {
	BaseModel
	Name        string `gorm:"size:50;not null;uniqueIndex" json:"name"`
	NameEN      string `gorm:"size:50;not null;uniqueIndex" json:"name_en"`
	Description string `gorm:"size:200" json:"description"`
	SortOrder   int    `gorm:"not null;default:0" json:"sort_order"`
	Enabled     bool   `gorm:"not null;default:true" json:"enabled"`
}

func (TicketCategory) TableName() string {
	return "ticket_category"
}

type TicketReply struct {
	BaseModel
	TicketID   uint   `gorm:"not null;index" json:"ticket_id"`
	UserID     uint   `gorm:"not null;index" json:"user_id"`
	Content    string `gorm:"type:text;not null" json:"content"`
	IsInternal bool   `gorm:"not null;default:false" json:"is_internal"`
}

func (TicketReply) TableName() string {
	return "ticket_reply"
}

type TicketStatusHistory struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	TicketID   uint      `gorm:"not null;index" json:"ticket_id"`
	FromStatus string    `gorm:"size:20" json:"from_status"`
	ToStatus   string    `gorm:"size:20;not null" json:"to_status"`
	ChangedBy  uint      `gorm:"not null" json:"changed_by"`
	ChangedAt  time.Time `gorm:"autoCreateTime" json:"changed_at"`
}

func (TicketStatusHistory) TableName() string {
	return "ticket_status_history"
}

