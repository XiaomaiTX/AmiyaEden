package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"encoding/json"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CorporationStructureRepository struct{}

func NewCorporationStructureRepository() *CorporationStructureRepository {
	return &CorporationStructureRepository{}
}

type DirectorCharacterOption struct {
	UserID        uint  `json:"user_id"`
	CharacterID   int64 `json:"character_id"`
	CharacterName string
	CorporationID int64 `json:"corporation_id"`
}

type StructureServiceSnapshot struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type FuelOfficerUserOption struct {
	UserID        uint   `json:"user_id"`
	DisplayName   string `json:"display_name"`
	CharacterID   int64  `json:"character_id"`
	CharacterName string `json:"character_name"`
}

func (r *CorporationStructureRepository) ListDirectorCharactersByCorporations(
	corporationIDs []int64,
) ([]DirectorCharacterOption, error) {
	options := make([]DirectorCharacterOption, 0)
	if len(corporationIDs) == 0 {
		return options, nil
	}

	err := global.DB.Table(`eve_character AS ec`).
		Select(`ec.user_id, ec.character_id, ec.character_name, ec.corporation_id`).
		Joins(`JOIN user_role AS ur ON ur.user_id = ec.user_id`).
		Joins(`JOIN eve_character_corp_role AS ecr ON ecr.character_id = ec.character_id`).
		Where(`ur.role_code IN ?`, []string{model.RoleAdmin, model.RoleSuperAdmin}).
		Where(`ec.corporation_id IN ?`, corporationIDs).
		Where(`LOWER(ecr.corp_role) = ?`, "director").
		Group(`ec.user_id, ec.character_id, ec.character_name, ec.corporation_id`).
		Order(`ec.corporation_id ASC, ec.character_name ASC`).
		Scan(&options).Error
	return options, err
}

func (r *CorporationStructureRepository) ListCorpStructures(
	corporationIDs []int64,
) ([]model.CorpStructureInfo, error) {
	rows := make([]model.CorpStructureInfo, 0)
	if len(corporationIDs) == 0 {
		return rows, nil
	}

	err := global.DB.Where(`corporation_id IN ?`, corporationIDs).
		Order(`corporation_id ASC, structure_id ASC`).
		Find(&rows).Error
	return rows, err
}

func (r *CorporationStructureRepository) UpsertCorpStructures(records []model.CorpStructureInfo) error {
	if len(records) == 0 {
		return nil
	}
	return global.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&records).Error
}

func (r *CorporationStructureRepository) DeleteCorpStructuresByCorporationIDs(corporationIDs []int64) error {
	if len(corporationIDs) == 0 {
		return nil
	}
	return global.DB.Where("corporation_id IN ?", corporationIDs).Delete(&model.CorpStructureInfo{}).Error
}

func (r *CorporationStructureRepository) DeleteCorpStructuresNotInCorporationIDs(corporationIDs []int64) (int64, error) {
	query := global.DB.Model(&model.CorpStructureInfo{})
	if len(corporationIDs) > 0 {
		query = query.Where("corporation_id NOT IN ?", corporationIDs)
	} else {
		query = query.Where("1 = 1")
	}
	result := query.Delete(&model.CorpStructureInfo{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *CorporationStructureRepository) ListFuelOfficerUsersByCorporations(
	corporationIDs []int64,
) ([]FuelOfficerUserOption, error) {
	options := make([]FuelOfficerUserOption, 0)
	if len(corporationIDs) == 0 {
		return options, nil
	}

	err := buildFuelOfficerUsersByCorporationsQuery(global.DB, corporationIDs).
		Scan(&options).Error
	return options, err
}

func buildFuelOfficerUsersByCorporationsQuery(db *gorm.DB, corporationIDs []int64) *gorm.DB {
	return db.Table(`"user" AS u`).
		Select(`
			u.id AS user_id,
			COALESCE(NULLIF(TRIM(u.nickname), ''), MIN(ec.character_name), 'User-' || u.id) AS display_name,
			MIN(ec.character_id) AS character_id,
			MIN(ec.character_name) AS character_name
		`).
		Joins(`JOIN user_role AS ur ON ur.user_id = u.id`).
		Joins(`JOIN eve_character AS ec ON ec.user_id = u.id`).
		Where(`ur.role_code = ?`, model.RoleFuelOfficer).
		Where(`ec.corporation_id IN ?`, corporationIDs).
		Group(`u.id, u.nickname`).
		Order(`display_name ASC`)
}

func (r *CorporationStructureRepository) ListAssignmentsByCorporations(
	corporationIDs []int64,
) ([]model.CorpStructureAssignment, error) {
	rows := make([]model.CorpStructureAssignment, 0)
	if len(corporationIDs) == 0 {
		return rows, nil
	}
	err := global.DB.Where(`corporation_id IN ?`, corporationIDs).
		Find(&rows).Error
	return rows, err
}

func (r *CorporationStructureRepository) ListAssignmentsByUserID(userID uint) ([]model.CorpStructureAssignment, error) {
	rows := make([]model.CorpStructureAssignment, 0)
	err := global.DB.Where(`assigned_user_id = ?`, userID).Find(&rows).Error
	return rows, err
}

func (r *CorporationStructureRepository) UpsertAssignments(records []model.CorpStructureAssignment) error {
	if len(records) == 0 {
		return nil
	}
	return global.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&records).Error
}

func (r *CorporationStructureRepository) DeleteAssignmentsByStructureIDs(structureIDs []int64) error {
	if len(structureIDs) == 0 {
		return nil
	}
	return global.DB.Where("structure_id IN ?", structureIDs).Delete(&model.CorpStructureAssignment{}).Error
}

func (r *CorporationStructureRepository) CreateFuelSalaryPayout(tx *model.FuelSalaryPayout) error {
	return global.DB.Create(tx).Error
}

func (r *CorporationStructureRepository) CreateFuelSalaryPayoutTx(dbTx *gorm.DB, tx *model.FuelSalaryPayout) error {
	return dbTx.Create(tx).Error
}

func (r *CorporationStructureRepository) ExistsFuelSalaryPayout(settlementMonth string, userID uint) (bool, error) {
	var count int64
	err := global.DB.Model(&model.FuelSalaryPayout{}).
		Where("settlement_month = ? AND user_id = ?", settlementMonth, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *CorporationStructureRepository) UpsertStructures(records []model.EveStructure) error {
	if len(records) == 0 {
		return nil
	}
	return global.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&records).Error
}

func DecodeStructureServices(raw string) []StructureServiceSnapshot {
	if strings.TrimSpace(raw) == "" {
		return []StructureServiceSnapshot{}
	}
	var services []StructureServiceSnapshot
	if err := json.Unmarshal([]byte(raw), &services); err != nil {
		return []StructureServiceSnapshot{}
	}
	return services
}
