package model

import "time"

const (
	BlacklistKindCharacter = "character"
	BlacklistKindQQ        = "qq"
	BlacklistKindDiscord   = "discord"

	BlacklistUnresolvedNotFound      = "not_found"
	BlacklistUnresolvedInvalidFormat = "invalid_format"
)

// BlacklistEntry is one independent check value mirrored from the alliance
// blacklist sheet. Characters, QQ numbers and Discord names are deliberately
// unlinked: the list answers "is this value blacklisted, and why".
// Value is the normalized match key (character ID in decimal, QQ digits, or the
// lowercased Discord name); Display keeps the human-readable spelling.
type BlacklistEntry struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	Kind      string    `gorm:"size:16;not null;uniqueIndex:idx_blacklist_entry_kind_value,priority:1" json:"kind"`
	Value     string    `gorm:"type:text;not null;uniqueIndex:idx_blacklist_entry_kind_value,priority:2" json:"value"`
	Display   string    `gorm:"type:text;not null" json:"display"`
	Reason    string    `gorm:"type:text;not null" json:"reason"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (BlacklistEntry) TableName() string { return "blacklist_entry" }

// BlacklistUnresolvedName remembers sheet names that ESI does not know as a
// character, or that cannot be an EVE character name, so later refreshes skip
// them while they stay on the sheet.
type BlacklistUnresolvedName struct {
	ID        uint   `gorm:"primaryKey"`
	NameKey   string `gorm:"type:text;not null;uniqueIndex:idx_blacklist_unresolved_name_key"`
	Name      string `gorm:"type:text;not null"`
	Kind      string `gorm:"size:20;not null"`
	CreatedAt time.Time
}

func (BlacklistUnresolvedName) TableName() string { return "blacklist_unresolved_name" }

// BlacklistRefreshRun is the summary of one successful refresh; the page shows
// the latest row.
type BlacklistRefreshRun struct {
	ID                uint      `gorm:"primaryKey"`
	OperatorUserID    uint      `gorm:"not null"`
	FinishedAt        time.Time `gorm:"not null;index"`
	Characters        int       `gorm:"not null"`
	QQs               int       `gorm:"column:qqs;not null"`
	Discords          int       `gorm:"not null"`
	Added             int       `gorm:"not null"`
	Updated           int       `gorm:"not null"`
	Removed           int       `gorm:"not null"`
	NewUnresolved     int       `gorm:"not null"`
	SkippedUnresolved int       `gorm:"not null"`
	// Names whose ESI lookup failed or ran out of time; retried next refresh.
	LookupFailed int `gorm:"not null;default:0"`
}

func (BlacklistRefreshRun) TableName() string { return "blacklist_refresh_run" }
