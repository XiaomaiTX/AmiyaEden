package model

import "time"

// CaptainRewardSettlement stores one captain reward payout per processing batch.
type CaptainRewardSettlement struct {
	BaseModel
	CaptainUserID      uint      `gorm:"not null;index" json:"captain_user_id"`
	AttributionCount   int64     `gorm:"not null" json:"attribution_count"`
	AttributedISKTotal float64   `gorm:"type:decimal(25,2);not null" json:"attributed_isk_total"`
	BonusRate          float64   `gorm:"type:decimal(8,2);not null" json:"bonus_rate"`
	CreditedValue      float64   `gorm:"type:decimal(25,2);not null" json:"credited_value"`
	ProcessedAt        time.Time `gorm:"not null;index" json:"processed_at"`
	WalletRefID        string    `gorm:"size:128;not null;uniqueIndex" json:"wallet_ref_id"`
}

func (CaptainRewardSettlement) TableName() string { return "captain_reward_settlement" }
