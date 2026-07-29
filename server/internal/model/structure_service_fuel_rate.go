package model

// StructureServiceFuelRate is the service-module catalogue. ServiceName is an
// application-owned stable key; TypeID, rather than the ESI activity label, is
// the fuel-rate identity.
type StructureServiceFuelRate struct {
	BaseModel
	ServiceName  string  `gorm:"size:64;uniqueIndex;not null" json:"service_name"`
	TypeID       int     `gorm:"index;not null;default:0"          json:"type_id"`
	TypeName     string  `gorm:"size:128;not null;default:''"      json:"type_name"`     // 服务模块名（便于运维）
	FuelPerHour  float64 `gorm:"not null;default:0"                json:"fuel_per_hour"` // 每小时燃料块消耗（dogma 2109）
	FuelCategory string  `gorm:"size:32;not null;default:'other'" json:"fuel_category"`
}

func (StructureServiceFuelRate) TableName() string { return "structure_service_fuel_rate" }
