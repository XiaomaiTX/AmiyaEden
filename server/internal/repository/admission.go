package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrAdmissionVersionConflict = errors.New("admission version conflict")

type AdmissionRepository struct{}

func (r *AdmissionRepository) ApplicantTx(tx *gorm.DB, id uint) (*model.User, error) {
	var user model.User
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, id).Error
	return &user, err
}

func (r *AdmissionRepository) BoundCharactersTx(tx *gorm.DB, userID uint) ([]model.EveCharacter, error) {
	var chars []model.EveCharacter
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).Order("character_id").Find(&chars).Error
	return chars, err
}

// AdmissionUserContact is an account's current contact details.
type AdmissionUserContact struct {
	ID        uint
	QQ        string
	DiscordID string
}

// BoundCharacterIDsByUsersTx reads, without locking, the characters currently
// bound to each of the given accounts.
func (r *AdmissionRepository) BoundCharacterIDsByUsersTx(tx *gorm.DB, userIDs []uint) (map[uint][]int64, error) {
	bound := make(map[uint][]int64, len(userIDs))
	if len(userIDs) == 0 {
		return bound, nil
	}
	var rows []struct {
		UserID      uint
		CharacterID int64
	}
	err := tx.Model(&model.EveCharacter{}).Select("user_id", "character_id").Where("user_id IN ?", userIDs).Order("user_id, character_id").Find(&rows).Error
	for _, row := range rows {
		bound[row.UserID] = append(bound[row.UserID], row.CharacterID)
	}
	return bound, err
}

// UserContactsTx reads, without locking, the current contacts of the given
// accounts; deleted accounts are absent.
func (r *AdmissionRepository) UserContactsTx(tx *gorm.DB, userIDs []uint) ([]AdmissionUserContact, error) {
	contacts := make([]AdmissionUserContact, 0, len(userIDs))
	if len(userIDs) == 0 {
		return contacts, nil
	}
	err := tx.Model(&model.User{}).Select("id", "qq", "discord_id").Where("id IN ?", userIDs).Order("id").Find(&contacts).Error
	return contacts, err
}

func (r *AdmissionRepository) SalaryTransactionTx(tx *gorm.DB, userID uint, refID string) (*model.WalletTransaction, error) {
	var transaction model.WalletTransaction
	err := tx.Where("user_id = ? AND ref_type = ? AND ref_id = ?", userID, model.WalletRefInvitationSalary, refID).First(&transaction).Error
	return &transaction, err
}

func NewAdmissionRepository() *AdmissionRepository { return &AdmissionRepository{} }

func (r *AdmissionRepository) CountPendingInCorporations(db *gorm.DB, corporations []int64, excludeUserID uint) (int64, error) {
	if len(corporations) == 0 {
		return 0, nil
	}
	var count int64
	err := db.Model(&model.AdmissionApplication{}).
		Where("target_corporation_id IN ? AND status = ? AND user_id <> ?", corporations, model.AdmissionStatusPending, excludeUserID).Count(&count).Error
	return count, err
}

func (r *AdmissionRepository) UserCorporationsWithRoles(db *gorm.DB, userID uint, corporations []int64, roles []string) ([]int64, error) {
	ids := make([]int64, 0)
	err := db.Model(&model.EveCharacterCorpRole{}).
		Joins("JOIN eve_character c ON c.character_id = eve_character_corp_role.character_id AND c.deleted_at IS NULL").
		Where("c.user_id = ? AND c.corporation_id = eve_character_corp_role.corporation_id", userID).
		Where("c.corporation_id IN ? AND corp_role IN ?", corporations, roles).
		Distinct("eve_character_corp_role.corporation_id").Order("eve_character_corp_role.corporation_id").Pluck("eve_character_corp_role.corporation_id", &ids).Error
	return ids, err
}

type AdmissionListFilter struct {
	UserID              uint
	CorporationIDs      []int64
	TargetCorporationID int64
	Statuses            []string
	Kind                string
}

func (r *AdmissionRepository) InTx(fn func(*gorm.DB) error) error {
	return global.DB.Transaction(fn)
}

func (r *AdmissionRepository) CreateTx(tx *gorm.DB, application *model.AdmissionApplication) error {
	return tx.Omit(clause.Associations).Create(application).Error
}

func (r *AdmissionRepository) GetByID(db *gorm.DB, id uint, lock bool) (*model.AdmissionApplication, error) {
	var application model.AdmissionApplication
	query := db
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.First(&application, id).Error; err != nil {
		return nil, err
	}
	return &application, nil
}

// Hold the authorization target stable until all private data has been read.
func (r *AdmissionRepository) ReadByIDTx(tx *gorm.DB, id uint) (*model.AdmissionApplication, error) {
	return r.GetByID(tx.Clauses(clause.Locking{Strength: "SHARE"}), id, false)
}

func (r *AdmissionRepository) UpdateTx(tx *gorm.DB, application *model.AdmissionApplication, expectedVersion uint) error {
	result := tx.Model(&model.AdmissionApplication{}).
		Where("id = ? AND version = ?", application.ID, expectedVersion).
		Select("*").Omit("id", "created_at", "deleted_at", clause.Associations).Updates(application)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrAdmissionVersionConflict
	}
	return nil
}

// List applies the owner or reviewer scope before counting and paginating. An
// empty reviewer scope deliberately returns no rows, including for counts.
func (r *AdmissionRepository) List(db *gorm.DB, filter AdmissionListFilter, page, pageSize int) ([]model.AdmissionApplication, int64, error) {
	query := db.Model(&model.AdmissionApplication{})
	if filter.UserID != 0 {
		query = query.Where("user_id = ?", filter.UserID)
	} else {
		query = query.Where("target_corporation_id IN ?", filter.CorporationIDs)
	}
	if filter.TargetCorporationID != 0 {
		query = query.Where("target_corporation_id = ?", filter.TargetCorporationID)
	}
	if len(filter.Statuses) != 0 {
		query = query.Where("status IN ?", filter.Statuses)
	}
	if filter.Kind != "" {
		query = query.Where("kind = ?", filter.Kind)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	applications := make([]model.AdmissionApplication, 0)
	// Keep the target stable while the separate character preload runs.
	err := query.Clauses(clause.Locking{Strength: "SHARE"}).Preload("Characters", func(db *gorm.DB) *gorm.DB { return db.Order("character_id") }).
		Order("submitted_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&applications).Error
	return applications, total, err
}

func (r *AdmissionRepository) Characters(db *gorm.DB, applicationID uint) ([]model.AdmissionCharacter, error) {
	characters := make([]model.AdmissionCharacter, 0)
	err := db.Where("application_id = ?", applicationID).Order("character_id").Find(&characters).Error
	return characters, err
}

func (r *AdmissionRepository) ReplaceCharactersTx(tx *gorm.DB, applicationID uint, characters []model.AdmissionCharacter) error {
	if err := tx.Where("application_id = ?", applicationID).Delete(&model.AdmissionCharacter{}).Error; err != nil {
		return err
	}
	return tx.Create(&characters).Error
}

func (r *AdmissionRepository) ReleaseClaimsTx(tx *gorm.DB, applicationID uint) error {
	return tx.Where("application_id = ?", applicationID).Delete(&model.AdmissionCharacterClaim{}).Error
}

func (r *AdmissionRepository) ClaimCharactersTx(tx *gorm.DB, applicationID uint, characters []model.AdmissionCharacter) error {
	claims := make([]model.AdmissionCharacterClaim, 0, len(characters))
	for _, character := range characters {
		claims = append(claims, model.AdmissionCharacterClaim{CharacterID: character.CharacterID, ApplicationID: applicationID})
	}
	return tx.Create(&claims).Error
}

func (r *AdmissionRepository) EvidenceMetadata(db *gorm.DB, applicationID uint) ([]model.AdmissionEvidence, error) {
	evidence := make([]model.AdmissionEvidence, 0)
	err := db.Select("application_id", "kind", "width", "height", "byte_size", "updated_at").
		Where("application_id = ?", applicationID).Order("kind").Find(&evidence).Error
	return evidence, err
}

func (r *AdmissionRepository) Evidence(db *gorm.DB, applicationID uint, kind string) (*model.AdmissionEvidence, error) {
	var evidence model.AdmissionEvidence
	err := db.Where("application_id = ? AND kind = ?", applicationID, kind).First(&evidence).Error
	return &evidence, err
}

func (r *AdmissionRepository) PutEvidenceTx(tx *gorm.DB, evidence *model.AdmissionEvidence) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "application_id"}, {Name: "kind"}},
		DoUpdates: clause.AssignmentColumns([]string{"data", "width", "height", "byte_size", "updated_at"}),
	}).Create(evidence).Error
}

func (r *AdmissionRepository) DeleteEvidenceTx(tx *gorm.DB, applicationID uint) error {
	return tx.Where("application_id = ?", applicationID).Delete(&model.AdmissionEvidence{}).Error
}

func (r *AdmissionRepository) AddEventTx(tx *gorm.DB, event *model.AdmissionProcessingEvent) error {
	return tx.Create(event).Error
}

func (r *AdmissionRepository) Events(db *gorm.DB, applicationID uint) ([]model.AdmissionProcessingEvent, error) {
	events := make([]model.AdmissionProcessingEvent, 0)
	err := db.Where("application_id = ?", applicationID).Order("id").Find(&events).Error
	return events, err
}

func (r *AdmissionRepository) AddSalaryTx(tx *gorm.DB, salary *model.AdmissionSalary) error {
	return tx.Create(salary).Error
}

func (r *AdmissionRepository) Salary(db *gorm.DB, applicationID uint) (*model.AdmissionSalary, error) {
	var salary model.AdmissionSalary
	err := db.Where("application_id = ?", applicationID).First(&salary).Error
	return &salary, err
}

// ExpiredWithEvidenceTx shares the application row lock with resubmission;
// subsequent cleanup can never delete evidence on a newly pending revision.
func (r *AdmissionRepository) ExpiredWithEvidenceTx(tx *gorm.DB, now time.Time, limit int) ([]model.AdmissionApplication, error) {
	applications := make([]model.AdmissionApplication, 0)
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status IN ? AND evidence_expires_at <= ?", []string{model.AdmissionStatusInvited, model.AdmissionStatusRejected}, now).
		Where("EXISTS (SELECT 1 FROM admission_evidence e WHERE e.application_id = admission_application.id)").
		Order("id").Limit(limit).Find(&applications).Error
	return applications, err
}
