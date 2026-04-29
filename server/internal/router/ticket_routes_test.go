package router

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTicketMemberRoutesRequireLoginUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	guestRouter := newTicketMemberPermissionTestRouter([]string{model.RoleGuest})
	assertTicketRouteStatus(t, guestRouter, http.MethodGet, "/ticket/tickets/me", http.StatusForbidden)

	userRouter := newTicketMemberPermissionTestRouter([]string{model.RoleUser})
	assertTicketRouteStatus(t, userRouter, http.MethodGet, "/ticket/tickets/me", http.StatusNoContent)

	adminRouter := newTicketMemberPermissionTestRouter([]string{model.RoleAdmin})
	assertTicketRouteStatus(t, adminRouter, http.MethodGet, "/ticket/tickets/me", http.StatusNoContent)
}

func TestTicketAdminRoutesRequireAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRouter := newTicketAdminPermissionTestRouter([]string{model.RoleUser})
	assertTicketRouteStatus(t, userRouter, http.MethodGet, "/system/ticket/tickets", http.StatusForbidden)

	adminRouter := newTicketAdminPermissionTestRouter([]string{model.RoleAdmin})
	assertTicketRouteStatus(t, adminRouter, http.MethodGet, "/system/ticket/tickets", http.StatusNoContent)

	superAdminRouter := newTicketAdminPermissionTestRouter([]string{model.RoleSuperAdmin})
	assertTicketRouteStatus(t, superAdminRouter, http.MethodGet, "/system/ticket/tickets", http.StatusNoContent)
}

func newTicketMemberPermissionTestRouter(roles []string) *gin.Engine {
	r := gin.New()
	injectRoles := func(c *gin.Context) {
		c.Set("roles", roles)
		c.Next()
	}

	member := r.Group("/ticket", injectRoles, middleware.RequireLoginUser())
	member.GET("/tickets/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	member.POST("/tickets", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	return r
}

func newTicketAdminPermissionTestRouter(roles []string) *gin.Engine {
	r := gin.New()
	injectRoles := func(c *gin.Context) {
		c.Set("roles", roles)
		c.Next()
	}

	admin := r.Group("/system", injectRoles, middleware.RequireRole(model.RoleAdmin))
	ticket := admin.Group("/ticket")
	ticket.GET("/tickets", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	ticket.GET("/statistics", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	return r
}

func assertTicketRouteStatus(t *testing.T, router *gin.Engine, method, path string, want int) {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != want {
		t.Fatalf("%s %s = %d, want %d", method, path, rec.Code, want)
	}
}
