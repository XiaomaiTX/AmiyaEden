package repository

import (
	"amiya-eden/internal/model"
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
	"time"
)

type QQGroupBusinessRepository struct{ db *gorm.DB }

func NewQQGroupBusinessRepository(db *gorm.DB) *QQGroupBusinessRepository {
	return &QQGroupBusinessRepository{db}
}
func (r *QQGroupBusinessRepository) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
func (r *QQGroupBusinessRepository) Rule(tx *gorm.DB, id int64) (model.QQGroupRule, error) {
	row := model.QQGroupRule{GroupID: id}
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "group_id = ?", id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return row, nil
	}
	return row, e
}
func (r *QQGroupBusinessRepository) SaveRule(tx *gorm.DB, row *model.QQGroupRule) error {
	return tx.Save(row).Error
}
func (r *QQGroupBusinessRepository) Application(tx *gorm.DB, row *model.QQGroupApplication) error {
	if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error; e != nil {
		return e
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(row, "id = ?", row.ID).Error
}
func (r *QQGroupBusinessRepository) SaveApplication(tx *gorm.DB, row *model.QQGroupApplication) error {
	return tx.Save(row).Error
}
func (r *QQGroupBusinessRepository) ApplicationForOperation(tx *gorm.DB, id string) (model.QQGroupApplication, error) {
	var row model.QQGroupApplication
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "operation_id = ?", id).Error
	return row, e
}
func (r *QQGroupBusinessRepository) CancelGroup(tx *gorm.DB, id int64) error {
	ids := tx.Model(&model.QQGroupApplication{}).Select("operation_id").Where("group_id = ?", id)
	return tx.Model(&model.QQBotOperation{}).Where("id IN (?) AND status IN ?", ids, []string{"queued", "leased"}).Update("cancel_requested", true).Error
}

type QQQualificationData struct {
	Users     []model.User
	Character model.EveCharacter
	Roles     []model.UserRole
}

func (r *QQGroupBusinessRepository) Qualification(tx *gorm.DB, qq int64) (QQQualificationData, error) {
	var d QQQualificationData
	if e := tx.Where("qq = ?", strconv.FormatInt(qq, 10)).Find(&d.Users).Error; e != nil {
		return d, e
	}
	if len(d.Users) != 1 || d.Users[0].PrimaryCharacterID <= 0 {
		return d, nil
	}
	u := d.Users[0]
	e := tx.First(&d.Character, "character_id = ? AND user_id = ?", u.PrimaryCharacterID, u.ID).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return d, nil
	}
	if e != nil {
		return d, e
	}
	e = tx.Where("user_id = ?", u.ID).Find(&d.Roles).Error
	return d, e
}
func (r *QQGroupBusinessRepository) RecheckIDs(ctx context.Context, now time.Time) ([]string, error) {
	ids := []string{}
	e := r.db.WithContext(ctx).Model(&model.QQGroupApplication{}).Where("status = ? AND recheck_at <= ?", "waiting", now).Limit(100).Pluck("id", &ids).Error
	return ids, e
}
func (r *QQGroupBusinessRepository) GetApplication(tx *gorm.DB, id string) (model.QQGroupApplication, error) {
	var row model.QQGroupApplication
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error
	return row, e
}
func (r *QQGroupBusinessRepository) Rules(ctx context.Context) ([]model.QQGroupRule, error) {
	rows := []model.QQGroupRule{}
	e := r.db.WithContext(ctx).Order("group_id").Find(&rows).Error
	return rows, e
}
func (r *QQGroupBusinessRepository) Applications(ctx context.Context) ([]model.QQGroupApplication, error) {
	rows := []model.QQGroupApplication{}
	e := r.db.WithContext(ctx).Order("created_at desc").Limit(200).Find(&rows).Error
	return rows, e
}

func (r *QQGroupBusinessRepository) Member(tx *gorm.DB, group, qq int64) (model.QQGroupMember, error) {
	row := model.QQGroupMember{GroupID: group, QQ: qq}
	if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; e != nil {
		return row, e
	}
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "group_id = ? AND qq = ?", group, qq).Error
	return row, e
}
func (r *QQGroupBusinessRepository) SaveMember(tx *gorm.DB, row *model.QQGroupMember) error {
	return tx.Save(row).Error
}
func (r *QQGroupBusinessRepository) AppliedEvent(tx *gorm.DB, id string) (bool, error) {
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.QQGroupAppliedEvent{ID: id})
	return result.RowsAffected == 1, result.Error
}
func (r *QQGroupBusinessRepository) CancelOperation(tx *gorm.DB, id string) error {
	if id == "" {
		return nil
	}
	return tx.Model(&model.QQBotOperation{}).Where("id = ? AND status IN ?", id, []string{"queued", "leased"}).Update("cancel_requested", true).Error
}
func (r *QQGroupBusinessRepository) CorporationName(tx *gorm.DB, id int64) (string, error) {
	var row model.EveEntityNameCache
	e := tx.First(&row, "entity_id = ? AND entity_type = ?", id, model.EntityNameTypeCorporation).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return row.Name, e
}
func (r *QQGroupBusinessRepository) CardMember(tx *gorm.DB, id string) (model.QQGroupMember, error) {
	var row model.QQGroupMember
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "card_operation_id = ?", id).Error
	return row, e
}
func (r *QQGroupBusinessRepository) CancelCards(tx *gorm.DB, group int64) error {
	ids := tx.Model(&model.QQGroupMember{}).Select("card_operation_id").Where("group_id = ?", group)
	return tx.Model(&model.QQBotOperation{}).Where("id IN (?) AND status IN ?", ids, []string{"queued", "leased"}).Update("cancel_requested", true).Error
}
func (r *QQGroupBusinessRepository) Members(ctx context.Context) ([]model.QQGroupMember, error) {
	rows := []model.QQGroupMember{}
	e := r.db.WithContext(ctx).Order("updated_at desc").Limit(200).Find(&rows).Error
	return rows, e
}

func (r *QQGroupBusinessRepository) CancelKicks(tx *gorm.DB, group int64) error {
	ids := tx.Model(&model.QQGroupMember{}).Select("kick_operation_id").Where("group_id = ?", group)
	return tx.Model(&model.QQBotOperation{}).Where("id IN (?) AND status IN ?", ids, []string{"queued", "leased"}).Update("cancel_requested", true).Error
}
func (r *QQGroupBusinessRepository) ResetConfirmations(tx *gorm.DB, group int64) error {
	return tx.Model(&model.QQGroupMember{}).Where("group_id = ?", group).Updates(map[string]interface{}{"confirmations": 0, "mismatch_since": nil, "chain_id": "", "last_mismatch_run_id": ""}).Error
}
func (r *QQGroupBusinessRepository) CreateInspection(tx *gorm.DB, run *model.QQGroupInspection) error {
	return tx.Create(run).Error
}
func (r *QQGroupBusinessRepository) Inspection(tx *gorm.DB, id string) (model.QQGroupInspection, error) {
	var row model.QQGroupInspection
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error
	return row, e
}
func (r *QQGroupBusinessRepository) InspectionForOperation(tx *gorm.DB, id string) (model.QQGroupInspection, error) {
	var row model.QQGroupInspection
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "operation_id = ?", id).Error
	return row, e
}
func (r *QQGroupBusinessRepository) SaveInspection(tx *gorm.DB, row *model.QQGroupInspection) error {
	return tx.Save(row).Error
}
func (r *QQGroupBusinessRepository) OpenInspections(tx *gorm.DB, group int64) (int64, error) {
	var n int64
	e := tx.Model(&model.QQGroupInspection{}).Where("group_id = ? AND status IN ?", group, []string{"snapshot_pending", "judging"}).Count(&n).Error
	return n, e
}
func (r *QQGroupBusinessRepository) GroupMembers(tx *gorm.DB, group int64) ([]model.QQGroupMember, error) {
	rows := []model.QQGroupMember{}
	e := tx.Where("group_id = ?", group).Find(&rows).Error
	return rows, e
}
func (r *QQGroupBusinessRepository) FreezeItems(tx *gorm.DB, rows []model.QQGroupInspectionItem) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(rows, 100).Error
}
func (r *QQGroupBusinessRepository) PendingItems(ctx context.Context) ([]model.QQGroupInspectionItem, error) {
	rows := []model.QQGroupInspectionItem{}
	e := r.db.WithContext(ctx).Where("done = ?", false).Order("run_id,qq").Limit(100).Find(&rows).Error
	return rows, e
}
func (r *QQGroupBusinessRepository) InspectionItem(tx *gorm.DB, run string, qq int64) (model.QQGroupInspectionItem, error) {
	var row model.QQGroupInspectionItem
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "run_id = ? AND qq = ?", run, qq).Error
	return row, e
}
func (r *QQGroupBusinessRepository) SaveItem(tx *gorm.DB, row *model.QQGroupInspectionItem) error {
	return tx.Save(row).Error
}
func (r *QQGroupBusinessRepository) RemainingItems(tx *gorm.DB, run string) (int64, error) {
	var n int64
	e := tx.Model(&model.QQGroupInspectionItem{}).Where("run_id = ? AND done = ?", run, false).Count(&n).Error
	return n, e
}
func (r *QQGroupBusinessRepository) KickMember(tx *gorm.DB, id string) (model.QQGroupMember, error) {
	var m model.QQGroupMember
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&m, "kick_operation_id = ?", id).Error
	return m, e
}
func (r *QQGroupBusinessRepository) Inspections(ctx context.Context) ([]model.QQGroupInspection, error) {
	rows := []model.QQGroupInspection{}
	e := r.db.WithContext(ctx).Order("created_at desc").Limit(100).Find(&rows).Error
	return rows, e
}
func (r *QQGroupBusinessRepository) Judgements(ctx context.Context) ([]model.QQGroupInspectionItem, error) {
	rows := []model.QQGroupInspectionItem{}
	e := r.db.WithContext(ctx).Order("updated_at desc").Limit(200).Find(&rows).Error
	return rows, e
}

func (r *QQGroupBusinessRepository) CancelSnapshots(tx *gorm.DB, group int64) error {
	ids := tx.Model(&model.QQGroupInspection{}).Select("operation_id").Where("group_id = ? AND status = ?", group, "snapshot_pending")
	return tx.Model(&model.QQBotOperation{}).Where("id IN (?) AND status IN ?", ids, []string{"queued", "leased"}).Update("cancel_requested", true).Error
}

func (r *QQGroupBusinessRepository) InspectionEvidence(tx *gorm.DB, id string) (model.QQGroupInspection, error) {
	var row model.QQGroupInspection
	e := tx.First(&row, "id = ?", id).Error
	return row, e
}
func (r *QQGroupBusinessRepository) JudgementEvidence(tx *gorm.DB, run string, qq int64) (model.QQGroupInspectionItem, error) {
	var row model.QQGroupInspectionItem
	e := tx.First(&row, "run_id = ? AND qq = ?", run, qq).Error
	return row, e
}
