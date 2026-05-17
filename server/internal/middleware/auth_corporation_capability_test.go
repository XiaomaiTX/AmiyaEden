package middleware

import (
	"amiya-eden/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireCorpCapabilityAllowsSuperAdmin(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("roles", []string{model.RoleSuperAdmin})
		c.Next()
	})
	r.GET("/test", RequireCorpCapability(model.CorpCapabilitySRPManage), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestRequireCorpCapabilityRejectsMissingCapability(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("roles", []string{model.RoleAdmin})
		c.Set("corpCapabilities", []string{model.CorpCapabilitySRPUser})
		c.Next()
	})
	r.GET("/test", RequireCorpCapability(model.CorpCapabilitySRPManage), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRequireCorpCapabilityAllowsFullAccess(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("roles", []string{model.RoleAdmin})
		c.Set("corpFullAccess", true)
		c.Next()
	})
	r.GET("/test", RequireCorpCapability(model.CorpCapabilitySRPManage), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
