package esimodel

type EveStructure struct {
	StructureID   int64   `gorm:"primaryKey" json:"structure_id"`
	StructureName string  `gorm:""           json:"structure_name"`
	OwnerID       int64   `gorm:"index"      json:"owner_id"`
	TypeID        int64   `gorm:""           json:"type_id"`
	SolarSystemID int64   `gorm:""           json:"solar_system_id"`
	X             float64 `gorm:""           json:"x"`
	Y             float64 `gorm:""           json:"y"`
	Z             float64 `gorm:""           json:"z"`
	UpdateAt      int64   `gorm:""           json:"update_at"`
}

func (EveStructure) TableName() string { return "eve_structures" }

type CorpStructureInfo struct {
	CorporationID      int64  `gorm:"index"      json:"corporation_id"`
	StructureID        int64  `gorm:"primaryKey" json:"structure_id"`
	FuelExpires        string `gorm:""           json:"fuel_expires"`
	Name               string `gorm:""           json:"name"`
	NextReinforceApply string `gorm:""           json:"next_reinforce_apply"`
	NextReinforceHour  int    `gorm:""           json:"next_reinforce_hour"`
	ProfileID          int64  `gorm:""           json:"profile_id"`
	ReinforceHour      int    `gorm:""           json:"reinforce_hour"`
	State              string `gorm:""           json:"state"`
	StateTimerEnd      string `gorm:""           json:"state_timer_end"`
	StateTimerStart    string `gorm:""           json:"state_timer_start"`
	SystemID           int64  `gorm:""           json:"system_id"`
	TypeID             int64  `gorm:""           json:"type_id"`
	UnanchorsAt        string `gorm:""           json:"unanchors_at"`
	UpdateAt           int64  `gorm:""           json:"update_at"`
}
