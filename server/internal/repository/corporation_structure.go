package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"encoding/json"
	"strings"

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
