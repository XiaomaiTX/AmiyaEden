package router

import (
	"amiya-eden/config"
	"amiya-eden/global"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"amiya-eden/pkg/jwt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTaskRouterTestService(t *testing.T) *service.TaskService {
	t.Helper()

	dsn := "file:task_router_test_" + time.Now().Format("150405.000000000") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.TaskSchedule{}, &model.TaskExecution{}); err != nil {
		t.Fatalf("auto migrate task router models: %v", err)
	}

	registry := taskregistry.New()
	repo := repository.NewTaskRepositoryWithDB(db)
	return service.NewTaskService(registry, repo)
}

func TestRegisterRoutesIncludesTaskManagerEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldConfig := global.Config
	oldLogger := global.Logger
	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Logger = zap.NewNop()
	t.Cleanup(func() {
		global.Config = oldConfig
		global.Logger = oldLogger
	})

	r := gin.New()

	RegisterRoutes(r, newTaskRouterTestService(t))

	routes := r.Routes()
	want := map[string]string{
		http.MethodGet:  "/api/v1/tasks",
		http.MethodPost: "/api/v1/tasks/:name/run",
		http.MethodPut:  "/api/v1/tasks/:name/schedule",
	}

	for method, path := range want {
		if !routeExists(routes, method, path) {
			t.Fatalf("expected route %s %s to be registered", method, path)
		}
	}

	if !routeExists(routes, http.MethodGet, "/api/v1/tasks/history") {
		t.Fatal("expected route GET /api/v1/tasks/history to be registered")
	}
	if !routeExists(routes, http.MethodGet, "/api/v1/tasks/esi/tasks") {
		t.Fatal("expected route GET /api/v1/tasks/esi/tasks to be registered")
	}
	if !routeExists(routes, http.MethodGet, "/api/v1/tasks/esi/statuses") {
		t.Fatal("expected route GET /api/v1/tasks/esi/statuses to be registered")
	}
	if routeExists(routes, http.MethodGet, "/api/v1/esi/refresh/tasks") {
		t.Fatal("expected legacy GET /api/v1/esi/refresh/tasks route to be removed")
	}
	if routeExists(routes, http.MethodPost, "/api/v1/system/newbro/attribution/sync") {
		t.Fatal("expected legacy POST /api/v1/system/newbro/attribution/sync route to be removed")
	}
	if routeExists(routes, http.MethodPost, "/api/v1/system/newbro/reward/process") {
		t.Fatal("expected legacy POST /api/v1/system/newbro/reward/process route to be removed")
	}
}

func routeExists(routes gin.RoutesInfo, method, path string) bool {
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}

func TestSrpManageRolesIncludeAdmin(t *testing.T) {
	if !containsRoleCode(srpPriceManageRoles, model.RoleAdmin) {
		t.Fatal("expected srp price manage roles to include admin")
	}
	if !containsRoleCode(srpPriceManageRoles, model.RoleSeniorFC) {
		t.Fatal("expected srp price manage roles to include senior_fc")
	}
	if containsRoleCode(srpPriceManageRoles, model.RoleSRP) {
		t.Fatalf("expected srp price manage roles to exclude srp, got %v", srpPriceManageRoles)
	}
	if !containsRoleCode(srpManageRoles, model.RoleAdmin) {
		t.Fatal("expected srp manage roles to include admin")
	}
	if !containsRoleCode(srpPayoutRoles, model.RoleAdmin) {
		t.Fatal("expected srp payout roles to include admin")
	}
}

func TestSrpPriceWriteRequiresAdminAndSeniorFCWhileReadAllowsSrp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRouter := newSrpPricePermissionTestRouter([]string{model.RoleUser})
	assertRouteStatus(t, userRouter, http.MethodGet, "/srp/prices", http.StatusNoContent)
	assertRouteStatus(t, userRouter, http.MethodPost, "/srp/prices", http.StatusForbidden)
	assertRouteStatus(t, userRouter, http.MethodDelete, "/srp/prices/1", http.StatusForbidden)

	srpRouter := newSrpPricePermissionTestRouter([]string{model.RoleSRP})
	assertRouteStatus(t, srpRouter, http.MethodGet, "/srp/prices", http.StatusNoContent)
	assertRouteStatus(t, srpRouter, http.MethodPost, "/srp/prices", http.StatusForbidden)
	assertRouteStatus(t, srpRouter, http.MethodDelete, "/srp/prices/1", http.StatusForbidden)

	adminRouter := newSrpPricePermissionTestRouter([]string{model.RoleAdmin})
	assertRouteStatus(t, adminRouter, http.MethodGet, "/srp/prices", http.StatusNoContent)
	assertRouteStatus(t, adminRouter, http.MethodPost, "/srp/prices", http.StatusNoContent)
	assertRouteStatus(t, adminRouter, http.MethodDelete, "/srp/prices/1", http.StatusNoContent)

	seniorFCRouter := newSrpPricePermissionTestRouter([]string{model.RoleSeniorFC})
	assertRouteStatus(t, seniorFCRouter, http.MethodGet, "/srp/prices", http.StatusNoContent)
	assertRouteStatus(t, seniorFCRouter, http.MethodPost, "/srp/prices", http.StatusNoContent)
	assertRouteStatus(t, seniorFCRouter, http.MethodDelete, "/srp/prices/1", http.StatusNoContent)

	superAdminRouter := newSrpPricePermissionTestRouter([]string{model.RoleSuperAdmin})
	assertRouteStatus(t, superAdminRouter, http.MethodGet, "/srp/prices", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodPost, "/srp/prices", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodDelete, "/srp/prices/1", http.StatusNoContent)
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

func TestFuxiAdminDirectoryReadUsesLoggedInRouteGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, userToken, guestToken := newFuxiAdminDirectoryRouteTestRouter(t)

	assertRouteStatus(t, router, http.MethodGet, "/api/v1/fuxi-admins", http.StatusUnauthorized)
	assertRouteStatus(t, router, http.MethodGet, "/api/v1/fuxi-admins?token="+guestToken, http.StatusForbidden)
	assertRouteStatus(t, router, http.MethodGet, "/api/v1/fuxi-admins?token="+userToken, http.StatusOK)
}

func TestSystemWebhookRequiresSuperAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if containsRoleCode(systemWebhookManageRoles, model.RoleAdmin) {
		t.Fatalf("expected systemWebhookManageRoles to exclude admin, got %v", systemWebhookManageRoles)
	}
	if !containsRoleCode(systemWebhookManageRoles, model.RoleSuperAdmin) {
		t.Fatalf("expected systemWebhookManageRoles to include super_admin, got %v", systemWebhookManageRoles)
	}

	adminRouter := newSystemWebhookPermissionTestRouter([]string{model.RoleAdmin})
	assertRouteStatus(t, adminRouter, http.MethodGet, "/system/webhook/config", http.StatusForbidden)
	assertRouteStatus(t, adminRouter, http.MethodPut, "/system/webhook/config", http.StatusForbidden)
	assertRouteStatus(t, adminRouter, http.MethodPost, "/system/webhook/test", http.StatusForbidden)

	superAdminRouter := newSystemWebhookPermissionTestRouter([]string{model.RoleSuperAdmin})
	assertRouteStatus(t, superAdminRouter, http.MethodGet, "/system/webhook/config", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodPut, "/system/webhook/config", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodPost, "/system/webhook/test", http.StatusNoContent)
}

func TestSystemBasicConfigRequiresSuperAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if containsRoleCode(systemBasicConfigManageRoles, model.RoleAdmin) {
		t.Fatalf("expected systemBasicConfigManageRoles to exclude admin, got %v", systemBasicConfigManageRoles)
	}
	if !containsRoleCode(systemBasicConfigManageRoles, model.RoleSuperAdmin) {
		t.Fatalf("expected systemBasicConfigManageRoles to include super_admin, got %v", systemBasicConfigManageRoles)
	}

	adminRouter := newSystemBasicConfigPermissionTestRouter([]string{model.RoleAdmin})
	assertRouteStatus(t, adminRouter, http.MethodGet, "/system/basic-config", http.StatusForbidden)
	assertRouteStatus(t, adminRouter, http.MethodGet, "/system/sde-config", http.StatusForbidden)
	assertRouteStatus(t, adminRouter, http.MethodPut, "/system/sde-config", http.StatusForbidden)
	assertRouteStatus(t, adminRouter, http.MethodGet, "/system/basic-config/allow-corporations", http.StatusForbidden)
	assertRouteStatus(t, adminRouter, http.MethodPut, "/system/basic-config/allow-corporations", http.StatusForbidden)
	assertRouteStatus(t, adminRouter, http.MethodGet, "/system/basic-config/character-esi-restriction", http.StatusForbidden)
	assertRouteStatus(t, adminRouter, http.MethodPut, "/system/basic-config/character-esi-restriction", http.StatusForbidden)

	superAdminRouter := newSystemBasicConfigPermissionTestRouter([]string{model.RoleSuperAdmin})
	assertRouteStatus(t, superAdminRouter, http.MethodGet, "/system/basic-config", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodGet, "/system/sde-config", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodPut, "/system/sde-config", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodGet, "/system/basic-config/allow-corporations", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodPut, "/system/basic-config/allow-corporations", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodGet, "/system/basic-config/character-esi-restriction", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodPut, "/system/basic-config/character-esi-restriction", http.StatusNoContent)
}

func TestAutoRoleRequiresSuperAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if containsRoleCode(autoRoleManageRoles, model.RoleAdmin) {
		t.Fatalf("expected autoRoleManageRoles to exclude admin, got %v", autoRoleManageRoles)
	}
	if !containsRoleCode(autoRoleManageRoles, model.RoleSuperAdmin) {
		t.Fatalf("expected autoRoleManageRoles to include super_admin, got %v", autoRoleManageRoles)
	}

	adminRouter := newAutoRolePermissionTestRouter([]string{model.RoleAdmin})
	assertRouteStatus(t, adminRouter, http.MethodGet, "/system/auto-role/esi-roles", http.StatusForbidden)
	assertRouteStatus(t, adminRouter, http.MethodGet, "/system/auto-role/esi-role-mappings", http.StatusForbidden)
	assertRouteStatus(t, adminRouter, http.MethodPost, "/system/auto-role/sync", http.StatusForbidden)

	superAdminRouter := newAutoRolePermissionTestRouter([]string{model.RoleSuperAdmin})
	assertRouteStatus(t, superAdminRouter, http.MethodGet, "/system/auto-role/esi-roles", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodGet, "/system/auto-role/esi-role-mappings", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodPost, "/system/auto-role/sync", http.StatusNoContent)
}

func TestShopOrderRoutesRequireAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if containsRoleCode(shopOrderManageRoles, model.RoleWelfare) {
		t.Fatalf("expected shopOrderManageRoles to exclude welfare, got %v", shopOrderManageRoles)
	}
	if !containsRoleCode(shopOrderManageRoles, model.RoleAdmin) {
		t.Fatalf("expected shopOrderManageRoles to include admin, got %v", shopOrderManageRoles)
	}

	welfareRouter := newShopOrderPermissionTestRouter([]string{model.RoleWelfare})
	assertRouteStatus(t, welfareRouter, http.MethodPost, "/system/shop/order/list", http.StatusForbidden)
	assertRouteStatus(t, welfareRouter, http.MethodPost, "/system/shop/order/deliver", http.StatusForbidden)
	assertRouteStatus(t, welfareRouter, http.MethodPost, "/system/shop/order/reject", http.StatusForbidden)

	adminRouter := newShopOrderPermissionTestRouter([]string{model.RoleAdmin})
	assertRouteStatus(t, adminRouter, http.MethodPost, "/system/shop/order/list", http.StatusNoContent)
	assertRouteStatus(t, adminRouter, http.MethodPost, "/system/shop/order/deliver", http.StatusNoContent)
	assertRouteStatus(t, adminRouter, http.MethodPost, "/system/shop/order/reject", http.StatusNoContent)

	superAdminRouter := newShopOrderPermissionTestRouter([]string{model.RoleSuperAdmin})
	assertRouteStatus(t, superAdminRouter, http.MethodPost, "/system/shop/order/list", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodPost, "/system/shop/order/deliver", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodPost, "/system/shop/order/reject", http.StatusNoContent)
}

func TestWelfareApprovalRoutesAllowWelfareWhileDeleteStaysAdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if !containsRoleCode(welfareApprovalRoles, model.RoleWelfare) {
		t.Fatalf("expected welfareApprovalRoles to include welfare, got %v", welfareApprovalRoles)
	}
	if !containsRoleCode(welfareApprovalRoles, model.RoleAdmin) {
		t.Fatalf("expected welfareApprovalRoles to include admin, got %v", welfareApprovalRoles)
	}

	welfareRouter := newWelfareApprovalPermissionTestRouter([]string{model.RoleWelfare})
	assertRouteStatus(t, welfareRouter, http.MethodPost, "/system/welfare/applications", http.StatusNoContent)
	assertRouteStatus(t, welfareRouter, http.MethodPost, "/system/welfare/review", http.StatusNoContent)
	assertRouteStatus(
		t,
		welfareRouter,
		http.MethodPost,
		"/system/welfare/applications/delete",
		http.StatusForbidden,
	)

	adminRouter := newWelfareApprovalPermissionTestRouter([]string{model.RoleAdmin})
	assertRouteStatus(t, adminRouter, http.MethodPost, "/system/welfare/applications", http.StatusNoContent)
	assertRouteStatus(t, adminRouter, http.MethodPost, "/system/welfare/review", http.StatusNoContent)
	assertRouteStatus(
		t,
		adminRouter,
		http.MethodPost,
		"/system/welfare/applications/delete",
		http.StatusNoContent,
	)

	superAdminRouter := newWelfareApprovalPermissionTestRouter([]string{model.RoleSuperAdmin})
	assertRouteStatus(t, superAdminRouter, http.MethodPost, "/system/welfare/applications", http.StatusNoContent)
	assertRouteStatus(t, superAdminRouter, http.MethodPost, "/system/welfare/review", http.StatusNoContent)
	assertRouteStatus(
		t,
		superAdminRouter,
		http.MethodPost,
		"/system/welfare/applications/delete",
		http.StatusNoContent,
	)
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

func newFuxiAdminDirectoryRouteTestRouter(t *testing.T) (*gin.Engine, string, string) {
	t.Helper()

	db := newFuxiAdminDirectoryRouteTestDB(t)

	oldConfig := global.Config
	oldLogger := global.Logger
	oldDB := global.DB
	oldRedis := global.Redis

	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Logger = zap.NewNop()
	global.DB = db
	global.Redis = redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:0",
		DialTimeout:  10 * time.Millisecond,
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: 10 * time.Millisecond,
		PoolTimeout:  10 * time.Millisecond,
		MaxRetries:   0,
	})

	t.Cleanup(func() {
		global.Config = oldConfig
		global.Logger = oldLogger
		global.DB = oldDB
		if global.Redis != nil {
			_ = global.Redis.Close()
		}
		global.Redis = oldRedis
	})

	if err := db.Create(&model.UserRole{UserID: 1, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("create user role: %v", err)
	}

	jwt.Init("fuxi-admin-router-test-secret")
	userToken, err := jwt.GenerateToken(1, 1001, model.RoleUser, 1)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	guestToken, err := jwt.GenerateToken(2, 1002, model.RoleGuest, 1)
	if err != nil {
		t.Fatalf("generate guest token: %v", err)
	}

	r := gin.New()
	RegisterRoutes(r, newTaskRouterTestService(t))
	return r, userToken, guestToken
}

func newFuxiAdminDirectoryRouteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:fuxi_admin_router_test_" + time.Now().Format("150405.000000000") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.UserRole{}, &model.FuxiAdminConfig{}, &model.FuxiAdminTier{}, &model.FuxiAdmin{}); err != nil {
		t.Fatalf("auto migrate fuxi admin router models: %v", err)
	}
	return db
}

func newSrpPricePermissionTestRouter(roles []string) *gin.Engine {
	r := gin.New()
	injectRoles := func(c *gin.Context) {
		c.Set("roles", roles)
		c.Next()
	}

	read := r.Group("/srp", injectRoles, middleware.RequireLoginUser())
	read.GET("/prices", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	write := r.Group("/srp", injectRoles, middleware.RequireLoginUser())
	write.POST("/prices", middleware.RequireRole(srpPriceManageRoles...), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	write.DELETE("/prices/:id", middleware.RequireRole(srpPriceManageRoles...), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	return r
}

func newSystemWebhookPermissionTestRouter(roles []string) *gin.Engine {
	r := gin.New()
	injectRoles := func(c *gin.Context) {
		c.Set("roles", roles)
		c.Next()
	}

	webhook := r.Group("/system/webhook", injectRoles, middleware.RequireRole(systemWebhookManageRoles...))
	webhook.GET("/config", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	webhook.PUT("/config", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	webhook.POST("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	return r
}

func newShopOrderPermissionTestRouter(roles []string) *gin.Engine {
	r := gin.New()
	injectRoles := func(c *gin.Context) {
		c.Set("roles", roles)
		c.Next()
	}

	shopOrder := r.Group("/system/shop/order", injectRoles, middleware.RequireRole(shopOrderManageRoles...))
	shopOrder.POST("/list", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	shopOrder.POST("/deliver", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	shopOrder.POST("/reject", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	return r
}

func newWelfareApprovalPermissionTestRouter(roles []string) *gin.Engine {
	r := gin.New()
	injectRoles := func(c *gin.Context) {
		c.Set("roles", roles)
		c.Next()
	}

	approval := r.Group(
		"/system/welfare",
		injectRoles,
		middleware.RequireRole(welfareApprovalRoles...),
	)
	approval.POST("/applications", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	approval.POST("/review", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	deleteOnly := r.Group("/system/welfare", injectRoles, middleware.RequireRole(model.RoleAdmin))
	deleteOnly.POST("/applications/delete", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	return r
}

func newSystemBasicConfigPermissionTestRouter(roles []string) *gin.Engine {
	r := gin.New()
	injectRoles := func(c *gin.Context) {
		c.Set("roles", roles)
		c.Next()
	}

	adminConfig := r.Group(
		"/system",
		injectRoles,
		middleware.RequireRole(model.RoleAdmin),
		middleware.RequireRole(systemBasicConfigManageRoles...),
	)
	adminConfig.GET("/sde-config", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	adminConfig.PUT("/sde-config", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	basicConfig := r.Group("/system/basic-config", injectRoles, middleware.RequireRole(systemBasicConfigManageRoles...))
	basicConfig.GET("", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	basicConfig.GET("/allow-corporations", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	basicConfig.PUT("/allow-corporations", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	basicConfig.GET("/character-esi-restriction", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	basicConfig.PUT("/character-esi-restriction", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	return r
}

func newAutoRolePermissionTestRouter(roles []string) *gin.Engine {
	r := gin.New()
	injectRoles := func(c *gin.Context) {
		c.Set("roles", roles)
		c.Next()
	}

	admin := r.Group("/system", injectRoles, middleware.RequireRole(model.RoleAdmin))
	autoRole := admin.Group("/auto-role", middleware.RequireRole(autoRoleManageRoles...))
	autoRole.GET("/esi-roles", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	autoRole.GET("/esi-role-mappings", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	autoRole.POST("/sync", func(c *gin.Context) { c.Status(http.StatusNoContent) })

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
