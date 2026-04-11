package model

import "time"

// FuxiAdminConfig 伏羲管理人员名录全局配置（单例）
type FuxiAdminConfig struct {
	ID           uint      `gorm:"primarykey"     json:"id"`
	BaseFontSize int       `gorm:"default:14"     json:"base_font_size"` // px, applied to name; title/contact scale via CSS
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FuxiAdminConfig) TableName() string { return "fuxi_admin_config" }

func DefaultFuxiAdminConfig() FuxiAdminConfig {
	return FuxiAdminConfig{ID: 1, BaseFontSize: 14}
}

// FuxiAdminTier 管理层级（高层、中层、基础层等）
type FuxiAdminTier struct {
	ID        uint      `gorm:"primarykey"            json:"id"`
	Name      string    `gorm:"size:256;not null"     json:"name"`
	SortOrder int       `gorm:"default:0"             json:"sort_order"`
	CreatedAt time.Time `gorm:"autoCreateTime"        json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"        json:"updated_at"`
}

func (FuxiAdminTier) TableName() string { return "fuxi_admin_tier" }

// FuxiAdmin 伏羲管理人员
type FuxiAdmin struct {
	BaseModel
	TierID         uint   `gorm:"not null;index"    json:"tier_id"`
	Name           string `gorm:"size:256;not null" json:"name"`
	Title          string `gorm:"size:512"          json:"title"`
	ContactQQ      string `gorm:"size:64"           json:"contact_qq"`
	ContactDiscord string `gorm:"size:256"          json:"contact_discord"`
	CharacterID    int64  `gorm:"default:0"         json:"character_id"`
}

func (FuxiAdmin) TableName() string { return "fuxi_admin" }
