package model

import "time"

const (
	GalaxyRegistryEntryStatusActive    = "active"
	GalaxyRegistryEntryStatusCompleted = "completed"
)

const (
	GalaxyRegistryValidationPending   = "pending"
	GalaxyRegistryValidationValid     = "valid"
	GalaxyRegistryValidationViolation = "violation"
)

const (
	GalaxyRegistryStatusIdle    = "idle"
	GalaxyRegistryStatusBusy    = "busy"
	GalaxyRegistryStatusOverdue = "overdue"
)

const (
	GalaxyRegistryViolationNoBountyInWindow     = "no_bounty_in_window"
	GalaxyRegistryViolationBountyBelowThreshold = "bounty_below_threshold"
)

const GalaxyRegistryDefaultMinBountyAmount = 10_000_000

type GalaxyRegistrySystem struct {
	BaseModel
	SolarSystemID     int64   `gorm:"uniqueIndex;not null" json:"solar_system_id"`
	SolarSystemName   string  `gorm:"size:120;not null"    json:"solar_system_name"`
	RegionID          int64   `gorm:"not null"             json:"region_id"`
	RegionName        string  `gorm:"size:120;not null"    json:"region_name"`
	ConstellationID   int64   `gorm:"not null"             json:"constellation_id"`
	ConstellationName string  `gorm:"size:120;not null"    json:"constellation_name"`
	Security          float64 `gorm:"not null"             json:"security"`
	Note              string  `gorm:"size:500"             json:"note"`
	MinBountyAmount   float64 `gorm:"not null;default:10000000" json:"min_bounty_amount"`
	IsEnabled         bool    `gorm:"not null;default:true" json:"is_enabled"`
}

func (GalaxyRegistrySystem) TableName() string {
	return "galaxy_registry_system"
}

type GalaxyRegistryEntry struct {
	BaseModel
	SystemConfigID        uint       `gorm:"index;not null"                         json:"system_config_id"`
	SolarSystemID         int64      `gorm:"index;not null"                         json:"solar_system_id"`
	SolarSystemName       string     `gorm:"size:120;not null"                      json:"solar_system_name"`
	CaptainUserID         uint       `gorm:"index;not null"                         json:"captain_user_id"`
	CaptainCharacterID    int64      `gorm:"not null"                               json:"captain_character_id"`
	CaptainCharacterName  string     `gorm:"size:120;not null"                      json:"captain_character_name"`
	Status                string     `gorm:"size:20;index;not null"                 json:"status"`
	ValidationStatus      string     `gorm:"size:20;index;not null"                 json:"validation_status"`
	ExpectedEndAt         time.Time  `gorm:"not null"                               json:"expected_end_at"`
	ActualStartAt         time.Time  `gorm:"not null"                               json:"actual_start_at"`
	ActualEndAt           *time.Time `json:"actual_end_at"`
	EndedByUserID         uint       `gorm:"default:0"                              json:"ended_by_user_id"`
	ForceEndedByAdmin     bool       `gorm:"not null;default:false"                 json:"force_ended_by_admin"`
	FrozenMinBountyAmount float64    `gorm:"not null;default:10000000"              json:"frozen_min_bounty_amount"`
	ValidatedAt           *time.Time `json:"validated_at"`
	ValidatedBountyAmount float64    `gorm:"not null;default:0"                     json:"validated_bounty_amount"`
	ValidatedBountyCount  int        `gorm:"not null;default:0"                     json:"validated_bounty_count"`
	ViolationReason       string     `gorm:"size:64"                                json:"violation_reason"`
}

func (GalaxyRegistryEntry) TableName() string {
	return "galaxy_registry_entry"
}
