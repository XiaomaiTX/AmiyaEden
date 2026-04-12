package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"amiya-eden/global"
	"amiya-eden/internal/model"

	"gorm.io/driver/sqlite"
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
	if err := db.AutoMigrate(&model.User{}, &model.SystemConfig{}, &model.NewbroRecruitment{}); err != nil {
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
