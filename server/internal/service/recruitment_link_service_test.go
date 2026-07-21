package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"amiya-eden/global"
	"amiya-eden/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newRecruitmentLinkServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:recruit_link_svc_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.SystemConfig{},
		&model.EveCharacter{},
		&model.NewbroRecruitment{},
		&model.NewbroRecruitmentEntry{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func base62Decode(s string) uint {
	var n uint
	for _, c := range s {
		n = n*62 + uint(strings.IndexRune(base62Chars, c))
	}
	return n
}

func TestBase62Encode_KnownValues(t *testing.T) {
	cases := []struct {
		n    uint
		want string
	}{
		{0, "0"},
		{1, "1"},
		{61, "Z"},
		{62, "10"},
	}
	for _, c := range cases {
		got := base62Encode(c.n)
		if got != c.want {
			t.Errorf("base62Encode(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestBase62Decode_RoundTrip(t *testing.T) {
	for _, n := range []uint{0, 1, 61, 62, 1000, 9999, 238328} {
		code := base62Encode(n)
		got := base62Decode(code)
		if got != n {
			t.Errorf("round-trip(%d): encode=%q decode=%d", n, code, got)
		}
	}
}

func TestRecruitmentLinkService_GenerateLinkCreatesRecordWithCode(t *testing.T) {
	db := newRecruitmentLinkServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	user := &model.User{Nickname: "recruiter", QQ: "123456", Role: model.RoleUser}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := NewRecruitmentLinkService()
	rec, created, err := svc.GenerateLink(user.ID, time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GenerateLink() error = %v", err)
	}
	if !created {
		t.Fatal("expected a new recruitment link to be created")
	}
	if rec == nil || rec.ID == 0 {
		t.Fatalf("expected persisted recruitment record, got %+v", rec)
	}
	if rec.Code == "" {
		t.Fatal("expected generated recruitment code to be populated")
	}

	var stored model.NewbroRecruitment
	if err := db.First(&stored, rec.ID).Error; err != nil {
		t.Fatalf("load stored recruitment: %v", err)
	}
	if stored.Code != rec.Code {
		t.Fatalf("expected stored code %q to match returned code %q", stored.Code, rec.Code)
	}
}

func TestRecruitmentLinkService_GenerateLinkRejectsCooldownWindow(t *testing.T) {
	db := newRecruitmentLinkServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	user := &model.User{Nickname: "recruiter", QQ: "123456", Role: model.RoleUser}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	if err := db.Create(&model.NewbroRecruitment{
		UserID:      user.ID,
		Code:        "existing",
		GeneratedAt: now.Add(-24 * time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed existing recruitment: %v", err)
	}

	svc := NewRecruitmentLinkService()
	rec, created, err := svc.GenerateLink(user.ID, now)
	if err == nil {
		t.Fatal("expected cooldown error, got nil")
	}
	if created {
		t.Fatal("expected cooldown check to block new link creation")
	}
	if rec != nil {
		t.Fatalf("expected no recruitment record on cooldown, got %+v", rec)
	}
}

func TestRecruitmentLinkService_ListAllLinksIncludesCharacterNames(t *testing.T) {
	db := newRecruitmentLinkServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	recruiter := &model.User{Nickname: "RecruiterNick", QQ: "11111", Role: model.RoleUser, PrimaryCharacterID: 1001}
	matchedWithCharacter := &model.User{Nickname: "MatchedNick", QQ: "22222", Role: model.RoleUser, PrimaryCharacterID: 2002}
	matchedWithoutCharacter := &model.User{Nickname: "FallbackNick", QQ: "33333", Role: model.RoleUser, PrimaryCharacterID: 3003}
	if err := db.Create(recruiter).Error; err != nil {
		t.Fatalf("seed recruiter: %v", err)
	}
	if err := db.Create(matchedWithCharacter).Error; err != nil {
		t.Fatalf("seed matched user with character: %v", err)
	}
	if err := db.Create(matchedWithoutCharacter).Error; err != nil {
		t.Fatalf("seed matched user without character: %v", err)
	}
	if err := db.Create(&model.EveCharacter{UserID: recruiter.ID, CharacterID: 1001, CharacterName: "RecruiterMain"}).Error; err != nil {
		t.Fatalf("seed recruiter character: %v", err)
	}
	if err := db.Create(&model.EveCharacter{UserID: matchedWithCharacter.ID, CharacterID: 2002, CharacterName: "MatchedMain"}).Error; err != nil {
		t.Fatalf("seed matched character: %v", err)
	}

	recruitment := &model.NewbroRecruitment{
		UserID:      recruiter.ID,
		Code:        "abc",
		Source:      model.RecruitmentSourceLink,
		GeneratedAt: time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC),
	}
	if err := db.Create(recruitment).Error; err != nil {
		t.Fatalf("seed recruitment: %v", err)
	}

	rewardedAt := time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC)
	entries := []model.NewbroRecruitmentEntry{
		{
			RecruitmentID: recruitment.ID,
			QQ:            "qq-1",
			EnteredAt:     time.Date(2026, 4, 12, 15, 0, 0, 0, time.UTC),
			Source:        model.RecruitEntrySourceLink,
			Status:        model.RecruitEntryStatusValid,
			MatchedUserID: matchedWithCharacter.ID,
			RewardedAt:    &rewardedAt,
		},
		{
			RecruitmentID: recruitment.ID,
			QQ:            "qq-2",
			EnteredAt:     time.Date(2026, 4, 12, 16, 0, 0, 0, time.UTC),
			Source:        model.RecruitEntrySourceLink,
			Status:        model.RecruitEntryStatusValid,
			MatchedUserID: matchedWithoutCharacter.ID,
			RewardedAt:    &rewardedAt,
		},
		{
			RecruitmentID: recruitment.ID,
			QQ:            "qq-3",
			EnteredAt:     time.Date(2026, 4, 12, 17, 0, 0, 0, time.UTC),
			Source:        model.RecruitEntrySourceLink,
			Status:        model.RecruitEntryStatusOngoing,
		},
	}
	if err := db.Create(&entries).Error; err != nil {
		t.Fatalf("seed entries: %v", err)
	}

	svc := NewRecruitmentLinkService()
	rows, total, err := svc.ListAllLinks(1, 20)
	if err != nil {
		t.Fatalf("ListAllLinks() error = %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total=1, got %d", total)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if row.UserCharacterName != "RecruiterMain" {
		t.Fatalf("expected recruiter main character name, got %q", row.UserCharacterName)
	}
	if len(row.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(row.Entries))
	}

	entryByQQ := make(map[string]RecruitEntryRow, len(row.Entries))
	for _, entry := range row.Entries {
		entryByQQ[entry.QQ] = entry
	}

	if entryByQQ["qq-1"].MatchedCharacterName != "MatchedMain" {
		t.Fatalf("expected matched character name from character table, got %q", entryByQQ["qq-1"].MatchedCharacterName)
	}
	if entryByQQ["qq-2"].MatchedCharacterName != "FallbackNick" {
		t.Fatalf("expected matched character name fallback to nickname, got %q", entryByQQ["qq-2"].MatchedCharacterName)
	}
	if entryByQQ["qq-3"].MatchedCharacterName != "" {
		t.Fatalf("expected non-valid entry matched character name to be empty, got %q", entryByQQ["qq-3"].MatchedCharacterName)
	}
}
