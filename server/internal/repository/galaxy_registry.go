package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GalaxyRegistryRepository struct{}

func NewGalaxyRegistryRepository() *GalaxyRegistryRepository {
	return &GalaxyRegistryRepository{}
}

type GalaxyRegistrySystemListFilter struct {
	OnlyEnabled bool
}

type GalaxyRegistryEntryListFilter struct {
	SystemConfigID   *uint
	CaptainUserID    *uint
	Keyword          string
	Status           string
	ValidationStatus string
	StartDateFrom    *time.Time
	StartDateTo      *time.Time
	EndDateFrom      *time.Time
	EndDateTo        *time.Time
}

type GalaxyRegistryActiveEntryView struct {
	ID                   uint      `json:"id"`
	SystemConfigID       uint      `json:"system_config_id"`
	SolarSystemID        int64     `json:"solar_system_id"`
	CaptainUserID        uint      `json:"captain_user_id"`
	CaptainCharacterID   int64     `json:"captain_character_id"`
	CaptainCharacterName string    `json:"captain_character_name"`
	ExpectedEndAt        time.Time `json:"expected_end_at"`
	ActualStartAt        time.Time `json:"actual_start_at"`
}

type GalaxyRegistrySdeSystem struct {
	SolarSystemID     int64   `json:"solar_system_id"`
	SolarSystemName   string  `json:"solar_system_name"`
	RegionID          int64   `json:"region_id"`
	RegionName        string  `json:"region_name"`
	ConstellationID   int64   `json:"constellation_id"`
	ConstellationName string  `json:"constellation_name"`
	Security          float64 `json:"security"`
}

type GalaxyRegistryValidationCandidate struct {
	ID                    uint
	CaptainUserID         uint
	SolarSystemID         int64
	FrozenMinBountyAmount float64
	ActualStartAt         time.Time
	ActualEndAt           time.Time
}

type GalaxyRegistryTopSystemStat struct {
	SystemConfigID  uint   `json:"system_config_id"`
	SolarSystemID   int64  `json:"solar_system_id"`
	SolarSystemName string `json:"solar_system_name"`
	RegisterCount   int64  `json:"register_count"`
}

func (r *GalaxyRegistryRepository) CreateSystem(row *model.GalaxyRegistrySystem) error {
	return global.DB.Create(row).Error
}

func (r *GalaxyRegistryRepository) UpdateSystem(id uint, updates map[string]any) error {
	result := global.DB.Model(&model.GalaxyRegistrySystem{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *GalaxyRegistryRepository) DeleteSystem(id uint) error {
	result := global.DB.Delete(&model.GalaxyRegistrySystem{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *GalaxyRegistryRepository) GetSystemByID(id uint) (*model.GalaxyRegistrySystem, error) {
	var row model.GalaxyRegistrySystem
	if err := global.DB.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GalaxyRegistryRepository) GetSystemBySolarSystemID(solarSystemID int64) (*model.GalaxyRegistrySystem, error) {
	var row model.GalaxyRegistrySystem
	if err := global.DB.Where("solar_system_id = ?", solarSystemID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GalaxyRegistryRepository) ListSystems(filter GalaxyRegistrySystemListFilter) ([]model.GalaxyRegistrySystem, error) {
	rows := make([]model.GalaxyRegistrySystem, 0)
	query := global.DB.Model(&model.GalaxyRegistrySystem{})
	if filter.OnlyEnabled {
		query = query.Where("is_enabled = ?", true)
	}
	err := query.Order("region_name ASC, constellation_name ASC, solar_system_name ASC").Find(&rows).Error
	return rows, err
}

func (r *GalaxyRegistryRepository) CreateEntry(row *model.GalaxyRegistryEntry) error {
	return global.DB.Create(row).Error
}

func (r *GalaxyRegistryRepository) UpdateEntry(id uint, updates map[string]any) error {
	result := global.DB.Model(&model.GalaxyRegistryEntry{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *GalaxyRegistryRepository) UpdateEntryValidationResult(
	id uint,
	validationStatus string,
	validatedAmount float64,
	validatedCount int,
	validatedAt time.Time,
	violationReason string,
) error {
	updates := map[string]any{
		"validation_status":       validationStatus,
		"validated_bounty_amount": validatedAmount,
		"validated_bounty_count":  validatedCount,
		"validated_at":            validatedAt,
		"violation_reason":        violationReason,
	}
	return r.UpdateEntry(id, updates)
}

func (r *GalaxyRegistryRepository) ResetEntryValidationPending(id uint) error {
	updates := map[string]any{
		"validation_status":       model.GalaxyRegistryValidationPending,
		"validated_bounty_amount": 0,
		"validated_bounty_count":  0,
		"validated_at":            nil,
		"violation_reason":        "",
	}
	return r.UpdateEntry(id, updates)
}

func (r *GalaxyRegistryRepository) GetEntryByID(id uint) (*model.GalaxyRegistryEntry, error) {
	var row model.GalaxyRegistryEntry
	if err := global.DB.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GalaxyRegistryRepository) GetEntryByIDForUpdateTx(tx *gorm.DB, id uint) (*model.GalaxyRegistryEntry, error) {
	var row model.GalaxyRegistryEntry
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GalaxyRegistryRepository) GetSystemByIDForUpdateTx(tx *gorm.DB, id uint) (*model.GalaxyRegistrySystem, error) {
	var row model.GalaxyRegistrySystem
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GalaxyRegistryRepository) FindActiveEntryBySystemConfigID(systemConfigID uint) (*model.GalaxyRegistryEntry, error) {
	var row model.GalaxyRegistryEntry
	err := global.DB.Where("system_config_id = ? AND status = ?", systemConfigID, model.GalaxyRegistryEntryStatusActive).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GalaxyRegistryRepository) FindActiveEntryBySystemConfigIDForUpdateTx(tx *gorm.DB, systemConfigID uint) (*model.GalaxyRegistryEntry, error) {
	var row model.GalaxyRegistryEntry
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("system_config_id = ? AND status = ?", systemConfigID, model.GalaxyRegistryEntryStatusActive).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GalaxyRegistryRepository) HasActiveEntryBySystemConfigID(systemConfigID uint) (bool, error) {
	var count int64
	err := global.DB.Model(&model.GalaxyRegistryEntry{}).
		Where("system_config_id = ? AND status = ?", systemConfigID, model.GalaxyRegistryEntryStatusActive).
		Count(&count).Error
	return count > 0, err
}

func (r *GalaxyRegistryRepository) ListActiveEntriesBySystemConfigIDs(systemConfigIDs []uint) ([]GalaxyRegistryActiveEntryView, error) {
	rows := make([]GalaxyRegistryActiveEntryView, 0)
	if len(systemConfigIDs) == 0 {
		return rows, nil
	}
	err := global.DB.Model(&model.GalaxyRegistryEntry{}).
		Select("id, system_config_id, solar_system_id, captain_user_id, captain_character_id, captain_character_name, expected_end_at, actual_start_at").
		Where("system_config_id IN ? AND status = ?", systemConfigIDs, model.GalaxyRegistryEntryStatusActive).
		Order("actual_start_at ASC").
		Find(&rows).Error
	return rows, err
}

func (r *GalaxyRegistryRepository) ListActiveEntriesStartedBefore(before time.Time, limit int) ([]model.GalaxyRegistryEntry, error) {
	rows := make([]model.GalaxyRegistryEntry, 0)
	query := global.DB.Model(&model.GalaxyRegistryEntry{}).
		Where("status = ? AND actual_start_at <= ?", model.GalaxyRegistryEntryStatusActive, before).
		Order("actual_start_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&rows).Error
	return rows, err
}

func buildGalaxyRegistryEntryQuery(db *gorm.DB, filter GalaxyRegistryEntryListFilter) *gorm.DB {
	query := db.Model(&model.GalaxyRegistryEntry{})
	if filter.SystemConfigID != nil && *filter.SystemConfigID > 0 {
		query = query.Where("system_config_id = ?", *filter.SystemConfigID)
	}
	if filter.CaptainUserID != nil && *filter.CaptainUserID > 0 {
		query = query.Where("captain_user_id = ?", *filter.CaptainUserID)
	}
	if strings.TrimSpace(filter.Keyword) != "" {
		keyword := "%" + strings.ToLower(strings.TrimSpace(filter.Keyword)) + "%"
		query = query.Where(
			"LOWER(solar_system_name) LIKE ? OR LOWER(captain_character_name) LIKE ?",
			keyword,
			keyword,
		)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ValidationStatus != "" {
		query = query.Where("validation_status = ?", filter.ValidationStatus)
	}
	if filter.StartDateFrom != nil {
		query = query.Where("actual_start_at >= ?", *filter.StartDateFrom)
	}
	if filter.StartDateTo != nil {
		query = query.Where("actual_start_at <= ?", *filter.StartDateTo)
	}
	if filter.EndDateFrom != nil {
		query = query.Where("actual_end_at IS NOT NULL AND actual_end_at >= ?", *filter.EndDateFrom)
	}
	if filter.EndDateTo != nil {
		query = query.Where("actual_end_at IS NOT NULL AND actual_end_at <= ?", *filter.EndDateTo)
	}
	return query
}

func (r *GalaxyRegistryRepository) ListEntries(
	filter GalaxyRegistryEntryListFilter,
	page int,
	pageSize int,
) ([]model.GalaxyRegistryEntry, int64, error) {
	rows := make([]model.GalaxyRegistryEntry, 0)
	query := buildGalaxyRegistryEntryQuery(global.DB, filter)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("actual_start_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error
	return rows, total, err
}

func (r *GalaxyRegistryRepository) ListPendingValidationCandidates(limit int) ([]GalaxyRegistryValidationCandidate, error) {
	rows := make([]GalaxyRegistryValidationCandidate, 0)
	query := global.DB.Model(&model.GalaxyRegistryEntry{}).
		Select("id, captain_user_id, solar_system_id, frozen_min_bounty_amount, actual_start_at, actual_end_at").
		Where("status = ? AND validation_status = ? AND actual_end_at IS NOT NULL",
			model.GalaxyRegistryEntryStatusCompleted,
			model.GalaxyRegistryValidationPending,
		).
		Order("actual_end_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&rows).Error
	return rows, err
}

func (r *GalaxyRegistryRepository) CountSystemsByEnabled(enabled bool) (int64, error) {
	var count int64
	err := global.DB.Model(&model.GalaxyRegistrySystem{}).Where("is_enabled = ?", enabled).Count(&count).Error
	return count, err
}

func (r *GalaxyRegistryRepository) CountCurrentBusySystems() (int64, error) {
	var count int64
	err := global.DB.Model(&model.GalaxyRegistryEntry{}).
		Distinct("system_config_id").
		Where("status = ?", model.GalaxyRegistryEntryStatusActive).
		Count(&count).Error
	return count, err
}

func (r *GalaxyRegistryRepository) CountCurrentOverdueEntries(now time.Time) (int64, error) {
	var count int64
	err := global.DB.Model(&model.GalaxyRegistryEntry{}).
		Where("status = ? AND expected_end_at < ?", model.GalaxyRegistryEntryStatusActive, now).
		Count(&count).Error
	return count, err
}

func (r *GalaxyRegistryRepository) CountEntriesCompletedBetween(start time.Time, end time.Time) (int64, error) {
	var count int64
	err := global.DB.Model(&model.GalaxyRegistryEntry{}).
		Where("actual_start_at >= ? AND actual_start_at < ?", start, end).
		Count(&count).Error
	return count, err
}

func (r *GalaxyRegistryRepository) CountEntriesByValidationBetween(status string, start time.Time, end time.Time) (int64, error) {
	var count int64
	err := global.DB.Model(&model.GalaxyRegistryEntry{}).
		Where("actual_end_at IS NOT NULL AND actual_end_at >= ? AND actual_end_at < ? AND validation_status = ?", start, end, status).
		Count(&count).Error
	return count, err
}

func (r *GalaxyRegistryRepository) ListTopSystemsByRegistrations(start time.Time, end time.Time, limit int) ([]GalaxyRegistryTopSystemStat, error) {
	rows := make([]GalaxyRegistryTopSystemStat, 0)
	query := global.DB.Model(&model.GalaxyRegistryEntry{}).
		Select("system_config_id, solar_system_id, solar_system_name, COUNT(*) AS register_count").
		Where("actual_start_at >= ? AND actual_start_at < ?", start, end).
		Group("system_config_id, solar_system_id, solar_system_name").
		Order("register_count DESC, solar_system_name ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Scan(&rows).Error
	return rows, err
}

func (r *GalaxyRegistryRepository) ListRecentViolations(
	start time.Time,
	end time.Time,
	limit int,
) ([]model.GalaxyRegistryEntry, error) {
	rows := make([]model.GalaxyRegistryEntry, 0)
	query := global.DB.Model(&model.GalaxyRegistryEntry{}).
		Where("validation_status = ? AND actual_end_at IS NOT NULL AND actual_end_at >= ? AND actual_end_at < ?",
			model.GalaxyRegistryValidationViolation,
			start,
			end,
		).
		Order("actual_end_at DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&rows).Error
	return rows, err
}

func (r *GalaxyRegistryRepository) SumBountyJournalInWindow(
	characterIDs []int64,
	solarSystemID int64,
	startAt time.Time,
	endAt time.Time,
) (float64, int64, error) {
	if len(characterIDs) == 0 {
		return 0, 0, nil
	}
	type result struct {
		Amount float64
		Count  int64
	}
	var row result
	err := global.DB.Model(&model.EVECharacterWalletJournal{}).
		Select("COALESCE(SUM(amount), 0) AS amount, COUNT(*) AS count").
		Where("character_id IN ?", characterIDs).
		Where("context_id = ?", solarSystemID).
		Where("ref_type = ?", "bounty_prizes").
		Where("date >= ? AND date <= ?", startAt, endAt).
		Where("amount > 0").
		Scan(&row).Error
	return row.Amount, row.Count, err
}
