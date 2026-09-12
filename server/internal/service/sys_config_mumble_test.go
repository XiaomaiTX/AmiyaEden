package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSysConfigServiceMumbleConfig(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	db, err := gorm.Open(sqlite.Open("file:sys_config_mumble?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	global.DB = db
	svc := NewSysConfigService()

	want := MumbleRuntimeConfig{
		ServiceToken: "mumble-to-seat-token", ServerURL: "https://mumble.internal.example/",
		RevalidateToken: "seat-to-mumble-token", RevalidateTimeoutMS: 750,
	}
	if err := svc.UpdateMumbleConfig(want); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := svc.GetMumbleConfig()
	if got.ServerURL != "https://mumble.internal.example" || got.ServiceToken != want.ServiceToken || got.RevalidateToken != want.RevalidateToken || got.RevalidateTimeoutMS != want.RevalidateTimeoutMS {
		t.Fatalf("config = %+v", got)
	}
}

func TestSysConfigServiceMumbleConfigValidation(t *testing.T) {
	svc := NewSysConfigService()
	base := MumbleRuntimeConfig{
		ServiceToken: "mumble-to-seat-token", ServerURL: "https://mumble.internal.example",
		RevalidateToken: "seat-to-mumble-token", RevalidateTimeoutMS: 1000,
	}
	invalidURL := base
	invalidURL.ServerURL = "javascript:alert(1)"
	if err := svc.UpdateMumbleConfig(invalidURL); err == nil {
		t.Fatal("invalid URL must fail")
	}
	sameTokens := base
	sameTokens.RevalidateToken = sameTokens.ServiceToken
	if err := svc.UpdateMumbleConfig(sameTokens); err == nil {
		t.Fatal("same directional tokens must fail")
	}
	invalidTimeout := base
	invalidTimeout.RevalidateTimeoutMS = 0
	if err := svc.UpdateMumbleConfig(invalidTimeout); err == nil {
		t.Fatal("zero timeout must fail")
	}
}
