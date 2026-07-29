package model

// StructureServiceActivity maps an opaque ESI activity label to the physical
// service-module type that provides it. The mapping is administrator managed.
type StructureServiceActivity struct {
	BaseModel
	ActivityName string `gorm:"size:128;uniqueIndex;not null" json:"activity_name"`
	TypeID       int    `gorm:"index;not null"                    json:"type_id"`
}

func (StructureServiceActivity) TableName() string { return "structure_service_activity" }
