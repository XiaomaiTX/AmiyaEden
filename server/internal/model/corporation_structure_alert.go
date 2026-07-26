package model

import "time"

const (
	CorpStructureAlertFuelExpiring    = "fuel_expiring"
	CorpStructureAlertReinforceEnding = "reinforce_ending"
)

// CorpStructureAlertState records whether one structure is currently inside a
// notification window. It makes threshold-entry notifications durable across
// process restarts and allows a later recovery to arm the next notification.
type CorpStructureAlertState struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CorporationID  int64     `gorm:"not null;uniqueIndex:idx_corp_structure_alert_state,priority:1" json:"corporation_id"`
	StructureID    int64     `gorm:"not null;uniqueIndex:idx_corp_structure_alert_state,priority:2" json:"structure_id"`
	AlertType      string    `gorm:"size:32;not null;uniqueIndex:idx_corp_structure_alert_state,priority:3" json:"alert_type"`
	Active         bool      `gorm:"not null;index" json:"active"`
	Delivered      bool      `gorm:"not null" json:"delivered"`
	StateChangedAt time.Time `gorm:"not null" json:"state_changed_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CorpStructureAlertState) TableName() string { return "corp_structure_alert_state" }
