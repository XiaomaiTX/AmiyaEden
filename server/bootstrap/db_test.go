package bootstrap

import (
	"amiya-eden/internal/model"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCustomIndexStatementsIncludeActiveAffiliationUniqueness(t *testing.T) {
	stmts := newbroCustomIndexStatements()
	if len(stmts) == 0 {
		t.Fatal("expected custom index statements")
	}

	found := false
	for _, stmt := range stmts {
		if strings.Contains(stmt, "newbro_captain_affiliation") &&
			strings.Contains(stmt, "UNIQUE INDEX") &&
			strings.Contains(stmt, "player_user_id") &&
			strings.Contains(stmt, "ended_at IS NULL") &&
			strings.Contains(stmt, "deleted_at IS NULL") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected active affiliation uniqueness statement, got %v", stmts)
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
