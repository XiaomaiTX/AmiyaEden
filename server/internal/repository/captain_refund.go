package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type CaptainRefundRepository struct{}

func NewCaptainRefundRepository() *CaptainRefundRepository { return &CaptainRefundRepository{} }
func (r *CaptainRefundRepository) Transaction(fn func(*gorm.DB) error) error {
	return global.DB.Transaction(fn)
}
func (r *CaptainRefundRepository) FindCharacters(names []string) ([]model.EveCharacter, error) {
	rows := []model.EveCharacter{}
	if len(names) == 0 {
		return rows, nil
	}
	err := global.DB.Select("eve_character.*").
		Joins(`JOIN "user" AS refund_user ON refund_user.id = eve_character.user_id AND refund_user.deleted_at IS NULL`).
		Where("eve_character.user_id > 0 AND LOWER(eve_character.character_name) IN ?", names).
		Order("eve_character.character_id").Find(&rows).Error
	return rows, err
}
func (r *CaptainRefundRepository) Journals(ids []int64, systemID int64, start, end time.Time) ([]model.EVECharacterWalletJournal, error) {
	rows := []model.EVECharacterWalletJournal{}
	err := global.DB.Where("character_id IN ? AND context_id = ? AND context_id_type = ? AND date >= ? AND date < ? AND ref_type = ? AND amount > 0", ids, systemID, "solar_system_id", start, end, "bounty_prizes").Order("date, id").Find(&rows).Error
	return rows, err
}
func (r *CaptainRefundRepository) WalletUpdates(ids []int64) ([]model.EVECharacterWallet, error) {
	rows := []model.EVECharacterWallet{}
	err := global.DB.Where("character_id IN ?", ids).Find(&rows).Error
	return rows, err
}
func (r *CaptainRefundRepository) Claims(ids, journalIDs []int64) ([]model.CaptainRefundClaim, error) {
	rows := []model.CaptainRefundClaim{}
	if len(journalIDs) == 0 {
		return rows, nil
	}
	err := global.DB.Where("character_id IN ? AND journal_id IN ?", ids, journalIDs).Find(&rows).Error
	return rows, err
}
func (r *CaptainRefundRepository) Create(tx *gorm.DB, row *model.CaptainRefund, claims []model.CaptainRefundClaim) error {
	if err := tx.Create(row).Error; err != nil {
		return err
	}
	for i := range claims {
		claims[i].RefundID = row.ID
	}
	return tx.Create(&claims).Error
}
func (r *CaptainRefundRepository) List(userID uint, page, size int) ([]model.CaptainRefund, int64, error) {
	q := global.DB.Model(&model.CaptainRefund{})
	if userID != 0 {
		q = q.Where("user_id = ?", userID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []model.CaptainRefund{}
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}
func (r *CaptainRefundRepository) Get(id uint) (*model.CaptainRefund, error) {
	var row model.CaptainRefund
	err := global.DB.First(&row, id).Error
	return &row, err
}
func (r *CaptainRefundRepository) Lock(tx *gorm.DB, id uint) (*model.CaptainRefund, error) {
	var row model.CaptainRefund
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error
	return &row, err
}
func (r *CaptainRefundRepository) Review(tx *gorm.DB, row *model.CaptainRefund) error {
	result := tx.Model(&model.CaptainRefund{}).Where("id = ? AND status = ?", row.ID, "pending").Updates(map[string]interface{}{"status": row.Status, "reviewed_at": row.ReviewedAt, "reviewed_by": row.ReviewedBy, "reject_reason": row.RejectReason, "wallet_ref_id": row.WalletRefID})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrInvalidTransaction
	}
	if row.Status == "rejected" {
		return tx.Where("refund_id = ?", row.ID).Delete(&model.CaptainRefundClaim{}).Error
	}
	return nil
}
