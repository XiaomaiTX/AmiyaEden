package middleware

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/pkg/jwt"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestJWTAuthRejectsRemovedAndDisabledAccounts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:auth_account_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB; sqlDB, _ := db.DB(); _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	jwt.Init("auth-account-test-secret")
	user := &model.User{QQ: "123456"}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(user).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint{user.ID, 9999} {
		token, err := jwt.GenerateToken(id, 9001, model.RoleSuperAdmin, 7)
		if err != nil {
			t.Fatal(err)
		}
		router := gin.New()
		router.GET("/private", JWTAuth(), func(c *gin.Context) { t.Error("removed/disabled account reached handler") })
		req := httptest.NewRequest("GET", "/private", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
}

func TestJWTAuthKeepsSessionOnTransientLookupFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:auth_transient_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()
	jwt.Init("auth-transient-test-secret")
	token, err := jwt.GenerateToken(1, 9001, model.RoleSuperAdmin, 7)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/private", JWTAuth(), func(c *gin.Context) { t.Error("failed lookup reached handler") })
	req := httptest.NewRequest("GET", "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code == 401 {
		t.Fatalf("transient lookup failure must not end the session: body=%s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":500`) {
		t.Fatalf("expected internal error response, got status=%d body=%s", w.Code, w.Body.String())
	}
}
