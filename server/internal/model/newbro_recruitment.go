package model

import "time"

// NewbroRecruitment one per generation per user; code is the short URL token.
type NewbroRecruitment struct {
	BaseModel
	UserID      uint      `gorm:"not null;index;index:idx_newbro_recruitment_user_generated,priority:1" json:"user_id"`
	Code        string    `gorm:"size:16;uniqueIndex;not null" json:"code"`
	Source      string    `gorm:"size:32;not null;default:'link';index" json:"source"`
	GeneratedAt time.Time `gorm:"not null;index;index:idx_newbro_recruitment_user_generated,priority:2,sort:desc" json:"generated_at"`
}

func (NewbroRecruitment) TableName() string { return "newbro_recruitment" }

// NewbroRecruitmentEntry one row per external QQ submission against a recruitment link.
type NewbroRecruitmentEntry struct {
	BaseModel
	RecruitmentID uint       `gorm:"not null;index;uniqueIndex:idx_newbro_recruitment_entry_recruitment_qq,priority:1" json:"recruitment_id"`
	QQ            string     `gorm:"size:20;not null;index;uniqueIndex:idx_newbro_recruitment_entry_recruitment_qq,priority:2" json:"qq"`
	EnteredAt     time.Time  `gorm:"not null"                                 json:"entered_at"`
	Source        string     `gorm:"size:32;not null;default:'link';index"     json:"source"`
	Status        string     `gorm:"size:16;not null;default:'ongoing';index" json:"status"`
	MatchedUserID uint       `gorm:"default:0"                                json:"matched_user_id"`
	RewardedAt    *time.Time `gorm:""                                         json:"rewarded_at"`
	WalletRefID   *string    `gorm:"size:128;uniqueIndex"                     json:"wallet_ref_id"`
}

func (NewbroRecruitmentEntry) TableName() string { return "newbro_recruitment_entry" }

const (
	RecruitmentSourceLink            = "link"
	RecruitmentSourceDirectReferral  = "direct_referral"
	RecruitEntrySourceLink           = "link"
	RecruitEntrySourceDirectReferral = "direct_referral"
	RecruitEntryStatusOngoing        = "ongoing"
	RecruitEntryStatusValid          = "valid"
	RecruitEntryStatusStalled        = "stalled"
)

func NormalizeRecruitmentSource(source string) string {
	if source == RecruitmentSourceDirectReferral {
		return source
	}
	return RecruitmentSourceLink
}

func NormalizeRecruitEntrySource(source string) string {
	if source == RecruitEntrySourceDirectReferral {
		return source
	}
	return RecruitEntrySourceLink
}
