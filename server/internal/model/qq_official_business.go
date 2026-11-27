package model

import "time"

// QQOfficialStructureAlert preserves the current episode and monotonically
// increasing version, even once the delivery ledger is pruned.
type QQOfficialStructureAlert struct {
	AppID         string `gorm:"primaryKey;size:64"`
	CorporationID int64  `gorm:"primaryKey;autoIncrement:false"`
	StructureID   int64  `gorm:"primaryKey;autoIncrement:false"`
	Kind          string `gorm:"primaryKey;size:16"`
	Version       int64  `gorm:"not null"`
	Active        bool   `gorm:"not null"`
	Pending       bool   `gorm:"not null"`
	TargetFence   string `gorm:"size:64"`
	ExpiresAt     time.Time
	TimerDeadline *time.Time
}

func (QQOfficialStructureAlert) TableName() string { return "qq_official_structure_alert" }
