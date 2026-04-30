package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/pkg/response"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type webhookHandlerResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func setupWebhookHandlerTestDB(t *testing.T) {
	t.Helper()

	dsn := "file:webhook_handler_test_" + time.Now().Format("20060102150405.000000000") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.AuditEvent{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
}

func decodeWebhookResp(t *testing.T, recorder *httptest.ResponseRecorder) webhookHandlerResp {
	t.Helper()
	var out webhookHandlerResp
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func TestWebhookHandlerSetConfigWritesAuditEvent(t *testing.T) {
	setupWebhookHandlerTestDB(t)

	gin.SetMode(gin.TestMode)
	h := NewWebhookHandler()
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(123))
		c.Next()
	})
	r.PUT("/api/v1/system/webhook/config", h.SetConfig)

	reqBody := []byte(`{
		"url":"https://discord.com/api/webhooks/1/token",
		"enabled":true,
		"type":"discord",
		"fleet_template":"test",
		"ob_target_type":"group",
		"ob_target_id":42,
		"ob_token":"token"
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/system/webhook/config", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("http status = %d, want %d", rec.Code, http.StatusOK)
	}
	resp := decodeWebhookResp(t, rec)
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}

	var events []model.AuditEvent
	if err := global.DB.Where("action = ?", "webhook_config_update").Order("id DESC").Find(&events).Error; err != nil {
		t.Fatalf("load audit events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected webhook_config_update audit event")
	}
	if events[0].Category != "config" || events[0].ActorUserID != 123 || events[0].Result != model.AuditResultSuccess {
		t.Fatalf("unexpected audit event: %+v", events[0])
	}
}

func TestWebhookHandlerSetConfigRejectsInvalidURL(t *testing.T) {
	setupWebhookHandlerTestDB(t)

	gin.SetMode(gin.TestMode)
	h := NewWebhookHandler()
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(123))
		c.Next()
	})
	r.PUT("/api/v1/system/webhook/config", h.SetConfig)

	reqBody := []byte(`{
		"url":"https://example.com/webhook",
		"enabled":true,
		"type":"discord",
		"fleet_template":"test",
		"ob_target_type":"group",
		"ob_target_id":42,
		"ob_token":"token"
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/system/webhook/config", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("http status = %d, want %d", rec.Code, http.StatusOK)
	}
	resp := decodeWebhookResp(t, rec)
	if resp.Code != response.CodeBizError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeBizError)
	}
}

func TestWebhookHandlerTestWebhookRejectsInvalidURL(t *testing.T) {
	setupWebhookHandlerTestDB(t)

	gin.SetMode(gin.TestMode)
	h := NewWebhookHandler()
	r := gin.New()
	r.POST("/api/v1/system/webhook/test", h.TestWebhook)

	reqBody := []byte(`{
		"url":"ftp://onebot.example.com",
		"type":"onebot",
		"ob_target_type":"group",
		"ob_target_id":42
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/webhook/test", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("http status = %d, want %d", rec.Code, http.StatusOK)
	}
	resp := decodeWebhookResp(t, rec)
	if resp.Code != response.CodeBizError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeBizError)
	}
}
