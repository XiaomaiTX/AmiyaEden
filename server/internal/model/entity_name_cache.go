package model

import "time"

const (
	EntityNameTypeCharacter   = "character"
	EntityNameTypeCorporation = "corporation"
	EntityNameTypeAlliance    = "alliance"
	EntityNameTypeUnknown     = "unknown"

	EntityNameSourceESI = "esi"
)

// EveEntityNameCache 缓存 character/corporation/alliance 名称解析结果。
type EveEntityNameCache struct {
	BaseModel
	EntityID       int64     `gorm:"uniqueIndex;not null"                json:"entity_id"`
	EntityType     string    `gorm:"size:32;not null;default:'unknown'"   json:"entity_type"`
	Name           string    `gorm:"size:256;not null"                    json:"name"`
	Source         string    `gorm:"size:32;not null;default:'esi'"       json:"source"`
	LastResolvedAt time.Time `gorm:"not null"                             json:"last_resolved_at"`
	ExpiresAt      time.Time `gorm:"index;not null"                       json:"expires_at"`
}

func (EveEntityNameCache) TableName() string { return "eve_entity_name_cache" }
