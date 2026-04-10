package bootstrap

import (
	"amiya-eden/global"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/router"
	"amiya-eden/internal/service"

	"github.com/gin-gonic/gin"
)

// InitRouter 初始化并返回 Gin 路由引擎
func InitRouter(taskSvc *service.TaskService) *gin.Engine {
	gin.SetMode(global.Config.Server.Mode)

	r := gin.New()

	// 全局中间件。
	// 进入 handler 前: RequestID(注入请求 ID) → SecureHeaders(预写安全头) → OperationLog(开始计时) →
	// ResponseWrapper(接管响应) → ZapLogger(开始计时) → ZapRecovery(注册 panic 恢复) →
	// Cors(写 CORS 头，OPTIONS 可在此提前返回) → handler
	// c.Next() 返回后: ZapRecovery(仅 panic 时处理) → ZapLogger(记录日志) →
	// ResponseWrapper(统一响应并写 biz_code) → OperationLog(读取 biz_code 入库)
	// RequestID / SecureHeaders / Cors 没有额外的 after 阶段逻辑。
	r.Use(
		middleware.RequestID(),
		middleware.SecureHeaders(),
		middleware.OperationLog(),
		middleware.ResponseWrapper(),
		middleware.ZapLogger(),
		middleware.ZapRecovery(),
		middleware.Cors(),
	)

	// 注册业务路由
	router.RegisterRoutes(r, taskSvc)

	global.Logger.Info("路由注册完成")
	return r
}
