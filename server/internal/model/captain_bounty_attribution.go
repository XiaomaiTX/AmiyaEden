package model

import "time"

// CaptainBountyAttribution stores persisted captain bounty attribution ledger rows.
type CaptainBountyAttribution struct {
	BaseModel
	AffiliationID          uint       `gorm:"not null;index" json:"affiliation_id"`
	PlayerUserID           uint       `gorm:"not null;index" json:"player_user_id"`
	PlayerCharacterID      int64      `gorm:"not null;index" json:"player_character_id"`
	CaptainUserID          uint       `gorm:"not null;index" json:"captain_user_id"`
	CaptainCharacterID     int64      `gorm:"not null;index" json:"captain_character_id"`
	CaptainWalletJournalID int64      `gorm:"not null;index" json:"captain_wallet_journal_id"`
	WalletJournalID        int64      `gorm:"not null;uniqueIndex" json:"wallet_journal_id"`
	RefType                string     `gorm:"size:64;not null;index" json:"ref_type"`
	SystemID               int64      `gorm:"not null;index" json:"system_id"`
	JournalAt              time.Time  `gorm:"not null;index" json:"journal_at"`
	Amount                 float64    `gorm:"type:decimal(25,2);not null" json:"amount"`
	ProcessedAt            *time.Time `gorm:"index" json:"processed_at"`
}

func (CaptainBountyAttribution) TableName() string { return "captain_bounty_attribution" }
