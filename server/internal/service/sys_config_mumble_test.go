package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"strings"
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
		PublicAddress: " mumble.example.com ", PublicPort: 64738, DisplayNameTemplate: " {nickname} ({character_name}) ",
	}
	if err := svc.UpdateMumbleConfig(want); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := svc.GetMumbleConfig()
	if got.ServerURL != "https://mumble.internal.example" || got.ServiceToken != want.ServiceToken || got.RevalidateToken != want.RevalidateToken || got.RevalidateTimeoutMS != want.RevalidateTimeoutMS {
		t.Fatalf("config = %+v", got)
	}
	if got.PublicAddress != "mumble.example.com" || got.PublicPort != 64738 {
		t.Fatalf("public connection config = %+v", got)
	}
	if got.DisplayNameTemplate != "{nickname} ({character_name})" {
		t.Fatalf("display name template = %q", got.DisplayNameTemplate)
	}
	if err := svc.UpdateMumbleConfig(MumbleRuntimeConfig{RevalidateTimeoutMS: 1000}); err != nil {
		t.Fatalf("zero-value update: %v", err)
	}
	got = svc.GetMumbleConfig()
	if got.PublicAddress != "" || got.PublicPort != 0 {
		t.Fatalf("unset public connection config = %+v", got)
	}
	if got.DisplayNameTemplate != defaultMumbleDisplayNameTemplate {
		t.Fatalf("empty template must use default, got %q", got.DisplayNameTemplate)
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
	invalidAddress := base
	invalidAddress.PublicAddress = "mumble.example.com/trailing"
	if err := svc.UpdateMumbleConfig(invalidAddress); err == nil {
		t.Fatal("address with path must fail")
	}
	invalidAddress = base
	invalidAddress.PublicAddress = "mumble example.com"
	if err := svc.UpdateMumbleConfig(invalidAddress); err == nil {
		t.Fatal("address with whitespace must fail")
	}
	invalidAddress = base
	invalidAddress.PublicAddress = strings.Repeat("a", 254)
	if err := svc.UpdateMumbleConfig(invalidAddress); err == nil {
		t.Fatal("overlong address must fail")
	}
	invalidPort := base
	invalidPort.PublicPort = -1
	if err := svc.UpdateMumbleConfig(invalidPort); err == nil {
		t.Fatal("negative port must fail")
	}
	invalidPort = base
	invalidPort.PublicPort = 65536
	if err := svc.UpdateMumbleConfig(invalidPort); err == nil {
		t.Fatal("port above 65535 must fail")
	}
	invalidTemplate := base
	invalidTemplate.DisplayNameTemplate = "{unknown}"
	if err := svc.UpdateMumbleConfig(invalidTemplate); err == nil {
		t.Fatal("unknown display name placeholder must fail")
	}
}
