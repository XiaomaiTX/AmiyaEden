package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NewbroRecruitmentRepository struct{}

func NewNewbroRecruitmentRepository() *NewbroRecruitmentRepository {
	return &NewbroRecruitmentRepository{}
}

// ── Recruitment (links) ───────────────────────────────────────────────

// Create inserts a new recruitment record and populates r.ID.
func (r *NewbroRecruitmentRepository) Create(rec *model.NewbroRecruitment) error {
	return r.CreateTx(global.DB, rec)
}

// CreateTx inserts a new recruitment record inside the provided transaction.
func (r *NewbroRecruitmentRepository) CreateTx(tx *gorm.DB, rec *model.NewbroRecruitment) error {
	return tx.Create(rec).Error
}

// UpdateCode sets the code field by ID (used after insert to store base62 of ID).
func (r *NewbroRecruitmentRepository) UpdateCode(id uint, code string) error {
	return r.UpdateCodeTx(global.DB, id, code)
}

// UpdateCodeTx sets the code field by ID inside the provided transaction.
func (r *NewbroRecruitmentRepository) UpdateCodeTx(tx *gorm.DB, id uint, code string) error {
	return tx.Model(&model.NewbroRecruitment{}).Where("id = ?", id).Update("code", code).Error
}

// GetByCode fetches a recruitment record by its short code. Returns nil, nil if not found.
func (r *NewbroRecruitmentRepository) GetByCode(code string) (*model.NewbroRecruitment, error) {
	var rec model.NewbroRecruitment
	err := global.DB.Where("code = ?", code).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rec, err
}

// GetLatestByUserID returns the most recent recruitment for the user, or nil if none.
func (r *NewbroRecruitmentRepository) GetLatestByUserID(userID uint) (*model.NewbroRecruitment, error) {
	return r.GetLatestByUserIDTx(global.DB, userID)
}

// GetLatestByUserIDTx returns the most recent recruitment for the user inside a transaction.
func (r *NewbroRecruitmentRepository) GetLatestByUserIDTx(tx *gorm.DB, userID uint) (*model.NewbroRecruitment, error) {
	var rec model.NewbroRecruitment
	err := tx.Where("user_id = ?", userID).Order("generated_at DESC").First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rec, err
}

// GetLatestGeneratedLinkByUserIDTx returns the latest user-generated recruit link, excluding direct-referral records.
func (r *NewbroRecruitmentRepository) GetLatestGeneratedLinkByUserIDTx(tx *gorm.DB, userID uint) (*model.NewbroRecruitment, error) {
	var rec model.NewbroRecruitment
	err := tx.Where(
		"user_id = ? AND (source = ? OR source = '' OR source IS NULL)",
		userID,
		model.RecruitmentSourceLink,
	).Order("generated_at DESC").First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rec, err
}

// GetByCodeForUpdateTx fetches a recruitment by code and locks the row for update.
func (r *NewbroRecruitmentRepository) GetByCodeForUpdateTx(tx *gorm.DB, code string) (*model.NewbroRecruitment, error) {
	var rec model.NewbroRecruitment
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", code).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rec, err
}

// ListByUserID returns all recruitment records for a user, newest first.
func (r *NewbroRecruitmentRepository) ListByUserID(userID uint) ([]model.NewbroRecruitment, error) {
	var recs []model.NewbroRecruitment
	err := global.DB.Where("user_id = ?", userID).Order("generated_at DESC").Find(&recs).Error
	return recs, err
}

// ListAllPaged returns paginated recruitment records across all users, newest first.
func (r *NewbroRecruitmentRepository) ListAllPaged(page, pageSize int) ([]model.NewbroRecruitment, int64, error) {
	var recs []model.NewbroRecruitment
	var total int64
	offset := (page - 1) * pageSize
	db := global.DB.Model(&model.NewbroRecruitment{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("generated_at DESC").Offset(offset).Limit(pageSize).Find(&recs).Error; err != nil {
		return nil, 0, err
	}
	return recs, total, nil
}

// ── Recruitment entries ───────────────────────────────────────────────

// CreateEntry inserts a new entry.
func (r *NewbroRecruitmentRepository) CreateEntry(entry *model.NewbroRecruitmentEntry) error {
	return r.CreateEntryTx(global.DB, entry)
}

// CreateEntryTx inserts a new entry inside the provided transaction.
func (r *NewbroRecruitmentRepository) CreateEntryTx(tx *gorm.DB, entry *model.NewbroRecruitmentEntry) error {
	return tx.Create(entry).Error
}

// GetEntryByRecruitmentIDAndQQ returns an existing entry for a recruitment and QQ pair.
func (r *NewbroRecruitmentRepository) GetEntryByRecruitmentIDAndQQ(recruitmentID uint, qq string) (*model.NewbroRecruitmentEntry, error) {
	return r.GetEntryByRecruitmentIDAndQQTx(global.DB, recruitmentID, qq)
}

// GetEntryByRecruitmentIDAndQQTx returns an existing entry for a recruitment and QQ pair inside a transaction.
func (r *NewbroRecruitmentRepository) GetEntryByRecruitmentIDAndQQTx(tx *gorm.DB, recruitmentID uint, qq string) (*model.NewbroRecruitmentEntry, error) {
	var entry model.NewbroRecruitmentEntry
	err := tx.Where("recruitment_id = ? AND qq = ?", recruitmentID, qq).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &entry, err
}

// ListEntriesByRecruitmentIDs returns all entries for a set of recruitment IDs.
func (r *NewbroRecruitmentRepository) ListEntriesByRecruitmentIDs(ids []uint) ([]model.NewbroRecruitmentEntry, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var entries []model.NewbroRecruitmentEntry
	err := global.DB.Where("recruitment_id IN ?", ids).Order("entered_at DESC").Find(&entries).Error
	return entries, err
}

// ListOngoingEntriesAfterID returns up to limit ongoing entries with IDs greater than afterID.
func (r *NewbroRecruitmentRepository) ListOngoingEntriesAfterID(afterID uint, limit int) ([]model.NewbroRecruitmentEntry, error) {
	var entries []model.NewbroRecruitmentEntry
	err := global.DB.Where("status = ? AND id > ?", model.RecruitEntryStatusOngoing, afterID).
		Order("id ASC").Limit(limit).Find(&entries).Error
	return entries, err
}

// HasEntryWithWalletRefID reports whether an entry has already claimed the reward ref ID.
func (r *NewbroRecruitmentRepository) HasEntryWithWalletRefID(walletRefID string) (bool, error) {
	return r.HasEntryWithWalletRefIDTx(global.DB, walletRefID)
}

// HasEntryWithWalletRefIDTx reports whether an entry has already claimed the reward ref ID inside a transaction.
func (r *NewbroRecruitmentRepository) HasEntryWithWalletRefIDTx(tx *gorm.DB, walletRefID string) (bool, error) {
	var count int64
	err := tx.Model(&model.NewbroRecruitmentEntry{}).Where("wallet_ref_id = ?", walletRefID).Count(&count).Error
	return count > 0, err
}

// MarkEntryValidTx updates a single entry to valid status and sets rewarded_at inside a transaction.
func (r *NewbroRecruitmentRepository) MarkEntryValidTx(tx *gorm.DB, entryID uint, matchedUserID uint, now interface{}, walletRefID string) error {
	return tx.Model(&model.NewbroRecruitmentEntry{}).Where("id = ?", entryID).Updates(map[string]interface{}{
		"status":          model.RecruitEntryStatusValid,
		"matched_user_id": matchedUserID,
		"rewarded_at":     now,
		"wallet_ref_id":   &walletRefID,
	}).Error
}

// MarkEntryStalled updates a single entry to stalled status.
func (r *NewbroRecruitmentRepository) MarkEntryStalled(entryID uint) error {
	return global.DB.Model(&model.NewbroRecruitmentEntry{}).Where("id = ?", entryID).
		Update("status", model.RecruitEntryStatusStalled).Error
}

// GetRecruitmentByID fetches a recruitment by primary key.
func (r *NewbroRecruitmentRepository) GetRecruitmentByID(id uint) (*model.NewbroRecruitment, error) {
	var rec model.NewbroRecruitment
	err := global.DB.Where("id = ?", id).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rec, err
}
