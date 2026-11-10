package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCharacterAlreadyRegistered = errors.New("该人物已完成注册，请重新登录")

// Contact uniqueness also covers soft-deleted accounts, matching the database indexes.
func (r *UserRepository) GetContactOwner(contact, value string) (*model.User, error) {
	if contact != "qq" && contact != "discord_id" {
		return nil, errors.New("invalid contact field")
	}
	var user model.User
	err := global.DB.Unscoped().Where(contact+" = ?", value).First(&user).Error
	return &user, err
}

// CreateRegistration commits the account, character credentials and initial role together.
func (r *UserRepository) CreateRegistration(user *model.User, character *model.EveCharacter) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		// Serialize different registration tickets for the same verified EVE identity.
		if tx.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", character.CharacterID).Error; err != nil {
				return err
			}
		}
		var existing model.EveCharacter
		err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("character_id = ?", character.CharacterID).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			var owner model.User
			ownerErr := tx.First(&owner, existing.UserID).Error
			if ownerErr == nil {
				return ErrCharacterAlreadyRegistered
			}
			if !errors.Is(ownerErr, gorm.ErrRecordNotFound) {
				return ownerErr
			}
		}
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		character.UserID = user.ID
		if existing.ID != 0 {
			character.ID = existing.ID
			if err := tx.Unscoped().Save(character).Error; err != nil {
				return err
			}
		} else if err := tx.Create(character).Error; err != nil {
			return err
		}
		return tx.Create(&model.UserRole{UserID: user.ID, RoleCode: user.Role}).Error
	})
}
