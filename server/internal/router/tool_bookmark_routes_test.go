package router

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestToolBookmarkInfoRouteRequiresLoginUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	guestRouter := newToolBookmarkInfoPermissionRouter([]string{model.RoleGuest})
	assertToolBookmarkRouteStatus(t, guestRouter, http.MethodGet, "/info/tool-bookmarks", http.StatusForbidden)

	userRouter := newToolBookmarkInfoPermissionRouter([]string{model.RoleUser})
	assertToolBookmarkRouteStatus(t, userRouter, http.MethodGet, "/info/tool-bookmarks", http.StatusNoContent)
}

func TestToolBookmarkSystemRoutesRequireAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRouter := newToolBookmarkAdminPermissionRouter([]string{model.RoleUser})
	assertToolBookmarkRouteStatus(t, userRouter, http.MethodGet, "/system/tool-bookmarks", http.StatusForbidden)
	assertToolBookmarkRouteStatus(t, userRouter, http.MethodPost, "/system/tool-bookmarks", http.StatusForbidden)

	adminRouter := newToolBookmarkAdminPermissionRouter([]string{model.RoleAdmin})
	assertToolBookmarkRouteStatus(t, adminRouter, http.MethodGet, "/system/tool-bookmarks", http.StatusNoContent)
	assertToolBookmarkRouteStatus(t, adminRouter, http.MethodPost, "/system/tool-bookmarks", http.StatusNoContent)
	assertToolBookmarkRouteStatus(t, adminRouter, http.MethodPut, "/system/tool-bookmarks/1", http.StatusNoContent)
	assertToolBookmarkRouteStatus(t, adminRouter, http.MethodDelete, "/system/tool-bookmarks/1", http.StatusNoContent)
}

func newToolBookmarkInfoPermissionRouter(roles []string) *gin.Engine {
	r := gin.New()
	injectRoles := func(c *gin.Context) {
		c.Set("roles", roles)
		c.Next()
	}
	info := r.Group("/info", injectRoles, middleware.RequireLoginUser())
	info.GET("/tool-bookmarks", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func newToolBookmarkAdminPermissionRouter(roles []string) *gin.Engine {
	r := gin.New()
	injectRoles := func(c *gin.Context) {
		c.Set("roles", roles)
		c.Next()
	}
	admin := r.Group("/system", injectRoles, middleware.RequireRole(model.RoleAdmin))
	bookmarks := admin.Group("/tool-bookmarks")
	bookmarks.GET("", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	bookmarks.POST("", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	bookmarks.PUT("/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	bookmarks.DELETE("/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func assertToolBookmarkRouteStatus(t *testing.T, router *gin.Engine, method, path string, want int) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != want {
		t.Fatalf("%s %s = %d, want %d", method, path, rec.Code, want)
	}
}
