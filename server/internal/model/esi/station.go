package esimodel

// EveStation NPC 空间站信息缓存表
type EveStation struct {
	StationID     int64   `gorm:"primaryKey" json:"station_id"`
	StationName   string  `gorm:""           json:"station_name"`
	OwnerID       int64   `gorm:"index"      json:"owner_id"`
	TypeID        int64   `gorm:""           json:"type_id"`
	SolarSystemID int64   `gorm:""           json:"solar_system_id"`
	X             float64 `gorm:""           json:"x"`
	Y             float64 `gorm:""           json:"y"`
	Z             float64 `gorm:""           json:"z"`
	UpdateAt      int64   `gorm:""           json:"update_at"`
}

func (EveStation) TableName() string { return "eve_stations" }
