package router

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSrpManageRolesIncludeAdmin(t *testing.T) {
	if !containsRoleCode(srpManageRoles, model.RoleAdmin) {
		t.Fatal("expected srp manage roles to include admin")
	}
	if !containsRoleCode(srpPayoutRoles, model.RoleAdmin) {
		t.Fatal("expected srp payout roles to include admin")
	}
}

func TestSkillPlanReadAllowsLoggedInUserAndWriteStillRequiresManager(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ordinaryRouter := newSkillPlanPermissionTestRouter([]string{"user"})
	assertRouteStatus(t, ordinaryRouter, http.MethodGet, "/skill-planning/skill-plans", http.StatusNoContent)
	assertRouteStatus(t, ordinaryRouter, http.MethodGet, "/skill-planning/skill-plans/1", http.StatusNoContent)
	assertRouteStatus(t, ordinaryRouter, http.MethodPost, "/skill-planning/skill-plans", http.StatusForbidden)
	assertRouteStatus(t, ordinaryRouter, http.MethodPut, "/skill-planning/skill-plans/reorder", http.StatusForbidden)
	assertRouteStatus(t, ordinaryRouter, http.MethodPut, "/skill-planning/skill-plans/1", http.StatusForbidden)
	assertRouteStatus(t, ordinaryRouter, http.MethodDelete, "/skill-planning/skill-plans/1", http.StatusForbidden)

	managerRouter := newSkillPlanPermissionTestRouter([]string{model.RoleAdmin})
	assertRouteStatus(t, managerRouter, http.MethodPost, "/skill-planning/skill-plans", http.StatusNoContent)
	assertRouteStatus(t, managerRouter, http.MethodPut, "/skill-planning/skill-plans/reorder", http.StatusNoContent)
	assertRouteStatus(t, managerRouter, http.MethodPut, "/skill-planning/skill-plans/1", http.StatusNoContent)
	assertRouteStatus(t, managerRouter, http.MethodDelete, "/skill-planning/skill-plans/1", http.StatusNoContent)
}

func newSkillPlanPermissionTestRouter(roles []string) *gin.Engine {
	r := gin.New()
	injectRoles := func(c *gin.Context) {
		c.Set("roles", roles)
		c.Next()
	}

	read := r.Group("/skill-planning/skill-plans", injectRoles, middleware.RequireLoginUser())
	read.GET("", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	read.GET("/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	write := r.Group("/skill-planning/skill-plans", injectRoles, middleware.RequireRole(skillPlanManageRoles...))
	write.POST("", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	write.PUT("/reorder", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	write.PUT("/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	write.DELETE("/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	return r
}

func assertRouteStatus(t *testing.T, router *gin.Engine, method, path string, want int) {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != want {
		t.Fatalf("%s %s = %d, want %d", method, path, rec.Code, want)
	}
}

func containsRoleCode(codes []string, target string) bool {
	for _, code := range codes {
		if code == target {
			return true
		}
	}
	return false
}
