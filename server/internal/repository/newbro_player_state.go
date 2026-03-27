package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"

	"gorm.io/gorm/clause"
)

type NewbroPlayerStateRepository struct{}

func NewNewbroPlayerStateRepository() *NewbroPlayerStateRepository {
	return &NewbroPlayerStateRepository{}
}

func (r *NewbroPlayerStateRepository) GetByUserID(userID uint) (*model.NewbroPlayerState, error) {
	var state model.NewbroPlayerState
	err := global.DB.Where("user_id = ?", userID).First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *NewbroPlayerStateRepository) Save(state *model.NewbroPlayerState) error {
	return global.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"is_currently_newbro": clause.Expr{SQL: "EXCLUDED.is_currently_newbro"},
			"evaluated_at":        clause.Expr{SQL: "EXCLUDED.evaluated_at"},
			"rule_version":        clause.Expr{SQL: "EXCLUDED.rule_version"},
			"disqualified_reason": clause.Expr{SQL: "EXCLUDED.disqualified_reason"},
			"updated_at":          clause.Expr{SQL: "EXCLUDED.updated_at"},
			"deleted_at":          nil,
		}),
	}).Create(state).Error
}
