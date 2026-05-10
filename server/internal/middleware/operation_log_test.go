package middleware

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestOperationLogReadsUserIDFromAuthContextKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:operation_log_test?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	r := gin.New()
	r.Use(OperationLog())
	r.GET("/test-op-log", func(c *gin.Context) {
		c.Set("userID", uint(12345))
		c.Set(CtxKeyBizCode, 200)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test-op-log", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var logs []model.OperationLog
		if err := db.Find(&logs).Error; err != nil {
			t.Fatalf("query operation_log: %v", err)
		}
		if len(logs) == 0 {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		if logs[0].UserID != 12345 {
			t.Fatalf("operation_log.user_id = %d, want %d", logs[0].UserID, 12345)
		}
		return
	}

	t.Fatal("operation_log insert not observed within timeout")
}
