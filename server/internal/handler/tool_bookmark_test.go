package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestToolBookmarkHandlerCreateRejectsInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewToolBookmarkHandler()

	r := gin.New()
	r.POST("/api/v1/system/tool-bookmarks", h.AdminCreate)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/tool-bookmarks", strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":400`) {
		t.Fatalf("expected param error response, got: %s", rec.Body.String())
	}
}

func TestToolBookmarkHandlerUpdateRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewToolBookmarkHandler()

	r := gin.New()
	r.PUT("/api/v1/system/tool-bookmarks/:id", h.AdminUpdate)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/system/tool-bookmarks/abc", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":400`) {
		t.Fatalf("expected param error response, got: %s", rec.Body.String())
	}
}
