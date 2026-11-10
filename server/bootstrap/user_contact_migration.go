package bootstrap

import (
	"amiya-eden/internal/model"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const userContactMigrationVersion = "20261110_require_user_contact"

type schemaMigration struct {
	Version   string `gorm:"primaryKey;size:128"`
	AppliedAt time.Time
}

func (schemaMigration) TableName() string { return "schema_migration" }

// Explicitly scoped to personal ESI data. Financial, business and audit history stays intact.
var contactCleanupCharacterTables = []string{
	"character_corporation_history", "eve_character_asset", "eve_character_clone_base_info",
	"eve_character_contract", "eve_character_contract_bid", "eve_character_contract_item",
	"eve_character_corp_role", "eve_character_fitting", "eve_character_fitting_item",
	"eve_character_implants", "eve_character_killmail", "eve_character_notification",
	"eve_character_skill", "eve_character_skill_queue", "eve_character_skills",
	"eve_character_title", "eve_character_wallet", "eve_character_wallet_journal",
	"eve_character_wallet_transaction", "eve_killmail_list",
}

func migrateRequiredUserContact(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if tx.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(20261110, 1)").Error; err != nil {
				return err
			}
		}
		if err := tx.AutoMigrate(&schemaMigration{}); err != nil {
			return err
		}
		var applied int64
		if err := tx.Model(&schemaMigration{}).Where("version = ?", userContactMigrationVersion).Count(&applied).Error; err != nil {
			return err
		}
		if applied > 0 {
			return nil
		}
		if tx.Name() == "postgres" {
			if err := tx.Exec(`LOCK TABLE "user" IN ACCESS EXCLUSIVE MODE`).Error; err != nil {
				return err
			}
		}
		// Use the application whitespace definition, including Unicode whitespace and NULLs.
		var users []model.User
		if err := tx.Unscoped().Select("id", "qq", "discord_id").Find(&users).Error; err != nil {
			return err
		}
		ids := make([]uint, 0)
		for _, user := range users {
			if !user.HasRequiredContact() {
				ids = append(ids, user.ID)
			}
		}
		if len(ids) > 0 {
			var characterIDs []int64
			if err := tx.Unscoped().Model(&model.EveCharacter{}).Where("user_id IN ?", ids).Pluck("character_id", &characterIDs).Error; err != nil {
				return err
			}
			if len(characterIDs) > 0 {
				for _, table := range contactCleanupCharacterTables {
					if !tx.Migrator().HasTable(table) {
						continue
					}
					if err := tx.Table(table).Where("character_id IN ?", characterIDs).Delete(map[string]any{}).Error; err != nil {
						return fmt.Errorf("cleanup %s: %w", table, err)
					}
				}
			}
			if tx.Migrator().HasTable("mumble_identity") {
				if err := tx.Table("mumble_identity").Where("seat_user_id IN ?", ids).Delete(map[string]any{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Unscoped().Where("user_id IN ?", ids).Delete(&model.UserRole{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("user_id IN ?", ids).Delete(&model.EveCharacter{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("id IN ?", ids).Delete(&model.User{}).Error; err != nil {
				return err
			}
		}
		for _, stmt := range []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_qq_unique ON "user" (qq) WHERE qq IS NOT NULL AND qq <> ''`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_discord_unique ON "user" (discord_id) WHERE discord_id IS NOT NULL AND discord_id <> ''`,
		} {
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}
		}
		if tx.Name() == "postgres" {
			// Match strings.TrimSpace without depending on the database locale.
			whitespace := `E' \u0009\u000a\u000b\u000c\u000d\u0085\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000'`
			stmt := fmt.Sprintf(`ALTER TABLE "user" ADD CONSTRAINT chk_user_required_contact CHECK (btrim(COALESCE(qq, ''), %s) <> '' OR btrim(COALESCE(discord_id, ''), %s) <> '')`, whitespace, whitespace)
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}
		}
		return tx.Create(&schemaMigration{Version: userContactMigrationVersion, AppliedAt: time.Now()}).Error
	})
}
