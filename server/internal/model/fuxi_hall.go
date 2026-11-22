package model

import "time"

// FuxiHallPageKey 展示页键值
type FuxiHallPageKey string

const (
	FuxiHallPageFame FuxiHallPageKey = "hall-of-fame"
)

type FuxiHallFloor struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}

func DefaultFuxiHallFloors() []FuxiHallFloor {
	return []FuxiHallFloor{{Key: "top", Capacity: 8}, {Key: "second", Capacity: 12}, {Key: "first", Capacity: 33}}
}

// FuxiHallPage stores the unified hall configuration and ordered floors.
type FuxiHallPage struct {
	ID              uint            `gorm:"primarykey"                  json:"id"`
	PageKey         string          `gorm:"size:32;uniqueIndex;not null" json:"page_key"`
	Title           string          `gorm:"size:256;not null"           json:"title"`
	Subtitle        string          `gorm:"size:512"                    json:"subtitle"`
	DescriptionHTML string          `gorm:"type:text;not null;default:''" json:"description_html"`
	Floors          []FuxiHallFloor `gorm:"type:jsonb;not null;default:'[]';serializer:json" json:"floors"`
}

func (FuxiHallPage) TableName() string { return "fuxi_hall_page" }

// FuxiHallCard 伏羲大厅成员卡片
type FuxiHallCard struct {
	BaseModel
	QQ                     string                `gorm:"-" json:"qq"`
	ContributionCounts     FuxiHallContributions `gorm:"type:jsonb;not null;default:'{}';serializer:json" json:"-"`
	Contributions          FuxiHallContributions `gorm:"-" json:"contributions"`
	ContributionsUpdatedAt *time.Time            `json:"contributions_updated_at"`
	FleetLedOffset         int64                 `gorm:"not null;default:0" json:"-"`
	SRPOffset              int64                 `gorm:"not null;default:0" json:"-"`
	RecruitmentOffset      int64                 `gorm:"not null;default:0" json:"-"`
	MentoringOffset        int64                 `gorm:"not null;default:0" json:"-"`
	CanEdit                bool                  `gorm:"-" json:"can_edit"`
	TenureMonths           *int                  `gorm:"-" json:"tenure_months"`
	FloorKey               string                `gorm:"size:32;not null;default:'first'" json:"floor_key"`
	IsCurrentManagement    bool                  `gorm:"not null;default:false" json:"is_current_management"`
	PageKey                string                `gorm:"size:32;index;not null"    json:"page_key"`
	Nickname               string                `gorm:"size:256;not null"         json:"nickname"`
	MainCharacterID        int64                 `gorm:"index"                     json:"main_character_id"`
	MainCharacterName      string                `gorm:"size:512;not null"         json:"main_character_name"`
	TitleTags              []string              `gorm:"type:jsonb;not null;default:'[]';serializer:json" json:"title_tags"`
	DescriptionHTML        string                `gorm:"type:text;not null;default:''" json:"description_html"`
	AvatarImage            string                `gorm:"type:text"                 json:"avatar_image"`
	AccentColor            string                `gorm:"size:16;not null;default:'#3b82f6'" json:"accent_color"`
	AvatarShape            string                `gorm:"size:32;not null;default:'circle'" json:"avatar_shape"`
	FontScale              int                   `gorm:"not null;default:14"       json:"font_scale"`
	Visible                bool                  `gorm:"not null;default:true"     json:"visible"`
	SortOrder              int                   `gorm:"index;not null;default:0"  json:"sort_order"`
	WelfareDeliveryOffset  int                   `gorm:"not null;default:0"        json:"-"`
}

func (FuxiHallCard) TableName() string { return "fuxi_hall_card" }
