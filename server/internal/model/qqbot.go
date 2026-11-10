package model

import "time"

// QQBotRequest is a closed operation union; validation belongs to the service.
type QQBotRequest struct {
	Kind             string `json:"kind"`
	GroupID          int64  `json:"group_id"`
	UserID           int64  `json:"user_id"`
	Text             string `json:"text"`
	Flag             string `json:"flag"`
	SubType          string `json:"sub_type"`
	Approve          bool   `json:"approve"`
	RejectAddRequest bool   `json:"reject_add_request"`
}
type QQBotOperation struct {
	ID              string     `gorm:"primaryKey;size:36" json:"id"`
	BusinessKey     string     `gorm:"uniqueIndex;size:200" json:"business_key"`
	RequestHash     string     `json:"-"`
	Request         string     `gorm:"type:text" json:"request"`
	Kind            string     `json:"kind"`
	Class           string     `json:"class"`
	Status          string     `gorm:"index;index:qqbot_operation_ready,priority:1" json:"status"`
	Priority        int        `gorm:"index:qqbot_operation_ready,priority:2" json:"priority"`
	ExpiresAt       time.Time  `gorm:"index" json:"expires_at"`
	AvailableAt     time.Time  `gorm:"index" json:"available_at"`
	LeaseToken      string     `json:"-"`
	LeaseUntil      time.Time  `gorm:"index" json:"lease_until"`
	Failures        int        `json:"failures"`
	CancelRequested bool       `json:"cancel_requested"`
	WaitReason      string     `json:"wait_reason"`
	Result          string     `gorm:"type:text" json:"result"`
	CreatedAt       time.Time  `gorm:"index:qqbot_operation_ready,priority:3" json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	TerminalAt      *time.Time `gorm:"index" json:"terminal_at"`
}

func (QQBotOperation) TableName() string { return "qqbot_operation" }

type QQBotAttempt struct {
	ID             string     `gorm:"primaryKey;size:36" json:"id"`
	OperationID    string     `gorm:"index" json:"operation_id"`
	Classification string     `json:"classification"`
	StartedAt      time.Time  `gorm:"index" json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
}

func (QQBotAttempt) TableName() string { return "qqbot_attempt" }

type QQBotInbox struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	Payload   string    `gorm:"type:text" json:"payload"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (QQBotInbox) TableName() string { return "qqbot_inbox" }

type QQBotDelivery struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	Topic       string    `gorm:"uniqueIndex:qqbot_delivery_item" json:"topic"`
	ItemID      string    `gorm:"uniqueIndex:qqbot_delivery_item" json:"item_id"`
	Consumer    string    `gorm:"uniqueIndex:qqbot_delivery_item" json:"consumer"`
	Status      string    `gorm:"index" json:"status"`
	Attempts    int       `json:"attempts"`
	AvailableAt time.Time `json:"available_at"`
	LeaseToken  string    `json:"-"`
	LeaseUntil  time.Time `json:"lease_until"`
	LastError   string    `json:"last_error"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (QQBotDelivery) TableName() string { return "qqbot_delivery" }

type QQBotCircuit struct {
	Class     string    `gorm:"primaryKey" json:"class"`
	State     string    `json:"state"`
	Failures  int       `json:"failures"`
	Successes int64     `json:"successes"`
	Errors    int64     `json:"errors"`
	ProbeAt   time.Time `json:"probe_at"`
	Paused    bool      `json:"paused"`
}

func (QQBotCircuit) TableName() string { return "qqbot_circuit" }

type QQBotSetting struct {
	ID         int    `gorm:"primaryKey" json:"-"`
	Enabled    bool   `json:"enabled"`
	Token      string `json:"-"`
	BotID      int64  `json:"bot_id"`
	CIDRs      string `json:"cidrs"`
	IntervalMS int    `json:"interval_ms"`
}

func (QQBotSetting) TableName() string { return "qqbot_setting" }
