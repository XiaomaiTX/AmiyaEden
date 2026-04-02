package model

import "time"

// NewbroCaptainAffiliation stores player-captain relationship history.
type NewbroCaptainAffiliation struct {
	BaseModel
	PlayerUserID                    uint       `gorm:"not null;index" json:"player_user_id"`
	PlayerPrimaryCharacterIDAtStart int64      `gorm:"not null;index" json:"player_primary_character_id_at_start"`
	CaptainUserID                   uint       `gorm:"not null;index" json:"captain_user_id"`
	CreatedBy                       uint       `gorm:"not null;index" json:"created_by"`
	StartedAt                       time.Time  `gorm:"not null;index" json:"started_at"`
	EndedAt                         *time.Time `gorm:"index" json:"ended_at"`
}

func (NewbroCaptainAffiliation) TableName() string { return "newbro_captain_affiliation" }
