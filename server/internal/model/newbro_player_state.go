package model

import "time"

// NewbroPlayerState caches the latest evaluated newbro eligibility state.
type NewbroPlayerState struct {
	BaseModel
	UserID             uint      `gorm:"not null;uniqueIndex" json:"user_id"`
	IsCurrentlyNewbro  bool      `gorm:"not null;default:false" json:"is_currently_newbro"`
	EvaluatedAt        time.Time `gorm:"not null" json:"evaluated_at"`
	RuleVersion        string    `gorm:"size:255;not null" json:"rule_version"`
	DisqualifiedReason string    `gorm:"size:64" json:"disqualified_reason"`
}

func (NewbroPlayerState) TableName() string { return "newbro_player_state" }
