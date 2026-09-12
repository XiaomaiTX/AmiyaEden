package model

import "time"

const (
	EntityTickerTypeCorporation = "corporation"
	EntityTickerTypeAlliance    = "alliance"
)

// EveEntityTickerCache 保存仅供展示使用的 ESI 归属缩写快照。
// 它由归属 ESI 任务刷新，Mumble 认证和身份重验不得访问 ESI。
type EveEntityTickerCache struct {
	BaseModel
	EntityID       int64     `gorm:"uniqueIndex:idx_entity_ticker_type_id;not null" json:"entity_id"`
	EntityType     string    `gorm:"size:32;uniqueIndex:idx_entity_ticker_type_id;not null" json:"entity_type"`
	Ticker         string    `gorm:"size:32;not null"                                  json:"ticker"`
	LastResolvedAt time.Time `gorm:"not null"                                          json:"last_resolved_at"`
	ExpiresAt      time.Time `gorm:"index;not null"                                   json:"expires_at"`
}

func (EveEntityTickerCache) TableName() string { return "eve_entity_ticker_cache" }
