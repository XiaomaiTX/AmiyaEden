package middleware

import (
	"amiya-eden/config"
	"amiya-eden/global"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
	if global.Config == nil {
		global.Config = &config.Config{}
	}
}

// ─── CORS ───

func TestCors_DebugModeWithNoCORSOrigins_AllowsAllOrigins(t *testing.T) {
	global.Config.Server.Mode = "debug"
	global.Config.Server.CORSOrigins = nil
	w, _ := serveCORSRequest("https://anything.example.com")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Allow-Origin = %q, want *", got)
	}
}

func TestCors_ReleaseModeWithNoCORSOrigins_DeniesOrigin(t *testing.T) {
	global.Config.Server.Mode = "release"
	global.Config.Server.CORSOrigins = nil
	w, _ := serveCORSRequest("https://attacker.example.com")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty (denied)", got)
	}
}

func TestCors_ExplicitOriginAllowlist_AllowsListedOrigin(t *testing.T) {
	global.Config.Server.Mode = "release"
	global.Config.Server.CORSOrigins = []string{"https://eden.example.com"}
	w, _ := serveCORSRequest("https://eden.example.com")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://eden.example.com" {
		t.Fatalf("Allow-Origin = %q, want https://eden.example.com", got)
	}
}

func TestCors_ExplicitOriginAllowlist_DeniesUnlistedOrigin(t *testing.T) {
	global.Config.Server.Mode = "release"
	global.Config.Server.CORSOrigins = []string{"https://eden.example.com"}
	w, _ := serveCORSRequest("https://evil.example.com")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty (denied)", got)
	}
}

func TestCors_PreflightReturnsNoContent(t *testing.T) {
	global.Config.Server.Mode = "debug"
	global.Config.Server.CORSOrigins = nil
	r := gin.New()
	r.Use(Cors())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func serveCORSRequest(origin string) (*httptest.ResponseRecorder, *http.Request) {
	r := gin.New()
	r.Use(Cors())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", origin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, req
}

// ─── Security Headers ───

func TestSecureHeaders_SetsRequiredHeaders(t *testing.T) {
	r := gin.New()
	r.Use(SecureHeaders())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "0",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for header, want := range checks {
		if got := w.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}
