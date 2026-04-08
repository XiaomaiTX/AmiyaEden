package esimodel

import "time"

// CharacterCorporationHistory 人物军团任职历史快照
type CharacterCorporationHistory struct {
	ID            uint      `gorm:"primarykey"                                                             json:"id"`
	CharacterID   int64     `gorm:"not null;index:idx_character_corporation_history_char_start,priority:1;uniqueIndex:idx_character_corporation_history_record,priority:1;index" json:"character_id"`
	RecordID      int64     `gorm:"not null;uniqueIndex:idx_character_corporation_history_record,priority:2" json:"record_id"`
	CorporationID int64     `gorm:"not null;index"                                                        json:"corporation_id"`
	IsDeleted     bool      `gorm:"not null;default:false"                                                json:"is_deleted"`
	StartDate     time.Time `gorm:"not null;index:idx_character_corporation_history_char_start,priority:2" json:"start_date"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"                                                        json:"updated_at"`
}

func (CharacterCorporationHistory) TableName() string { return "character_corporation_history" }
