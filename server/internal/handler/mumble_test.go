package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMumbleInternalEndpointsRejectJWTAndMalformedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := global.DB
	db, err := gorm.Open(sqlite.Open("file:mumble_handler_auth?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&model.SystemConfig{Key: model.SysConfigMumbleServiceToken, Value: "mumble-service-test-token"}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}
	global.DB = db
	defer func() { global.DB = previousDB }()

	router := gin.New()
	h := NewMumbleHandler()
	internal := router.Group("/api/internal/mumble/v1", middleware.RequireMumbleService())
	internal.POST("/authenticate", h.Authenticate)

	jwtRequest := httptest.NewRequest(http.MethodPost, "/api/internal/mumble/v1/authenticate", strings.NewReader(`{}`))
	jwtRequest.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiJ9.seat.jwt")
	jwtRecorder := httptest.NewRecorder()
	router.ServeHTTP(jwtRecorder, jwtRequest)
	if jwtRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("Seat JWT status=%d, want %d", jwtRecorder.Code, http.StatusUnauthorized)
	}

	malformed := httptest.NewRequest(http.MethodPost, "/api/internal/mumble/v1/authenticate", strings.NewReader(`{"username":"pilot"}`))
	malformed.Header.Set("Authorization", "Bearer mumble-service-test-token")
	malformed.Header.Set("Content-Type", "application/json")
	malformedRecorder := httptest.NewRecorder()
	router.ServeHTTP(malformedRecorder, malformed)
	if malformedRecorder.Code != http.StatusOK {
		t.Fatalf("malformed response status=%d, want %d", malformedRecorder.Code, http.StatusOK)
	}
	if !strings.Contains(malformedRecorder.Body.String(), "请求参数错误") {
		t.Fatalf("unexpected malformed response: %s", malformedRecorder.Body.String())
	}

	withoutInstanceID := httptest.NewRequest(http.MethodPost, "/api/internal/mumble/v1/authenticate", strings.NewReader(`{"username":"pilot","password":"password"}`))
	withoutInstanceID.Header.Set("Authorization", "Bearer mumble-service-test-token")
	withoutInstanceID.Header.Set("Content-Type", "application/json")
	withoutInstanceIDRecorder := httptest.NewRecorder()
	router.ServeHTTP(withoutInstanceIDRecorder, withoutInstanceID)
	if withoutInstanceIDRecorder.Code != http.StatusOK {
		t.Fatalf("request without server instance ID status=%d, want %d", withoutInstanceIDRecorder.Code, http.StatusOK)
	}
	if !strings.Contains(withoutInstanceIDRecorder.Body.String(), `"decision":"deny"`) {
		t.Fatalf("request without server instance ID must return a decision: %s", withoutInstanceIDRecorder.Body.String())
	}
}
