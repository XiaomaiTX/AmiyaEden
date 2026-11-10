package bootstrap

import (
	"amiya-eden/internal/model"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCustomIndexesKeepMentorAndRemoveCaptainBinding(t *testing.T) {
	for _, stmt := range newbroCustomIndexStatements() {
		if strings.Contains(stmt, "newbro_captain_affiliation") {
			t.Fatal("obsolete index remains")
		}
	}
	if !strings.Contains(strings.Join(newbroCustomIndexStatements(), " "), "idx_mentor_rel_active_mentee") {
		t.Fatal("mentor index missing")
	}
}

func TestObsoleteColumnDropsIncludeLegacyPortraitColumns(t *testing.T) {
	drops := obsoleteColumnDrops()
	joined := make([]string, 0, len(drops))
	for _, drop := range drops {
		joined = append(joined, drop.table+"."+drop.col)
	}

	for _, expected := range []string{"user.avatar", "eve_character.portrait_url", "hall_of_fame_card.avatar"} {
		if !strings.Contains(strings.Join(joined, "\n"), expected) {
			t.Fatalf("expected obsolete column drop %q, got %v", expected, joined)
		}
	}
}

func TestObsoleteTablesIncludeRemovedShopRedeemCodeTable(t *testing.T) {
	for _, table := range obsoleteTables() {
		if table == "shop_redeem_code" {
			return
		}
	}

	t.Fatalf("expected obsoleteTables to include %q", "shop_redeem_code")
}

func TestUserQQUniqueIndexEnforcesUniqueNonEmptyQQ(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user_qq_unique_index?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("auto migrate user: %v", err)
	}

	for _, stmt := range userCustomIndexStatements() {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create qq unique index: %v", err)
		}
	}

	if err := db.Create(&model.User{Nickname: "A", QQ: "12345"}).Error; err != nil {
		t.Fatalf("create first user: %v", err)
	}

	if err := db.Create(&model.User{Nickname: "B", QQ: "12345"}).Error; err == nil {
		t.Fatal("expected duplicate qq insert to fail")
	}

	if err := db.Create(&model.User{Nickname: "C", QQ: ""}).Error; err != nil {
		t.Fatalf("create first blank qq user: %v", err)
	}

	if err := db.Create(&model.User{Nickname: "D", QQ: ""}).Error; err != nil {
		t.Fatalf("create second blank qq user: %v", err)
	}
}

func TestCaptainCleanupIsIdempotentAndPreservesWalletAndOtherTasks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.TaskSchedule{}, &model.TaskExecution{}, &model.SystemConfig{}, &model.SystemWallet{}, &model.WalletTransaction{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"captain_attribution_sync", "captain_reward_processing", "mentor_reward", "recruit_link_check"} {
		if err = db.Create(&model.TaskSchedule{TaskName: name, CronExpr: "@every 1h"}).Error; err != nil {
			t.Fatal(err)
		}
		if err = db.Create(&model.TaskExecution{TaskName: name}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, key := range []string{"newbro.max_character_sp", "newbro.bonus_rate", "mentor.mentee_max_character_sp", "newbro.recruit_reward_amount", "newbro.refund_settings"} {
		if err = db.Create(&model.SystemConfig{Key: key, Value: "0"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err = db.Create(&model.SystemWallet{UserID: 1, Balance: 123}).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.WalletTransaction{UserID: 1, Amount: 123, RefType: model.WalletRefNewbroCaptainReward}).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = cleanupLegacyCaptainData(db); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"task_schedules", "task_executions", "system_config"} {
		var count int64
		if err = db.Table(table).Count(&count).Error; err != nil || count != 2 {
			t.Fatalf("table %s count %d error %v", table, count, err)
		}
	}
	var wallet model.SystemWallet
	if err = db.First(&wallet).Error; err != nil || wallet.Balance != 123 {
		t.Fatalf("wallet modified %+v %v", wallet, err)
	}
	var count int64
	db.Model(&model.WalletTransaction{}).Count(&count)
	if count != 1 {
		t.Fatal("historical wallet row deleted")
	}
	for _, table := range []string{"newbro_captain_affiliation", "captain_bounty_attribution", "captain_bounty_sync_state", "captain_reward_settlement", "newbro_player_state"} {
		found := false
		for _, obsolete := range obsoleteTables() {
			if obsolete == table {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing obsolete table %s", table)
		}
	}
}
