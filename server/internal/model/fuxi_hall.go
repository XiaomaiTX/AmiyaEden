package model

// FuxiHallPageKey 展示页键值
type FuxiHallPageKey string

const (
	FuxiHallPageLeadership   FuxiHallPageKey = "leadership"
	FuxiHallPageContributors FuxiHallPageKey = "contributors"
)

// FuxiHallPage 伏羲大厅页面配置（按 page_key 固定两页）
type FuxiHallPage struct {
	ID              uint   `gorm:"primarykey"                  json:"id"`
	PageKey         string `gorm:"size:32;uniqueIndex;not null" json:"page_key"`
	Title           string `gorm:"size:256;not null"           json:"title"`
	Subtitle        string `gorm:"size:512"                    json:"subtitle"`
	DescriptionHTML string `gorm:"type:text;not null;default:''" json:"description_html"`
}

func (FuxiHallPage) TableName() string { return "fuxi_hall_page" }

// FuxiHallCard 伏羲大厅成员卡片
type FuxiHallCard struct {
	BaseModel
	PageKey           string   `gorm:"size:32;index;not null"    json:"page_key"`
	Nickname          string   `gorm:"size:256;not null"         json:"nickname"`
	MainCharacterID   int64    `gorm:"index"                     json:"main_character_id"`
	MainCharacterName string   `gorm:"size:512;not null"         json:"main_character_name"`
	TitleTags         []string `gorm:"type:jsonb;not null;default:'[]';serializer:json" json:"title_tags"`
	DescriptionHTML   string   `gorm:"type:text;not null;default:''" json:"description_html"`
	AvatarImage       string   `gorm:"type:text"                 json:"avatar_image"`
	AccentColor       string   `gorm:"size:16;not null;default:'#3b82f6'" json:"accent_color"`
	AvatarShape       string   `gorm:"size:32;not null;default:'circle'" json:"avatar_shape"`
	FontScale         int      `gorm:"not null;default:14"       json:"font_scale"`
	Visible           bool     `gorm:"not null;default:true"     json:"visible"`
	SortOrder         int      `gorm:"index;not null;default:0"  json:"sort_order"`
}

func (FuxiHallCard) TableName() string { return "fuxi_hall_card" }
