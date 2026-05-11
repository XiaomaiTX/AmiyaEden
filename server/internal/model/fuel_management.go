package model

import "time"

// CorpStructureAssignment stores structure-to-fuel-officer assignment.
type CorpStructureAssignment struct {
	CorporationID       int64     `gorm:"not null;index" json:"corporation_id"`
	StructureID         int64     `gorm:"primaryKey" json:"structure_id"`
	AssignedUserID      uint      `gorm:"not null;index" json:"assigned_user_id"`
	AssignedCharacterID int64     `gorm:"not null" json:"assigned_character_id"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CorpStructureAssignment) TableName() string { return "corp_structure_assignment" }

// FuelSalaryPayout stores monthly fuel salary payout records.
type FuelSalaryPayout struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	SettlementMonth string    `gorm:"size:7;not null;uniqueIndex:idx_fuel_salary_month_user,priority:1" json:"settlement_month"`
	UserID          uint      `gorm:"not null;uniqueIndex:idx_fuel_salary_month_user,priority:2;index" json:"user_id"`
	AssignedCount   int       `gorm:"not null" json:"assigned_count"`
	UnitSalary      int       `gorm:"not null" json:"unit_salary"`
	Amount          int       `gorm:"not null" json:"amount"`
	WalletRefID     string    `gorm:"size:64;not null" json:"wallet_ref_id"`
	OperatorUserID  uint      `gorm:"not null;index" json:"operator_user_id"`
	CreatedAt       time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (FuelSalaryPayout) TableName() string { return "fuel_salary_payout" }

const (
	SysConfigFuelSalaryPerStructureMonthly        = "fuel.salary_per_structure_monthly"
	SysConfigDefaultFuelSalaryPerStructureMonthly = 0

	WalletRefFuelSalary = "fuel_salary"
)
