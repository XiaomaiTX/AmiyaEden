package router

import (
	"amiya-eden/global"
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestQQOfficialGeneratedRoutesAndRetiredWebhook(t *testing.T) {
	_, user, admin, guest, super := newFuxiHallManageRouteTestRouter(t)
	db := global.DB
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	// Serialize the SQLite test connection against asynchronous operation logs.
	sqlDB.SetMaxOpenConns(1)
	models := append(model.QQOfficialModels(), &model.AuditEvent{}, &model.OperationLog{})
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	previous := service.DefaultQQOfficialRuntime()
	runtime := service.NewQQOfficialRuntime(db, "qq-route-test-deployment-key-only")
	service.SetDefaultQQOfficialRuntime(runtime)
	t.Cleanup(func() { service.SetDefaultQQOfficialRuntime(previous) })
	logCore, logs := observer.New(zap.DebugLevel)
	global.SetLogger(zap.New(logCore))
	r := gin.New()
	r.Use(middleware.OperationLog(), middleware.ResponseWrapper(), middleware.ZapLogger(), middleware.ZapRecovery())
	RegisterRoutes(r, newTaskRouterTestService(t))

	const base = "/api/v1/system/qq-official"
	const appID = "1234"
	const firstSecret = "test-initial-qq-route-secret-012345"
	const newSecret = "test-replaced-qq-route-secret-01234"
	const querySecret = "rejected-settings-query-marker"
	const malformedSecret = "rejected-settings-binder-marker"
	assertNoSecret := func(surface, data string) {
		t.Helper()
		for _, secret := range []string{firstSecret, newSecret, querySecret, malformedSecret} {
			if strings.Contains(data, secret) {
				t.Fatalf("QQ credential leaked in %s", surface)
			}
		}
	}
	requestCount := 0
	t.Cleanup(func() {
		// Wait for asynchronous log writes before the router fixture restores globals.
		deadline := time.Now().Add(2 * time.Second)
		var rows []model.OperationLog
		for {
			if err := db.Find(&rows).Error; err != nil {
				t.Fatal(err)
			}
			if len(rows) == requestCount {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("observed %d/%d operation logs", len(rows), requestCount)
			}
			time.Sleep(10 * time.Millisecond)
		}
		var sawOrdinaryQuery bool
		for _, row := range rows {
			if (row.Path == base+"/settings" || row.Path == base+"/settings/") && row.Query != "" {
				t.Error("settings operation log retained its raw query")
			}
			sawOrdinaryQuery = sawOrdinaryQuery || (row.Path == base+"/tasks" && row.Query == "size=bad")
		}
		if !t.Failed() && !sawOrdinaryQuery {
			t.Error("non-settings operation query was not logged")
		}
		entry, err := json.Marshal(rows)
		if err != nil {
			t.Fatal(err)
		}
		assertNoSecret("operation logs", string(entry))
		entry, err = json.Marshal(logs.All())
		if err != nil {
			t.Fatal(err)
		}
		assertNoSecret("access logs", string(entry))
	})
	call := func(method, path, token, body string) *httptest.ResponseRecorder {
		t.Helper()
		requestCount++
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assertNoSecret("console response", w.Body.String())
		return w
	}
	assertError := func(w *httptest.ResponseRecorder, code int, msg string) {
		t.Helper()
		var body apicontract.QQOfficialError
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != code || int(body.Code) != code || body.Msg != msg {
			t.Fatalf("error = %d %s, want %d %s", w.Code, w.Body, code, msg)
		}
	}
	readStatus := func(w *httptest.ResponseRecorder) apicontract.QQOfficialStatus {
		t.Helper()
		var body apicontract.QQOfficialStatusSuccess
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != 200 || int(body.Code) != response.CodeOK || w.Header().Get("Cache-Control") != "no-store" || strings.Contains(w.Body.String(), `"app_secret"`) {
			t.Fatalf("unexpected QQ status: %d %s", w.Code, w.Body)
		}
		return body.Data
	}

	malformed := `{"app_secret": "` + malformedSecret
	for _, tc := range []struct {
		token  string
		status int
	}{{"", 401}, {user, 403}, {admin, 403}, {guest, 403}} {
		if w := call("GET", base+"/status", tc.token, ""); w.Code != tc.status {
			t.Fatalf("unexpected status %d want %d", w.Code, tc.status)
		}
		if w := call("PUT", base+"/settings", tc.token, malformed); w.Code != tc.status {
			t.Fatalf("auth should precede body decode: %d want %d", w.Code, tc.status)
		}
	}
	if s := readStatus(call("GET", base+"/status", super, "")); s.Configured || s.AppID != nil || s.HasAppSecret || s.Enabled || s.ApprovalWritesEnabled || s.GatewayState != "idle" || s.LastEventAt != nil {
		t.Fatalf("unexpected initial status: %+v", s)
	}
	groups := call("GET", base+"/groups", super, "")
	if groups.Code != 200 || !strings.Contains(groups.Body.String(), `"list":[]`) {
		t.Fatalf("unconfigured groups = %d %s", groups.Code, groups.Body)
	}
	assertError(call("PUT", base+"/settings", super, malformed), 400, "invalid_input")
	assertError(call("PUT", base+"/settings?app_secret="+querySecret, super, `{"app_id":"1234","app_secret":"`+firstSecret+`"}`), 400, "invalid_input")
	// Even an unmatched method with a trailing slash passes through request logs.
	call("GET", base+"/settings/?app_secret="+querySecret, super, "")
	assertError(call("PUT", base+"/settings", super, `{"app_id":"invalid","app_secret":"`+firstSecret+`"}`), 400, "invalid_input")
	assertError(call("PUT", base+"/settings", super, `{"app_id":"1234"}`), 409, "conflict")
	if s := readStatus(call("PUT", base+"/settings", super, `{"app_id":"1234","app_secret":"`+firstSecret+`"}`)); !s.Configured || s.AppID == nil || *s.AppID != appID || !s.HasAppSecret || s.Enabled || s.ApprovalWritesEnabled {
		t.Fatalf("unexpected first setup status: %+v", s)
	}
	if s := readStatus(call("PUT", base+"/settings", super, `{"approval_writes_enabled":true}`)); !s.ApprovalWritesEnabled || !s.HasAppSecret {
		t.Fatalf("approval checkbox did not turn on: %+v", s)
	}
	if s := readStatus(call("PUT", base+"/settings", super, `{"approval_writes_enabled":false}`)); s.ApprovalWritesEnabled || !s.HasAppSecret {
		t.Fatalf("approval checkbox did not turn off: %+v", s)
	}
	if s := readStatus(call("PUT", base+"/settings", super, `{"enabled":true,"approval_writes_enabled":true}`)); !s.Enabled || !s.ApprovalWritesEnabled {
		t.Fatalf("enable settings did not persist: %+v", s)
	}
	assertError(call("GET", base+"/tasks?size=bad", super, ""), 400, "invalid_input")
	verify := `{"group_openid":"ABCDEFGHIJKLMNOPQRSTUV1234567890","idempotency_key":"26f35df9-d323-46be-b864-045db7b69272"}`
	w := call("POST", base+"/groups/verify", super, verify)
	if w.Code != 202 || strings.Contains(w.Body.String(), "ABCDEFGHIJKLMNOP") {
		t.Fatalf("verify %d %s", w.Code, w.Body)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			ID     string `json:"task_id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != response.CodeOK || body.Data.ID == "" || body.Data.Status != "queued" {
		t.Fatalf("receipt %+v %v", body, err)
	}
	if w := call("GET", base+"/tasks/"+body.Data.ID, super, ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"attempts":[]`) {
		t.Fatalf("detail %d %s", w.Code, w.Body)
	}

	if w := call(http.MethodPost, "/api/callbacks/qq-official", "", `{}`); w.Code != http.StatusNotFound {
		t.Fatalf("retired QQ webhook returned %d, want 404: %s", w.Code, w.Body)
	}
	assertError(call("PUT", base+"/settings", super, `{"app_id":"9999"}`), 409, "conflict")
	if s := readStatus(call("PUT", base+"/settings", super, `{"app_secret":"`+newSecret+`"}`)); !s.Configured || s.AppID == nil || *s.AppID != appID || !s.HasAppSecret || s.Enabled || s.ApprovalWritesEnabled {
		t.Fatalf("replacement did not reset write controls: %+v", s)
	}

	// Simulate restoring the database under the wrong deployment key. Only the
	// status may describe the unusable configuration; other paths fail closed.
	service.SetDefaultQQOfficialRuntime(service.NewQQOfficialRuntime(db, "wrong-deployment-key-for-test"))
	if s := readStatus(call("GET", base+"/status", super, "")); s.Configured || s.AppID == nil || *s.AppID != appID || !s.HasAppSecret || s.TokenState != "credentials_unavailable" || s.GatewayState != "idle" {
		t.Fatalf("unexpected unreadable credentials status: %+v", s)
	}
	assertError(call("GET", base+"/groups", super, ""), 503, "unavailable")
	assertError(call("POST", base+"/groups/verify", super, verify), 503, "unavailable")
	service.SetDefaultQQOfficialRuntime(runtime)
	if s := readStatus(call("GET", base+"/status", super, "")); !s.Configured || !s.HasAppSecret {
		t.Fatalf("restored deployment key could not read replaced credentials: %+v", s)
	}
}
