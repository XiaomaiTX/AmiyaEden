package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCorpKillmailsTaskLoadExistingKillmailsReturnsExistingRows(t *testing.T) {
	db := newCorpKillmailTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	if err := db.Create(&model.EveKillmailList{KillmailID: 9001, CharacterID: 1001}).Error; err != nil {
		t.Fatalf("create killmail: %v", err)
	}

	task := &CorpKillmailsTask{}
	existing, err := task.loadExistingKillmails([]int64{9001, 9002})
	if err != nil {
		t.Fatalf("load existing killmails: %v", err)
	}
	if _, ok := existing[9001]; !ok {
		t.Fatal("expected existing killmail 9001 to be returned")
	}
	if _, ok := existing[9002]; ok {
		t.Fatal("expected missing killmail 9002 to stay absent")
	}
}

func TestCorpKillmailsTaskEnsureVictimLinksCreatesMissingLinksIdempotently(t *testing.T) {
	db := newCorpKillmailTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	killmail := model.EveKillmailList{KillmailID: 9001, CharacterID: 1001}
	if err := db.Create(&killmail).Error; err != nil {
		t.Fatalf("create killmail: %v", err)
	}

	task := &CorpKillmailsTask{}
	existing := map[int64]model.EveKillmailList{9001: killmail}
	knownChars := map[int64]bool{1001: true}

	if err := task.ensureVictimLinks(existing, knownChars); err != nil {
		t.Fatalf("ensure victim links: %v", err)
	}
	if err := task.ensureVictimLinks(existing, knownChars); err != nil {
		t.Fatalf("ensure victim links second call: %v", err)
	}

	var links []model.EveCharacterKillmail
	if err := db.Order("id asc").Find(&links).Error; err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected exactly one victim link, got %d", len(links))
	}
	if links[0].CharacterID != 1001 || links[0].KillmailID != 9001 || !links[0].Victim {
		t.Fatalf("unexpected link: %+v", links[0])
	}
}

func newCorpKillmailTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:corp_killmail_task_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.EveKillmailList{}, &model.EveCharacterKillmail{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}
