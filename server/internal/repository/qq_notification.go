package repository

import (
	"amiya-eden/internal/model"
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QQNotificationRepository struct{ db *gorm.DB }

func NewQQNotificationRepository(db *gorm.DB) *QQNotificationRepository {
	return &QQNotificationRepository{db}
}
func (r *QQNotificationRepository) Transaction(ctx context.Context, tx *gorm.DB, fn func(*gorm.DB) error) error {
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx).Transaction(fn)
}
func (r *QQNotificationRepository) Channels(ctx context.Context) ([]model.QQNotificationChannel, error) {
	rows := []model.QQNotificationChannel{}
	err := r.db.WithContext(ctx).Order("name").Find(&rows).Error
	return rows, err
}
func (r *QQNotificationRepository) Channel(tx *gorm.DB, name string) (model.QQNotificationChannel, error) {
	var row model.QQNotificationChannel
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "name = ?", name).Error
	return row, err
}
func (r *QQNotificationRepository) SaveChannel(tx *gorm.DB, row *model.QQNotificationChannel) error {
	return tx.Save(row).Error
}
func (r *QQNotificationRepository) Insert(tx *gorm.DB, row *model.QQNotification) (bool, error) {
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(row)
	return result.RowsAffected == 1, result.Error
}
func (r *QQNotificationRepository) Get(tx *gorm.DB, id string) (model.QQNotification, error) {
	var row model.QQNotification
	err := tx.First(&row, "id = ?", id).Error
	return row, err
}
func (r *QQNotificationRepository) AddTarget(tx *gorm.DB, row *model.QQNotificationTarget) error {
	return tx.Create(row).Error
}
func (r *QQNotificationRepository) Targets(tx *gorm.DB, id string) ([]model.QQNotificationTarget, error) {
	rows := []model.QQNotificationTarget{}
	err := tx.Where("notification_id = ?", id).Order("group_id").Find(&rows).Error
	return rows, err
}
func (r *QQNotificationRepository) RecentTargets(ctx context.Context) ([]model.QQNotificationTarget, error) {
	rows := []model.QQNotificationTarget{}
	err := r.db.WithContext(ctx).Order("updated_at desc").Limit(200).Find(&rows).Error
	return rows, err
}
func (r *QQNotificationRepository) Complete(tx *gorm.DB, operationID, status, result string) error {
	return tx.Model(&model.QQNotificationTarget{}).Where("operation_id = ? AND status = ?", operationID, "pending").Updates(map[string]interface{}{"status": status, "result": result}).Error
}
func (r *QQNotificationRepository) OperationNotification(tx *gorm.DB, operationID string) (*model.QQNotification, error) {
	var target model.QQNotificationTarget
	err := tx.First(&target, "operation_id = ?", operationID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row, err := r.Get(tx, target.NotificationID)
	return &row, err
}

// Persist cancellation so re-enabling cannot resurrect old notification intentions.
func (r *QQNotificationRepository) CancelPending(tx *gorm.DB, channel string) error {
	ids := tx.Model(&model.QQNotificationTarget{}).Select("operation_id").Where("notification_id IN (?)", tx.Model(&model.QQNotification{}).Select("id").Where("channel = ?", channel))
	return tx.Model(&model.QQBotOperation{}).Where("id IN (?) AND status IN ?", ids, []string{"queued", "leased"}).Update("cancel_requested", true).Error
}

func (r *QQNotificationRepository) CancelNotification(tx *gorm.DB, id string) error {
	ids := tx.Model(&model.QQNotificationTarget{}).Select("operation_id").Where("notification_id = ?", id)
	return tx.Model(&model.QQBotOperation{}).Where("id IN (?) AND status IN ?", ids, []string{"queued", "leased"}).Update("cancel_requested", true).Error
}
