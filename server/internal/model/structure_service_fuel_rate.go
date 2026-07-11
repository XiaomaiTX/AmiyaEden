package model

// StructureServiceFuelRate 建筑「服务名 → 每小时燃料块消耗」映射表
//
// ESI 的军团建筑快照里 services[].name 是 snake_case 的通用服务标识符
// （如 market、industry、clone_bay、reaction），并非服务模块 typeName。
// 因此本表以 service_name 为主键，type_id 仅作记录与同步追溯用途。
type StructureServiceFuelRate struct {
	BaseModel
	ServiceName string  `gorm:"size:64;uniqueIndex;not null" json:"service_name"`       // ESI services[].name（小写）
	TypeID      int     `gorm:"index;not null;default:0"          json:"type_id"`       // 对应服务模块 typeID
	TypeName    string  `gorm:"size:128;not null;default:''"      json:"type_name"`     // 服务模块名（便于运维）
	FuelPerHour float64 `gorm:"not null;default:0"                json:"fuel_per_hour"` // 每小时燃料块消耗（dogma 2109）
}

func (StructureServiceFuelRate) TableName() string { return "structure_service_fuel_rate" }
