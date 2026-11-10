package model

import "time"

type QQGroupRule struct {
	InspectionEnabled     bool      `json:"inspection_enabled"`
	AutoKick              bool      `json:"auto_kick"`
	MismatchConfirmations int       `json:"mismatch_confirmations"`
	ObservationHours      int       `json:"observation_hours"`
	LastInspectionAt      time.Time `json:"last_inspection_at"`
	CardEnabled           bool      `json:"card_enabled"`
	CardTemplate          string    `json:"card_template"`
	GroupID               int64     `gorm:"primaryKey" json:"group_id"`
	Version               int64     `json:"version"`
	AdmissionEnabled      bool      `json:"admission_enabled"`
	AutoReject            bool      `json:"auto_reject"`
	CorporationIDs        []int64   `gorm:"serializer:json;type:text" json:"corporation_ids"`
	Roles                 []string  `gorm:"serializer:json;type:text" json:"roles"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (QQGroupRule) TableName() string { return "qq_group_rule" }

type QQGroupApplication struct {
	ID          string    `gorm:"primaryKey;size:64" json:"id"`
	GroupID     int64     `gorm:"index" json:"group_id"`
	QQ          int64     `json:"qq"`
	Flag        string    `json:"-"`
	SubType     string    `json:"sub_type"`
	Status      string    `json:"status"`
	Decision    string    `json:"decision"`
	Evidence    string    `json:"evidence"`
	RuleVersion int64     `json:"rule_version"`
	OperationID string    `gorm:"index" json:"operation_id"`
	ExpiresAt   time.Time `json:"expires_at"`
	RecheckAt   time.Time `json:"recheck_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (QQGroupApplication) TableName() string { return "qq_group_application" }

type QQGroupMember struct {
	MembershipReceivedAt time.Time  `json:"membership_received_at"`
	Confirmations        int        `json:"confirmations"`
	ConfirmRuleVersion   int64      `json:"confirm_rule_version"`
	MismatchSince        *time.Time `json:"mismatch_since"`
	ChainID              string     `json:"chain_id"`
	LastMismatchRunID    string     `json:"last_mismatch_run_id"`
	KickOperationID      string     `json:"kick_operation_id"`
	KickStatus           string     `json:"kick_status"`
	GroupID              int64      `gorm:"primaryKey" json:"group_id"`
	QQ                   int64      `gorm:"primaryKey" json:"qq"`
	Present              bool       `json:"present"`
	EventAt              time.Time  `json:"event_at"`
	Epoch                int64      `json:"epoch"`
	Evidence             string     `json:"evidence"`
	CardTarget           string     `json:"card_target"`
	CardVersion          int64      `json:"card_version"`
	CardOperationID      string     `json:"card_operation_id"`
	CardStatus           string     `json:"card_status"`
	CardLastAttemptAt    time.Time  `json:"card_last_attempt_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func (QQGroupMember) TableName() string { return "qq_group_member" }

type QQGroupAppliedEvent struct {
	ID string `gorm:"primaryKey;size:64"`
}

func (QQGroupAppliedEvent) TableName() string { return "qq_group_applied_event" }

type QQGroupInspection struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	GroupID     int64     `gorm:"index" json:"group_id"`
	RuleVersion int64     `json:"rule_version"`
	Status      string    `gorm:"index" json:"status"`
	OperationID string    `gorm:"uniqueIndex" json:"operation_id"`
	SnapshotAt  time.Time `json:"snapshot_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (QQGroupInspection) TableName() string { return "qq_group_inspection" }

type QQGroupInspectionItem struct {
	RunID       string    `gorm:"primaryKey;size:36" json:"run_id"`
	QQ          int64     `gorm:"primaryKey" json:"qq"`
	Present     bool      `json:"present"`
	Done        bool      `gorm:"index" json:"done"`
	Decision    string    `json:"decision"`
	Evidence    string    `json:"evidence"`
	OperationID string    `json:"operation_id"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (QQGroupInspectionItem) TableName() string { return "qq_group_inspection_item" }
