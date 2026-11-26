package router

import (
	"amiya-eden/internal/handler"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有业务路由
func RegisterRoutes(r *gin.Engine, taskSvc *service.TaskService) {
	mumbleInternalH := handler.NewMumbleHandler()
	mumbleInternal := r.Group("/api/internal/mumble/v1", middleware.RequireMumbleService())
	{
		mumbleInternal.POST("/authenticate", mumbleInternalH.Authenticate)
		mumbleInternal.POST("/identities/resolve", mumbleInternalH.ResolveIdentities)
	}

	// ─── 上传文件静态目录 ───
	r.Static("/uploads", "./uploads")

	registerOpenAPIRoutes(r, taskSvc)
}
