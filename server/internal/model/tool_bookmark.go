package model

// ToolBookmark 常用工具网站书签
type ToolBookmark struct {
	BaseModel
	Name        string `gorm:"size:128;not null"              json:"name"`
	URL         string `gorm:"type:text;not null"             json:"url"`
	Description string `gorm:"size:1024;not null;default:''"  json:"description"`
	LogoURL     string `gorm:"type:text;not null;default:''"  json:"logo_url"`
	LogoSource  string `gorm:"size:16;not null;default:''"    json:"logo_source"`
	IsEnabled   bool   `gorm:"not null;default:true;index"    json:"is_enabled"`
	SortOrder   int    `gorm:"not null;default:0;index"       json:"sort_order"`
	CreatedBy   uint   `gorm:"index;not null;default:0"       json:"created_by"`
}

func (ToolBookmark) TableName() string { return "tool_bookmark" }
