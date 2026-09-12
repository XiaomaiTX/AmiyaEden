package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNotifyMumbleIdentityChanged(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	db, err := gorm.Open(sqlite.Open("file:mumble_revalidation?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.MumbleIdentity{}, &model.SystemConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	identity := model.MumbleIdentity{BaseModel: model.BaseModel{ID: 73}, SeatUserID: 42}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatalf("create identity: %v", err)
	}
	global.DB = db

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/api/internal/identity/v1/revalidate" {
			t.Errorf("request path = %q, want %q", r.URL.Path, "/api/internal/identity/v1/revalidate")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer reverse-token" {
			t.Errorf("authorization = %q", got)
		}
		var body struct {
			UserID uint32 `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.UserID != 73 {
			t.Errorf("user_id = %d, want 73", body.UserID)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	settings := []model.SystemConfig{
		{Key: model.SysConfigMumbleServerURL, Value: server.URL},
		{Key: model.SysConfigMumbleRevalidateToken, Value: "reverse-token"},
		{Key: model.SysConfigMumbleRevalidateTimeoutMS, Value: "500"},
	}
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("create settings: %v", err)
	}

	if err := notifyMumbleIdentityChanged(context.Background(), 42); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if !called {
		t.Fatal("expected revalidation callback")
	}
}

func TestNotifyMumbleIdentityChangedDisabledIsNoop(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	db, err := gorm.Open(sqlite.Open("file:mumble_revalidation_disabled?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.MumbleIdentity{}, &model.SystemConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	global.DB = db
	if err := notifyMumbleIdentityChanged(context.Background(), 42); err != nil {
		t.Fatalf("disabled notifier returned error: %v", err)
	}
}
