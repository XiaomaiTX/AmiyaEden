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
	"github.com/glebarez/sqlite"
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

func TestWebhookHandlerRequiredBodiesFailInBinder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	operations := []struct {
		name   string
		method string
		path   string
	}{
		{"setWebhookConfig", http.MethodPut, "/api/v1/system/webhook/config"},
		{"testWebhook", http.MethodPost, "/api/v1/system/webhook/test"},
	}
	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			for _, body := range []struct{ name, json string }{
				{"empty", ""},
				{"wrong type", `{"url":false}`},
			} {
				t.Run(body.name, func(t *testing.T) {
					r := gin.New()
					h := newMigratedOperationAdapter(migratedOperations{WebhookHandler: &WebhookHandler{}})
					var called bool
					registerMigratedOperationRoutes(r, h, func(c *gin.Context, op string, err error, status int) {
						called = true
						if op != operation.name || err == nil || status != http.StatusBadRequest {
							t.Errorf("binder error: operation=%q status=%d error=%v", op, status, err)
						}
						migratedOperationParamError(c, op, err, status)
					})

					rec := httptest.NewRecorder()
					req := httptest.NewRequest(operation.method, operation.path, bytes.NewBufferString(body.json))
					req.Header.Set("Content-Type", "application/json")
					r.ServeHTTP(rec, req)

					if !called || rec.Code != http.StatusOK {
						t.Fatalf("expected binder failure in business envelope, called=%t status=%d", called, rec.Code)
					}
					resp := decodeWebhookResp(t, rec)
					if resp.Code != response.CodeParamError || resp.Msg != "请求参数错误" {
						t.Fatalf("expected generic parameter error, got %+v", resp)
					}
				})
			}
		})
	}
}

func TestWebhookHandlerSetConfigWritesAuditEvent(t *testing.T) {
	setupWebhookHandlerTestDB(t)

	gin.SetMode(gin.TestMode)
	h := newMigratedOperationAdapter(migratedOperations{WebhookHandler: NewWebhookHandler()})
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(123))
		c.Next()
	})
	registerMigratedOperationRoutes(r, h, migratedOperationParamError)

	reqBody := []byte(`{
		"url":"https://discord.com/api/webhooks/1/token",
		"enabled":true,
		"type":"discord",
		"fleet_template":"test"
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
	h := newMigratedOperationAdapter(migratedOperations{WebhookHandler: NewWebhookHandler()})
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(123))
		c.Next()
	})
	registerMigratedOperationRoutes(r, h, migratedOperationParamError)

	reqBody := []byte(`{
		"url":"https://example.com/webhook",
		"enabled":true,
		"type":"discord",
		"fleet_template":"test"
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
	h := newMigratedOperationAdapter(migratedOperations{WebhookHandler: NewWebhookHandler()})
	r := gin.New()
	registerMigratedOperationRoutes(r, h, migratedOperationParamError)

	reqBody := []byte(`{
		"url":"ftp://discord.com/api/webhooks/1/token",
		"type":"discord"
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
