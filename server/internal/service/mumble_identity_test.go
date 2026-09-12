package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupMumbleIdentityTest(t *testing.T) *MumbleIdentityService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:mumble_identity_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.EveCharacter{}, &model.UserRole{}, &model.MumbleIdentity{}, &model.AuditEvent{}, &model.SystemConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	global.DB = db
	user := model.User{BaseModel: model.BaseModel{ID: 42}, Status: 1, Role: model.RoleUser, PrimaryCharacterID: 900001}
	char := model.EveCharacter{CharacterID: 900001, CharacterName: "Primary Pilot", UserID: 42}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&char).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.UserRole{UserID: 42, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.UserRole{UserID: 42, RoleCode: model.RoleFC}).Error; err != nil {
		t.Fatal(err)
	}
	return NewMumbleIdentityService()
}

func TestMumbleIdentityServiceCredentialAndEntitlement(t *testing.T) {
	svc := setupMumbleIdentityTest(t)
	status, password, err := svc.CreateCredential(42)
	if err != nil || password == "" || status.StableUserID == 0 {
		t.Fatalf("create credential: status=%+v err=%v", status, err)
	}
	if status.StableUserID == 42 {
		t.Fatal("stable Mumble user id must not reuse Seat user id")
	}

	claims, err := svc.Authenticate("Primary Pilot", password)
	if err != nil || !claims.Eligible || claims.Name != "Primary Pilot" {
		t.Fatalf("authenticate: claims=%+v err=%v", claims, err)
	}
	if !containsMumbleGroup(claims.Groups, "fuxi_role_fc") || !containsMumbleGroup(claims.Groups, "fuxi_role_user") {
		t.Fatalf("missing multi-role claims: %v", claims.Groups)
	}
	if _, err := svc.Authenticate("Primary Pilot", "wrong"); err == nil {
		t.Fatal("wrong password must deny")
	}
	if _, err := svc.Authenticate("SuperUser", password); err == nil {
		t.Fatal("SuperUser must never be externally claimed")
	}

	beforeID := claims.StableUserID
	_, nextPassword, err := svc.RotateCredential(42)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if _, err := svc.Authenticate("Primary Pilot", password); err == nil {
		t.Fatal("rotated password must immediately stop working")
	}
	claims, err = svc.Authenticate("Primary Pilot", nextPassword)
	if err != nil || claims.StableUserID != beforeID {
		t.Fatalf("rotation changed stable id or denied: %+v %v", claims, err)
	}

	if err := global.DB.Model(&model.User{}).Where("id = 42").Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate("Primary Pilot", nextPassword); err == nil {
		t.Fatal("disabled user must deny")
	}
	if err := global.DB.Model(&model.User{}).Where("id = 42").Update("status", 1).Error; err != nil {
		t.Fatal(err)
	}
	if err := global.DB.Where("user_id = ?", 42).Delete(&model.UserRole{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := global.DB.Create(&model.UserRole{UserID: 42, RoleCode: model.RoleGuest}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate("Primary Pilot", nextPassword); err == nil {
		t.Fatal("guest user must deny")
	}
}

func TestMumbleIdentityServicePrimaryCharacterAndRevoke(t *testing.T) {
	svc := setupMumbleIdentityTest(t)
	status, password, err := svc.CreateCredential(42)
	if err != nil {
		t.Fatal(err)
	}
	newChar := model.EveCharacter{CharacterID: 900002, CharacterName: "New Primary", UserID: 42}
	if err := global.DB.Create(&newChar).Error; err != nil {
		t.Fatal(err)
	}
	if err := global.DB.Model(&model.User{}).Where("id = 42").Update("primary_character_id", 900002).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate("Primary Pilot", password); err == nil {
		t.Fatal("old primary name must deny")
	}
	claims, err := svc.Authenticate("New Primary", password)
	if err != nil || claims.StableUserID != status.StableUserID {
		t.Fatalf("primary change must retain identity: %+v %v", claims, err)
	}
	if err := global.DB.Model(&model.User{}).Where("id = 42").Update("primary_character_id", 0).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate("New Primary", password); err == nil {
		t.Fatal("user without primary character must deny")
	}
	if err := global.DB.Model(&model.User{}).Where("id = 42").Update("primary_character_id", 900002).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.RevokeCredential(42); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate("New Primary", password); err == nil {
		t.Fatal("revoked credential must deny")
	}
}

func containsMumbleGroup(groups []string, want string) bool {
	for _, group := range groups {
		if group == want {
			return true
		}
	}
	return false
}
