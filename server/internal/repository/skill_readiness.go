package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"database/sql"

	"gorm.io/gorm"
)

const readinessBatchSize = 500

type SkillReadinessRepository struct{}

func NewSkillReadinessRepository() *SkillReadinessRepository { return &SkillReadinessRepository{} }

type SkillReadinessCharacter struct {
	CharacterID   int64
	CharacterName string
	CorporationID int64
	UserID        uint
	Nickname      string
	TokenInvalid  bool
}

type SkillReadinessSnapshot struct {
	Plan         model.SkillPlan
	Requirements []model.SkillPlanSkill
	Characters   []SkillReadinessCharacter
	Snapshots    []model.EveCharacterSkill
	Skills       []model.EveCharacterSkills
}

type SkillReadinessFilter struct {
	CorporationIDs []int64
	PlanID         uint
	UserID         uint
	CharacterID    int64
	AllSkills      bool
}

// withReadinessSnapshot keeps affiliation, bindings and transactional skill replacement
// consistent. Name localization is deliberately performed outside this transaction.
func withReadinessSnapshot(ctx context.Context, fn func(*gorm.DB) error) error {
	db := global.DB.WithContext(ctx)
	options := &sql.TxOptions{}
	if db.Name() == "postgres" {
		options.Isolation = sql.LevelRepeatableRead
		options.ReadOnly = true
	}
	return db.Transaction(fn, options)
}

func readinessCharactersQuery(db *gorm.DB, filter SkillReadinessFilter) *gorm.DB {
	query := db.Table("eve_character AS ec").
		Joins(`JOIN "user" AS u ON u.id = ec.user_id AND u.deleted_at IS NULL`).
		Where("ec.deleted_at IS NULL")
	if len(filter.CorporationIDs) > 0 {
		query = query.Where("ec.corporation_id IN ?", filter.CorporationIDs)
	}
	if filter.UserID > 0 {
		query = query.Where("ec.user_id = ?", filter.UserID)
	}
	if filter.CharacterID > 0 {
		query = query.Where("ec.character_id = ?", filter.CharacterID)
	}
	return query
}

func (r *SkillReadinessRepository) Load(ctx context.Context, filter SkillReadinessFilter) (*SkillReadinessSnapshot, error) {
	result := &SkillReadinessSnapshot{}
	err := withReadinessSnapshot(ctx, func(tx *gorm.DB) error {
		if err := tx.First(&result.Plan, filter.PlanID).Error; err != nil {
			return err
		}
		if err := tx.Where("skill_plan_id = ?", filter.PlanID).Order("sort ASC, id ASC").Find(&result.Requirements).Error; err != nil {
			return err
		}
		if err := readinessCharactersQuery(tx, filter).
			Select("ec.character_id, ec.character_name, ec.corporation_id, ec.user_id, ec.token_invalid, u.nickname").
			Order("ec.user_id ASC, ec.character_name ASC, ec.character_id ASC").Scan(&result.Characters).Error; err != nil {
			return err
		}
		skillIDs := make([]int, 0, len(result.Requirements))
		for _, requirement := range result.Requirements {
			skillIDs = append(skillIDs, requirement.SkillTypeID)
		}
		for start := 0; start < len(result.Characters); start += readinessBatchSize {
			end := min(start+readinessBatchSize, len(result.Characters))
			ids := make([]int64, 0, end-start)
			for _, character := range result.Characters[start:end] {
				ids = append(ids, character.CharacterID)
			}
			var snapshots []model.EveCharacterSkill
			if err := tx.Where("character_id IN ?", ids).Find(&snapshots).Error; err != nil {
				return err
			}
			result.Snapshots = append(result.Snapshots, snapshots...)
			if !filter.AllSkills && len(skillIDs) == 0 {
				continue
			}
			query := tx.Where("character_id IN ?", ids)
			if !filter.AllSkills {
				query = query.Where("skill_id IN ?", skillIDs)
			}
			var skills []model.EveCharacterSkills
			if err := query.Order("character_id ASC, skill_id ASC").Find(&skills).Error; err != nil {
				return err
			}
			result.Skills = append(result.Skills, skills...)
		}
		return nil
	})
	return result, err
}

func (r *SkillReadinessRepository) Options(ctx context.Context) ([]int64, []model.SkillPlan, error) {
	ids := make([]int64, 0)
	plans := make([]model.SkillPlan, 0)
	err := withReadinessSnapshot(ctx, func(tx *gorm.DB) error {
		if err := readinessCharactersQuery(tx, SkillReadinessFilter{}).
			Where("ec.corporation_id > 0").Distinct("ec.corporation_id").Order("ec.corporation_id ASC").Pluck("ec.corporation_id", &ids).Error; err != nil {
			return err
		}
		return tx.Where("plan_scope = ?", model.SkillPlanScopeCorp).Order("sort_order ASC, id DESC").Find(&plans).Error
	})
	return ids, plans, err
}
