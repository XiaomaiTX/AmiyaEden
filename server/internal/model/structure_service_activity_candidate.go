package model

// StructureServiceActivityCandidate maps one opaque ESI activity label to a
// possible physical service module. Multiple candidates are intentional: the
// asset service-slot snapshot selects the uniquely installed module.
type StructureServiceActivityCandidate struct {
	BaseModel
	ActivityName  string `gorm:"size:128;uniqueIndex:idx_structure_service_activity_candidate;not null" json:"activity_name"`
	TypeID        int    `gorm:"uniqueIndex:idx_structure_service_activity_candidate;index;not null"                         json:"type_id"`
	SystemManaged bool   `gorm:"not null;default:false"                                                                      json:"system_managed"`
}

func (StructureServiceActivityCandidate) TableName() string {
	return "structure_service_activity_candidate"
}
