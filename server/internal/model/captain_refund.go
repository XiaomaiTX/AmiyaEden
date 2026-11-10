package model

import "time"

const ISKPerFuxiCoin = 1_000_000

// CaptainRefund contains an immutable calculation snapshot. Only review fields change.
type CaptainRefund struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"not null;index" json:"user_id"`
	Status       string     `gorm:"size:16;not null;index" json:"status"`
	Snapshot     string     `gorm:"type:text;not null" json:"-"`
	Reward       float64    `gorm:"type:decimal(25,2);not null" json:"reward"`
	CreatedAt    time.Time  `json:"created_at"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	ReviewedBy   uint       `json:"reviewed_by"`
	RejectReason string     `gorm:"type:text" json:"reject_reason"`
	WalletRefID  string     `gorm:"size:128" json:"wallet_ref_id"`
}

func (CaptainRefund) TableName() string { return "captain_refund" }

// Active claims use a composite primary key, shared across all captains and roles.
// Rejection deletes claims but keeps the immutable snapshot on the application.
type CaptainRefundClaim struct {
	CharacterID int64     `gorm:"primaryKey;autoIncrement:false" json:"character_id"`
	JournalID   int64     `gorm:"primaryKey;autoIncrement:false" json:"journal_id"`
	RefundID    uint      `gorm:"not null;index" json:"refund_id"`
	JournalAt   time.Time `json:"journal_at"`
}

func (CaptainRefundClaim) TableName() string { return "captain_refund_claim" }
