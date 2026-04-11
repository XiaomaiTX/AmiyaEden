package model

import "time"

// FuxiAdminConfig 伏羲管理人员名录全局配置（单例）
type FuxiAdminConfig struct {
	ID                  uint      `gorm:"primarykey"                          json:"id"`
	BaseFontSize        int       `gorm:"default:14"                          json:"base_font_size"`
	CardWidth           int       `gorm:"default:240"                         json:"card_width"`
	PageBackgroundColor string    `gorm:"size:16;not null;default:'#10243a'"  json:"page_background_color"`
	CardBackgroundColor string    `gorm:"size:16;not null;default:'#1b324c'"  json:"card_background_color"`
	CardBorderColor     string    `gorm:"size:16;not null;default:'#d9a441'"  json:"card_border_color"`
	TierTitleColor      string    `gorm:"size:16;not null;default:'#f8d26b'"  json:"tier_title_color"`
	NameTextColor       string    `gorm:"size:16;not null;default:'#fff7d6'"  json:"name_text_color"`
	BodyTextColor       string    `gorm:"size:16;not null;default:'#d7dfef'"  json:"body_text_color"`
	CreatedAt           time.Time `gorm:"autoCreateTime"                      json:"created_at"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime"                      json:"updated_at"`
}

func (FuxiAdminConfig) TableName() string { return "fuxi_admin_config" }

func DefaultFuxiAdminConfig() FuxiAdminConfig {
	return FuxiAdminConfig{
		ID:                  1,
		BaseFontSize:        14,
		CardWidth:           240,
		PageBackgroundColor: "#10243a",
		CardBackgroundColor: "#1b324c",
		CardBorderColor:     "#d9a441",
		TierTitleColor:      "#f8d26b",
		NameTextColor:       "#fff7d6",
		BodyTextColor:       "#d7dfef",
	}
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
	Description    string `gorm:"size:1024"         json:"description"`
	ContactQQ      string `gorm:"size:64"           json:"contact_qq"`
	ContactDiscord string `gorm:"size:256"          json:"contact_discord"`
	CharacterID    int64  `gorm:"default:0"         json:"character_id"`
}

func (FuxiAdmin) TableName() string { return "fuxi_admin" }
