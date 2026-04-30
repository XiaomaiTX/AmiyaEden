package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/jobs"
	"amiya-eden/pkg/response"
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
)

type CorporationStructureHandler struct {
	svc *service.CorporationStructureService
}

func NewCorporationStructureHandler() *CorporationStructureHandler {
	return &CorporationStructureHandler{
		svc: service.NewCorporationStructureService(),
	}
}

func (h *CorporationStructureHandler) GetSettings(c *gin.Context) {
	result, err := h.svc.GetSettings(c.Request.Context())
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *CorporationStructureHandler) UpdateAuthorizations(c *gin.Context) {
	var req service.CorporationStructureAuthorizationUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	req.OperatorUserID = middleware.GetUserID(c)
	if err := h.svc.UpdateAuthorizations(c.Request.Context(), req); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CorporationStructureHandler) ListStructures(c *gin.Context) {
	var req service.CorporationStructureListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	result, err := h.svc.ListStructures(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *CorporationStructureHandler) GetFilterOptions(c *gin.Context) {
	var req service.CorporationStructureFilterOptionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	result, err := h.svc.GetFilterOptions(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *CorporationStructureHandler) RunTask(c *gin.Context) {
	var req service.CorporationStructureRunTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	characterID, err := h.svc.ResolveRefreshAuthorizationCharacter(c.Request.Context(), req.CorporationID)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}

	queue := jobs.GetESIQueue()
	if queue == nil {
		response.Fail(c, response.CodeBizError, "刷新队列未初始化")
		return
	}

	taskName := fmt.Sprintf("dashboard_run_corporation_structures_%d", req.CorporationID)
	if ok := global.EnsureBackgroundTaskManager().Go(taskName, func(ctx context.Context) error {
		return queue.RunTask(ctx, "corporation_structures", characterID)
	}); !ok {
		response.Fail(c, response.CodeBizError, "服务正在关闭，任务未启动")
		return
	}

	response.OK(c, service.CorporationStructureRunTaskResponse{
		CorporationID: req.CorporationID,
		Queued:        true,
		Running:       false,
		Message:       "已触发后台 ESI 刷新任务",
	})
}
