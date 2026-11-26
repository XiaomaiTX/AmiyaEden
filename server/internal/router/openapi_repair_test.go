package router

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/jobs"
	"amiya-eden/pkg/eve/esi"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

func postOpenAPIJSON(t *testing.T, router *gin.Engine, method, path, token, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if w.Code != http.StatusOK {
		t.Fatalf("%s %s: expected HTTP 200, got %d: %s", method, path, w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("%s %s: decode response: %v", method, path, err)
	}
	return result.Code, result.Msg
}

func TestGeneratedWebhookRoutesPreserveBindingReason(t *testing.T) {
	router, _, _, _, token := newFuxiHallManageRouteTestRouter(t)
	for _, route := range []struct{ method, path string }{
		{http.MethodPut, "/api/v1/system/webhook/config"},
		{http.MethodPost, "/api/v1/system/webhook/test"},
	} {
		t.Run(route.path, func(t *testing.T) {
			code, msg := postOpenAPIJSON(t, router, route.method, route.path, token, `{"url":`)
			if code != response.CodeParamError || msg != "请求参数错误: unexpected EOF" {
				t.Fatalf("expected original parameter error with reason, got code %d, %q", code, msg)
			}
		})
	}
}

func TestGeneratedSaveFittingRequiresEquipment(t *testing.T) {
	router, _, _, _, token := newFuxiHallManageRouteTestRouter(t)
	if err := global.DB.AutoMigrate(&model.EveCharacter{}); err != nil {
		t.Fatal(err)
	}
	const path = "/api/v1/info/fittings/save"
	const body = `{"character_id":1234,"ship_type_id":5678,"fitting_id":42,"name":"Fit"`
	for _, suffix := range []string{`}`, `,"items":null}`, `,"items":[]}`} {
		t.Run(suffix, func(t *testing.T) {
			code, msg := postOpenAPIJSON(t, router, http.MethodPost, path, token, body+suffix)
			if code != response.CodeParamError || msg != "装配参数无效" {
				t.Fatalf("expected rejection before reaching the service, got code %d, %q", code, msg)
			}
		})
	}
	// The character doesn't exist. A business error shows that the non-empty
	// request passed binding and reached the service without calling ESI.
	code, msg := postOpenAPIJSON(t, router, http.MethodPost, path, token,
		body+`,"items":[{"type_id":123,"quantity":1,"flag":""}]}`)
	if code != response.CodeBizError {
		t.Fatalf("expected to reach the service, got code %d, %q", code, msg)
	}
}

func TestGeneratedESIStatusesPreservesPaginationErrorsWhenQueueAvailable(t *testing.T) {
	router, _, _, _, token := newFuxiHallManageRouteTestRouter(t)
	oldQueue := jobs.GetESIQueue()
	// Invalid pagination fails before any queue methods are called.
	jobs.SetTestESIQueue(&esi.Queue{})
	t.Cleanup(func() { jobs.SetTestESIQueue(oldQueue) })

	for _, parameter := range []string{"current", "size"} {
		t.Run(parameter, func(t *testing.T) {
			path := "/api/v1/tasks/esi/statuses?" + parameter + "=abc"
			code, msg := postOpenAPIJSON(t, router, http.MethodGet, path, token, "")
			want := "invalid " + parameter + " query parameter: expected integer"
			if code != response.CodeParamError || msg != want {
				t.Fatalf("expected parameter error %q, got code %d, %q", want, code, msg)
			}
		})
	}
}

func TestGeneratedTicketListPreservesQueryErrors(t *testing.T) {
	router, _, _, _, token := newFuxiHallManageRouteTestRouter(t)
	for _, tc := range []struct{ query, message string }{
		{"user_id=abc", "无效的 user_id"},
		{"user_id=4294967296", "无效的 user_id"},
		{"category_id=abc", "无效的 category_id"},
		{"category_id=0", "无效的 category_id"},
		{"category_id=abc&user_id=abc", "无效的 user_id"},
		{"current=abc&user_id=abc", "invalid current query parameter: expected integer"},
		{"size=abc", "invalid size query parameter: expected integer"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			code, msg := postOpenAPIJSON(t, router, http.MethodGet, "/api/v1/system/ticket/tickets?"+tc.query, token, "")
			if code != response.CodeParamError || msg != tc.message {
				t.Fatalf("expected parameter error %q, got code %d, %q", tc.message, code, msg)
			}
		})
	}
}

func TestGeneratedGalaxyListTrimsSystemFilter(t *testing.T) {
	router, _, _, _, token := newFuxiHallManageRouteTestRouter(t)
	if err := global.DB.AutoMigrate(&model.GalaxyRegistryEntry{}); err != nil {
		t.Fatal(err)
	}
	if err := global.DB.Create(&[]model.GalaxyRegistryEntry{{SystemConfigID: 7}, {SystemConfigID: 8}}).Error; err != nil {
		t.Fatal(err)
	}
	const path = "/api/v1/dashboard/galaxy-registry/admin/entries"
	for _, tc := range []struct {
		query string
		total int
	}{
		{"", 2},
		{"?system_config_id=%20%09", 2},
		{"?system_config_id=7", 1},
		{"?system_config_id=%207%09", 1},
	} {
		t.Run(tc.query, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path+tc.query, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			var result struct {
				Code int
				Data struct{ Total int }
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != http.StatusOK || result.Code != response.CodeOK || result.Data.Total != tc.total {
				t.Fatalf("expected success with total=%d, got HTTP %d: %s", tc.total, w.Code, w.Body.String())
			}
		})
	}
	code, msg := postOpenAPIJSON(t, router, http.MethodGet, path+"?system_config_id=%20abc%20", token, "")
	if code != response.CodeParamError || msg != "无效的 system_config_id" {
		t.Fatalf("expected field-specific error, got code %d, %q", code, msg)
	}
}

func TestGeneratedJSONBindsWithoutDefaultBodyLimitOrPreRead(t *testing.T) {
	router, _, _, _, token := newFuxiHallManageRouteTestRouter(t)
	const path = "/api/v1/sde/types"
	padding := strings.Repeat(" ", (10<<20)+1)
	// A valid JSON object beyond the former default limit must reach the handler.
	code, msg := postOpenAPIJSON(t, router, http.MethodPost, path, token, padding+`{"type_ids":[]}`)
	if code != response.CodeParamError || msg != "type_ids 不能为空" {
		t.Fatalf("expected handler validation for a large JSON body, got code %d, %q", code, msg)
	}

	// A small first object must not cause the decoder to drain a large tail.
	source := strings.NewReader(`{"type_ids":[]}` + padding)
	req := httptest.NewRequest(http.MethodPost, path, source)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "type_ids 不能为空") || source.Len() < len(padding)/2 {
		t.Fatalf("expected handler validation without draining the body, got HTTP %d: %s (%d unread bytes)", w.Code, w.Body.String(), source.Len())
	}
}

func TestGeneratedMultipartKeepsLegacyBodyLimit(t *testing.T) {
	router, token, _, _, _ := newFuxiHallManageRouteTestRouter(t)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "large.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(make([]byte, 3<<20)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/image", &body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var result struct {
		Code int
		Msg  string
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != http.StatusOK || result.Code != response.CodeParamError || !strings.Contains(result.Msg, "request body too large") {
		t.Fatalf("expected legacy multipart size error, got HTTP %d: %s", w.Code, w.Body.String())
	}
}
