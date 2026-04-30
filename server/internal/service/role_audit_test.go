package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSetUserRolesWritesSuccessAuditEvent(t *testing.T) {
	db := newRoleAuditTestDB(t)
	originalDB := global.DB
	originalRedis := global.Redis
	global.DB = db
	global.Redis = redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:0",
		DialTimeout:  1 * time.Millisecond,
		ReadTimeout:  1 * time.Millisecond,
		WriteTimeout: 1 * time.Millisecond,
	})
	defer func() {
		global.DB = originalDB
		global.Redis = originalRedis
	}()

	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 101}, Role: model.RoleGuest}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: 101, RoleCode: model.RoleGuest}).Error; err != nil {
		t.Fatalf("create initial role: %v", err)
	}

	svc := NewRoleService()
	if err := svc.SetUserRoles(context.Background(), 7, []string{model.RoleSuperAdmin}, 101, []string{model.RoleUser, model.RoleFC}); err != nil {
		t.Fatalf("SetUserRoles() error = %v", err)
	}

	var events []model.AuditEvent
	if err := db.Order("id ASC").Find(&events).Error; err != nil {
		t.Fatalf("load audit events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("audit event count = %d, want 1", len(events))
	}
	if events[0].Category != "permission" || events[0].Action != "set_user_roles" {
		t.Fatalf("unexpected audit event type: %+v", events[0])
	}
	if events[0].Result != model.AuditResultSuccess {
		t.Fatalf("audit result = %q, want %q", events[0].Result, model.AuditResultSuccess)
	}
	if events[0].ActorUserID != 7 || events[0].TargetUserID != 101 {
		t.Fatalf("unexpected audit actor/target: %+v", events[0])
	}
}

func TestSetUserRolesWritesFailedAuditEvent(t *testing.T) {
	db := newRoleAuditTestDB(t)
	originalDB := global.DB
	originalRedis := global.Redis
	global.DB = db
	global.Redis = redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:0",
		DialTimeout:  1 * time.Millisecond,
		ReadTimeout:  1 * time.Millisecond,
		WriteTimeout: 1 * time.Millisecond,
	})
	defer func() {
		global.DB = originalDB
		global.Redis = originalRedis
	}()

	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 202}, Role: model.RoleGuest}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: 202, RoleCode: model.RoleGuest}).Error; err != nil {
		t.Fatalf("create initial role: %v", err)
	}

	svc := NewRoleService()
	err := svc.SetUserRoles(context.Background(), 9, []string{model.RoleAdmin}, 202, []string{model.RoleSuperAdmin})
	if err == nil {
		t.Fatal("expected SetUserRoles() to fail for super_admin assignment")
	}

	var events []model.AuditEvent
	if err := db.Order("id ASC").Find(&events).Error; err != nil {
		t.Fatalf("load audit events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("audit event count = %d, want 1", len(events))
	}
	if events[0].Result != model.AuditResultFailed {
		t.Fatalf("audit result = %q, want %q", events[0].Result, model.AuditResultFailed)
	}
}

func newRoleAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:role_audit_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserRole{}, &model.AuditEvent{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}
