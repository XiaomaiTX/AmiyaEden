package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/pkg/response"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type auditEventHandlerResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func setupAuditEventHandlerTestDB(t *testing.T) {
	t.Helper()

	dsn := fmt.Sprintf("file:audit_event_handler_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AuditEvent{}, &model.AuditExportTask{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
}

func newAuditEventHandlerRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewAuditEventHandler()
	r := gin.New()
	r.POST("/api/v1/system/audit/events", h.AdminList)
	r.POST("/api/v1/system/audit/export", func(c *gin.Context) {
		c.Set("userID", uint(1))
		h.CreateExportTask(c)
	})
	r.GET("/api/v1/system/audit/export/:task_id", h.GetExportTaskStatus)
	return r
}

func decodeAuditEventResp(t *testing.T, recorder *httptest.ResponseRecorder) auditEventHandlerResp {
	t.Helper()
	var out auditEventHandlerResp
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func TestAuditEventHandlerAdminListReturnsPage(t *testing.T) {
	setupAuditEventHandlerTestDB(t)
	now := time.Now()
	if err := global.DB.Create(&model.AuditEvent{
		EventID:      "evt-1",
		OccurredAt:   now,
		Category:     "permission",
		Action:       "set_user_roles",
		ActorUserID:  1,
		TargetUserID: 2,
		Result:       model.AuditResultSuccess,
	}).Error; err != nil {
		t.Fatalf("seed audit event: %v", err)
	}

	r := newAuditEventHandlerRouter()
	body, _ := json.Marshal(map[string]any{
		"current":    1,
		"size":       20,
		"start_date": now.Add(-24 * time.Hour).Format("2006-01-02"),
		"end_date":   now.Add(24 * time.Hour).Format("2006-01-02"),
		"category":   "permission",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/audit/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("http status = %d, want %d", rec.Code, http.StatusOK)
	}
	resp := decodeAuditEventResp(t, rec)
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
	var page struct {
		List []model.AuditEvent `json:"list"`
	}
	if err := json.Unmarshal(resp.Data, &page); err != nil {
		t.Fatalf("decode page data: %v", err)
	}
	if len(page.List) != 1 {
		t.Fatalf("record count = %d, want 1", len(page.List))
	}
}

func TestAuditEventHandlerAdminListRejectsInvalidDate(t *testing.T) {
	setupAuditEventHandlerTestDB(t)
	r := newAuditEventHandlerRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/audit/events", bytes.NewReader([]byte(`{"start_date":"2026/01/01"}`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	resp := decodeAuditEventResp(t, rec)
	if resp.Code != response.CodeParamError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeParamError)
	}
}

func TestAuditEventHandlerCreateAndGetExportTask(t *testing.T) {
	setupAuditEventHandlerTestDB(t)
	if err := global.DB.Create(&model.AuditEvent{
		EventID:    "evt-export-1",
		OccurredAt: time.Now(),
		Category:   "permission",
		Action:     "set_user_roles",
		Result:     model.AuditResultSuccess,
	}).Error; err != nil {
		t.Fatalf("seed audit event: %v", err)
	}

	r := newAuditEventHandlerRouter()
	createReq := `{"format":"csv","filter":{"category":"permission"}}`
	createRec := httptest.NewRecorder()
	createHTTPReq := httptest.NewRequest(http.MethodPost, "/api/v1/system/audit/export", strings.NewReader(createReq))
	createHTTPReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(createRec, createHTTPReq)

	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d, want %d", createRec.Code, http.StatusOK)
	}
	createResp := decodeAuditEventResp(t, createRec)
	if createResp.Code != response.CodeOK {
		t.Fatalf("create code = %d, want %d", createResp.Code, response.CodeOK)
	}

	var created struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(createResp.Data, &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.TaskID == "" {
		t.Fatal("task_id should not be empty")
	}

	var gotStatus string
	for i := 0; i < 20; i++ {
		getRec := httptest.NewRecorder()
		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/system/audit/export/"+created.TaskID, nil)
		r.ServeHTTP(getRec, getReq)
		resp := decodeAuditEventResp(t, getRec)
		if resp.Code != response.CodeOK {
			t.Fatalf("get code = %d, want %d", resp.Code, response.CodeOK)
		}
		var statusResp struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(resp.Data, &statusResp); err != nil {
			t.Fatalf("decode get response: %v", err)
		}
		gotStatus = statusResp.Status
		if gotStatus == model.AuditExportStatusDone || gotStatus == model.AuditExportStatusFailed {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if gotStatus != model.AuditExportStatusDone {
		t.Fatalf("final status = %s, want %s", gotStatus, model.AuditExportStatusDone)
	}
}

func TestAuditEventHandlerGetExportTaskStatusNotFound(t *testing.T) {
	setupAuditEventHandlerTestDB(t)
	r := newAuditEventHandlerRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/audit/export/not-exists", nil)
	r.ServeHTTP(rec, req)

	resp := decodeAuditEventResp(t, rec)
	if resp.Code != response.CodeParamError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeParamError)
	}
}
