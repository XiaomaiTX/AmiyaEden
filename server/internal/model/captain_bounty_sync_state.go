package model

import "time"

// CaptainBountySyncState stores incremental captain attribution sync progress.
type CaptainBountySyncState struct {
	BaseModel
	SyncKey           string     `gorm:"size:64;not null;uniqueIndex" json:"sync_key"`
	LastWalletJournalID int64    `gorm:"not null;default:0" json:"last_wallet_journal_id"`
	LastJournalAt     *time.Time `json:"last_journal_at"`
}

func (CaptainBountySyncState) TableName() string { return "captain_bounty_sync_state" }
