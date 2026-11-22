package model

import "time"

type Announcement struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:200" json:"title"`
	BodyHTML  string    `gorm:"type:text" json:"body_html,omitempty"`
	AuthorID  uint      `json:"author_id"`
	Hidden    bool      `gorm:"index" json:"hidden"`
	Version   uint      `json:"version"`
	Revision  uint      `json:"revision"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Announcement) TableName() string { return "announcement" }

type AnnouncementRead struct {
	AnnouncementID uint `gorm:"primaryKey;autoIncrement:false"`
	UserID         uint `gorm:"primaryKey;autoIncrement:false"`
	Version        uint
}

func (AnnouncementRead) TableName() string { return "announcement_read" }

type AnnouncementImage struct {
	AnnouncementID uint   `gorm:"primaryKey;autoIncrement:false"`
	Digest         string `gorm:"primaryKey;size:64"`
	Data           []byte
}

func (AnnouncementImage) TableName() string { return "announcement_image" }
