package model

import "time"

// QQNotificationChannel is a business destination set, independent of transport settings.
type QQNotificationChannel struct {
	Name      string    `gorm:"primaryKey;size:32" json:"name"`
	Enabled   bool      `json:"enabled"`
	GroupIDs  string    `gorm:"type:text" json:"group_ids"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (QQNotificationChannel) TableName() string { return "qq_notification_channel" }

type QQNotification struct {
	ID            string    `gorm:"primaryKey;size:64" json:"id"`
	Channel       string    `json:"channel"`
	BusinessKey   string    `json:"business_key"`
	CorrelationID string    `json:"correlation_id"`
	ContentHash   string    `json:"-"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
}

func (QQNotification) TableName() string { return "qq_notification" }

type QQNotificationTarget struct {
	Channel        string    `json:"channel"`
	CorrelationID  string    `json:"correlation_id"`
	BusinessKey    string    `json:"business_key"`
	NotificationID string    `gorm:"primaryKey;size:64" json:"notification_id"`
	GroupID        int64     `gorm:"primaryKey" json:"group_id"`
	OperationID    string    `gorm:"uniqueIndex" json:"operation_id"`
	Status         string    `json:"status"`
	Result         string    `gorm:"type:text" json:"result"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (QQNotificationTarget) TableName() string { return "qq_notification_target" }
