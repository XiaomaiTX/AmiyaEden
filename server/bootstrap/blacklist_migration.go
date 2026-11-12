package bootstrap

import (
	"amiya-eden/internal/model"

	"gorm.io/gorm"
)

func migrateBlacklist(db *gorm.DB) error {
	return db.AutoMigrate(&model.BlacklistEntry{}, &model.BlacklistUnresolvedName{}, &model.BlacklistRefreshRun{})
}
