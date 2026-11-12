package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newBlacklistRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:blacklist_repo_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.BlacklistEntry{}, &model.BlacklistUnresolvedName{}, &model.BlacklistRefreshRun{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() {
		global.DB = oldDB
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	return db
}

func seedBlacklistEntries(t *testing.T, db *gorm.DB) {
	t.Helper()
	rows := []model.BlacklistEntry{
		{Kind: model.BlacklistKindCharacter, Value: "90000001", Display: "Akina Pilot", Reason: "scam"},
		{Kind: model.BlacklistKindCharacter, Value: "123456789", Display: "Zed Alt", Reason: "awox"},
		{Kind: model.BlacklistKindQQ, Value: "123456789", Display: "123456789", Reason: "scam"},
		{Kind: model.BlacklistKindQQ, Value: "1234567890", Display: "1234567890", Reason: "spy"},
		{Kind: model.BlacklistKindQQ, Value: "12345678", Display: "12345678", Reason: "spy"},
		{Kind: model.BlacklistKindDiscord, Value: "akina suzuki#7576", Display: "Akina Suzuki#7576", Reason: "scam"},
		{Kind: model.BlacklistKindDiscord, Value: "under_score", Display: "under_score", Reason: "spy"},
		{Kind: model.BlacklistKindDiscord, Value: "underxscore", Display: "underXscore", Reason: "spy"},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed entries: %v", err)
	}
}

func blacklistKeys(rows []model.BlacklistEntry) []string {
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.Kind+":"+row.Value)
	}
	return keys
}

func TestBlacklistRepositoryListFilters(t *testing.T) {
	db := newBlacklistRepoTestDB(t)
	seedBlacklistEntries(t, db)
	repo := NewBlacklistRepository()

	cases := []struct {
		name   string
		filter BlacklistEntryFilter
		want   []string
	}{
		{"all ordered by kind then display", BlacklistEntryFilter{}, []string{
			"character:90000001", "character:123456789",
			"discord:akina suzuki#7576", "discord:under_score", "discord:underxscore",
			"qq:12345678", "qq:123456789", "qq:1234567890",
		}},
		{"digits match QQ and character ID exactly", BlacklistEntryFilter{Keyword: "123456789"}, []string{"character:123456789", "qq:123456789"}},
		{"digits never match a longer or shorter number", BlacklistEntryFilter{Keyword: "1234567"}, []string{}},
		{"text matches names case-insensitively", BlacklistEntryFilter{Keyword: " AKINA "}, []string{"character:90000001", "discord:akina suzuki#7576"}},
		{"text never searches QQ numbers", BlacklistEntryFilter{Keyword: "1234a"}, []string{}},
		{"underscore is literal, not a wildcard", BlacklistEntryFilter{Keyword: "r_s"}, []string{"discord:under_score"}},
		{"kind filter", BlacklistEntryFilter{Kind: model.BlacklistKindDiscord}, []string{"discord:akina suzuki#7576", "discord:under_score", "discord:underxscore"}},
		{"kind and digits keyword", BlacklistEntryFilter{Kind: model.BlacklistKindQQ, Keyword: "123456789"}, []string{"qq:123456789"}},
		{"kind and text keyword", BlacklistEntryFilter{Kind: model.BlacklistKindCharacter, Keyword: "akina"}, []string{"character:90000001"}},
		{"QQ kind with text keyword is empty", BlacklistEntryFilter{Kind: model.BlacklistKindQQ, Keyword: "akina"}, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows, total, err := repo.List(1, 50, tc.filter)
			if err != nil {
				t.Fatal(err)
			}
			got := blacklistKeys(rows)
			if fmt.Sprint(got) != fmt.Sprint(tc.want) || total != int64(len(tc.want)) {
				t.Fatalf("got %v (total %d), want %v", got, total, tc.want)
			}
		})
	}
}

func TestBlacklistRepositoryFindEntries(t *testing.T) {
	db := newBlacklistRepoTestDB(t)
	seedBlacklistEntries(t, db)
	repo := NewBlacklistRepository()

	for _, tc := range []struct {
		name                      string
		characters, qqs, discords []string
		want                      string
	}{
		{"each kind", []string{"90000001"}, []string{"12345678"}, []string{"under_score"}, "[character:90000001 discord:under_score qq:12345678]"},
		{"exact values only", []string{"9000000"}, []string{"1234567", "123456789012"}, []string{"under"}, "[]"},
		{"kinds do not cross", []string{"12345678"}, []string{"90000001"}, []string{"123456789"}, "[]"},
		{"same digits per kind", []string{"123456789"}, []string{"123456789"}, nil, "[character:123456789 qq:123456789]"},
		{"one set only", nil, nil, []string{"akina suzuki#7576", "nobody"}, "[discord:akina suzuki#7576]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := repo.FindEntriesTx(db, tc.characters, tc.qqs, tc.discords)
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprint(blacklistKeys(rows)); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}

	// Nothing to look up: returns without touching the database.
	rows, err := repo.FindEntriesTx(nil, nil, []string{}, nil)
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty input = %v, %v", rows, err)
	}
}

func TestBlacklistRepositoryListPagination(t *testing.T) {
	db := newBlacklistRepoTestDB(t)
	seedBlacklistEntries(t, db)
	repo := NewBlacklistRepository()

	rows, total, err := repo.List(3, 3, BlacklistEntryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 8 || fmt.Sprint(blacklistKeys(rows)) != "[qq:123456789 qq:1234567890]" {
		t.Fatalf("last page = %v (total %d)", blacklistKeys(rows), total)
	}
	rows, total, err = repo.List(4, 3, BlacklistEntryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 8 || len(rows) != 0 {
		t.Fatalf("page past the end = %v (total %d)", blacklistKeys(rows), total)
	}
}

func TestBlacklistRepositoryWritesRollBackTogether(t *testing.T) {
	db := newBlacklistRepoTestDB(t)
	seedBlacklistEntries(t, db)
	repo := NewBlacklistRepository()
	if err := db.Create(&model.BlacklistUnresolvedName{NameKey: "ghost", Name: "Ghost", Kind: model.BlacklistUnresolvedNotFound}).Error; err != nil {
		t.Fatal(err)
	}

	errAbort := errors.New("abort")
	err := db.Transaction(func(tx *gorm.DB) error {
		entries, err := repo.ListAllEntriesTx(tx)
		if err != nil {
			return err
		}
		if err := repo.DeleteEntriesTx(tx, []uint{entries[0].ID, entries[1].ID}); err != nil {
			return err
		}
		if err := repo.UpdateEntryTx(tx, entries[2].ID, "changed", "changed"); err != nil {
			return err
		}
		if err := repo.CreateEntriesTx(tx, []model.BlacklistEntry{{Kind: model.BlacklistKindQQ, Value: "99999", Display: "99999"}}); err != nil {
			return err
		}
		unresolved, err := repo.ListUnresolvedNamesTx(tx)
		if err != nil {
			return err
		}
		if err := repo.DeleteUnresolvedNamesTx(tx, []uint{unresolved[0].ID}); err != nil {
			return err
		}
		if err := repo.CreateUnresolvedNamesTx(tx, []model.BlacklistUnresolvedName{{NameKey: "new", Name: "New", Kind: model.BlacklistUnresolvedInvalidFormat}}); err != nil {
			return err
		}
		if err := repo.CreateRefreshRunTx(tx, &model.BlacklistRefreshRun{OperatorUserID: 1, FinishedAt: time.Now()}); err != nil {
			return err
		}
		return errAbort
	})
	if !errors.Is(err, errAbort) {
		t.Fatalf("transaction error = %v", err)
	}

	entries, err := repo.ListAllEntriesTx(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 8 || entries[2].Display != "123456789" {
		t.Fatalf("entries changed after rollback: %v", blacklistKeys(entries))
	}
	unresolved, err := repo.ListUnresolvedNamesTx(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(unresolved) != 1 || unresolved[0].NameKey != "ghost" {
		t.Fatalf("unresolved names changed after rollback: %+v", unresolved)
	}
	if run, err := repo.LatestRefreshRun(); err != nil || run != nil {
		t.Fatalf("refresh run kept after rollback: %+v, %v", run, err)
	}
}

func TestBlacklistRepositoryWritesCommit(t *testing.T) {
	db := newBlacklistRepoTestDB(t)
	seedBlacklistEntries(t, db)
	repo := NewBlacklistRepository()

	err := db.Transaction(func(tx *gorm.DB) error {
		entries, err := repo.ListAllEntriesTx(tx)
		if err != nil {
			return err
		}
		if err := repo.DeleteEntriesTx(tx, []uint{entries[0].ID}); err != nil {
			return err
		}
		if err := repo.UpdateEntryTx(tx, entries[1].ID, "Zed Alt", "awox\nspy"); err != nil {
			return err
		}
		return repo.CreateEntriesTx(tx, []model.BlacklistEntry{{Kind: model.BlacklistKindQQ, Value: "99999", Display: "99999", Reason: "new"}})
	})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := repo.ListAllEntriesTx(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 8 || entries[0].Value != "123456789" || entries[0].Reason != "awox\nspy" || entries[7].Value != "99999" {
		t.Fatalf("unexpected entries after commit: %+v", entries)
	}
}

func TestBlacklistRepositoryLatestRefreshRun(t *testing.T) {
	db := newBlacklistRepoTestDB(t)
	repo := NewBlacklistRepository()
	if run, err := repo.LatestRefreshRun(); err != nil || run != nil {
		t.Fatalf("empty state = %+v, %v", run, err)
	}
	now := time.Now()
	for i, finished := range []time.Time{now.Add(-time.Hour), now, now.Add(-2 * time.Hour)} {
		if err := repo.CreateRefreshRunTx(db, &model.BlacklistRefreshRun{OperatorUserID: uint(i + 1), FinishedAt: finished}); err != nil {
			t.Fatal(err)
		}
	}
	run, err := repo.LatestRefreshRun()
	if err != nil || run == nil || run.OperatorUserID != 2 {
		t.Fatalf("latest run = %+v, %v", run, err)
	}
}
