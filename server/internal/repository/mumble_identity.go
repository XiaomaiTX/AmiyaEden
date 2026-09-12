package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"time"
)

type MumbleIdentityRepository struct{}

func NewMumbleIdentityRepository() *MumbleIdentityRepository { return &MumbleIdentityRepository{} }

func (r *MumbleIdentityRepository) GetBySeatUserID(userID uint) (*model.MumbleIdentity, error) {
	var identity model.MumbleIdentity
	err := global.DB.Where("seat_user_id = ?", userID).First(&identity).Error
	return &identity, err
}

func (r *MumbleIdentityRepository) GetByStableUserID(userID uint32) (*model.MumbleIdentity, error) {
	var identity model.MumbleIdentity
	err := global.DB.First(&identity, uint(userID)).Error
	return &identity, err
}

func (r *MumbleIdentityRepository) Create(identity *model.MumbleIdentity) error {
	return global.DB.Create(identity).Error
}

func (r *MumbleIdentityRepository) UpdateCredential(userID uint, hash string, enabled bool, version uint, rotatedAt time.Time) error {
	return global.DB.Model(&model.MumbleIdentity{}).Where("seat_user_id = ?", userID).Updates(map[string]any{
		"credential_hash": hash, "credential_enabled": enabled, "credential_version": version,
		"identity_version": version, "password_rotated_at": rotatedAt,
	}).Error
}

func (r *MumbleIdentityRepository) Revoke(userID uint, version uint) error {
	return global.DB.Model(&model.MumbleIdentity{}).Where("seat_user_id = ?", userID).Updates(map[string]any{
		"credential_hash": "", "credential_enabled": false, "credential_version": version, "identity_version": version,
	}).Error
}

func (r *MumbleIdentityRepository) TouchLastUsed(identityID uint, at time.Time) error {
	return global.DB.Model(&model.MumbleIdentity{}).Where("id = ?", identityID).Update("last_used_at", at).Error
}
