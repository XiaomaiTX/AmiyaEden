package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"errors"
	"strings"

	"gorm.io/gorm"
)

const blacklistWriteBatchSize = 500

type BlacklistRepository struct{}

func NewBlacklistRepository() *BlacklistRepository {
	return &BlacklistRepository{}
}

type BlacklistEntryFilter struct {
	Kind    string
	Keyword string
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// A digits-only keyword is an exact QQ number or character ID; anything else is
// a case-insensitive substring of a character or Discord name. QQ numbers are
// never substring-matched, so a prefix of a number cannot produce a false hit.
func applyBlacklistEntryFilter(db *gorm.DB, filter BlacklistEntryFilter) *gorm.DB {
	if filter.Kind != "" {
		db = db.Where("kind = ?", filter.Kind)
	}
	keyword := strings.TrimSpace(filter.Keyword)
	if keyword == "" {
		return db
	}
	if isASCIIDigits(keyword) {
		return db.Where("value = ? AND kind IN ?", keyword, []string{model.BlacklistKindCharacter, model.BlacklistKindQQ})
	}
	pattern := "%" + likeEscaper.Replace(strings.ToLower(keyword)) + "%"
	return db.Where("kind IN ? AND LOWER(display) LIKE ? ESCAPE '\\'", []string{model.BlacklistKindCharacter, model.BlacklistKindDiscord}, pattern)
}

func isASCIIDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

func (r *BlacklistRepository) List(page, size int, filter BlacklistEntryFilter) ([]model.BlacklistEntry, int64, error) {
	var total int64
	if err := applyBlacklistEntryFilter(global.DB.Model(&model.BlacklistEntry{}), filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]model.BlacklistEntry, 0, size)
	err := applyBlacklistEntryFilter(global.DB.Model(&model.BlacklistEntry{}), filter).
		Order("kind ASC, LOWER(display) ASC, id ASC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&rows).Error
	return rows, total, err
}

func (r *BlacklistRepository) ListAllEntriesTx(tx *gorm.DB) ([]model.BlacklistEntry, error) {
	var rows []model.BlacklistEntry
	err := tx.Order("id ASC").Find(&rows).Error
	return rows, err
}

// FindEntriesTx returns the entries equal to any of the given character IDs, QQ
// numbers or Discord keys, each set matched only against its own kind. Callers
// pass values already normalized like stored entries.
func (r *BlacklistRepository) FindEntriesTx(tx *gorm.DB, characters, qqs, discords []string) ([]model.BlacklistEntry, error) {
	rows := make([]model.BlacklistEntry, 0)
	var conditions []string
	var args []any
	for _, set := range []struct {
		kind   string
		values []string
	}{
		{model.BlacklistKindCharacter, characters},
		{model.BlacklistKindQQ, qqs},
		{model.BlacklistKindDiscord, discords},
	} {
		if len(set.values) > 0 {
			conditions = append(conditions, "(kind = ? AND value IN ?)")
			args = append(args, set.kind, set.values)
		}
	}
	if len(conditions) == 0 {
		return rows, nil
	}
	err := tx.Where(strings.Join(conditions, " OR "), args...).Order("kind ASC, LOWER(display) ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *BlacklistRepository) ListUnresolvedNamesTx(tx *gorm.DB) ([]model.BlacklistUnresolvedName, error) {
	var rows []model.BlacklistUnresolvedName
	err := tx.Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *BlacklistRepository) CreateEntriesTx(tx *gorm.DB, rows []model.BlacklistEntry) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.CreateInBatches(rows, blacklistWriteBatchSize).Error
}

func (r *BlacklistRepository) UpdateEntryTx(tx *gorm.DB, id uint, display, reason string) error {
	return tx.Model(&model.BlacklistEntry{}).Where("id = ?", id).
		Updates(map[string]any{"display": display, "reason": reason}).Error
}

func (r *BlacklistRepository) DeleteEntriesTx(tx *gorm.DB, ids []uint) error {
	return deleteByIDsInBatches(tx, &model.BlacklistEntry{}, ids)
}

func (r *BlacklistRepository) CreateUnresolvedNamesTx(tx *gorm.DB, rows []model.BlacklistUnresolvedName) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.CreateInBatches(rows, blacklistWriteBatchSize).Error
}

func (r *BlacklistRepository) DeleteUnresolvedNamesTx(tx *gorm.DB, ids []uint) error {
	return deleteByIDsInBatches(tx, &model.BlacklistUnresolvedName{}, ids)
}

func deleteByIDsInBatches(tx *gorm.DB, value any, ids []uint) error {
	for start := 0; start < len(ids); start += blacklistWriteBatchSize {
		end := min(start+blacklistWriteBatchSize, len(ids))
		if err := tx.Where("id IN ?", ids[start:end]).Delete(value).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *BlacklistRepository) CreateRefreshRunTx(tx *gorm.DB, run *model.BlacklistRefreshRun) error {
	return tx.Create(run).Error
}

// LatestRefreshRun returns nil when no refresh has ever succeeded.
func (r *BlacklistRepository) LatestRefreshRun() (*model.BlacklistRefreshRun, error) {
	var run model.BlacklistRefreshRun
	err := global.DB.Order("finished_at DESC, id DESC").First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}
