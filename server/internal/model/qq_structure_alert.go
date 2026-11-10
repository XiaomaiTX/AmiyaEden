package model

import "time"

type QQStructureAlert struct {
	StructureID    int64     `gorm:"primaryKey" json:"structure_id"`
	Kind           string    `gorm:"primaryKey" json:"kind"`
	Active         bool      `json:"active"`
	Version        int64     `json:"version"`
	Target         string    `json:"target"`
	NotificationID string    `json:"notification_id"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (QQStructureAlert) TableName() string { return "qq_structure_alert" }
