package handler

import (
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

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
	if err := h.svc.UpdateAuthorizations(c.Request.Context(), req); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CorporationStructureHandler) ListStructures(c *gin.Context) {
	var req service.CorporationStructureListRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.svc.ListStructures(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *CorporationStructureHandler) RefreshStructures(c *gin.Context) {
	var req service.CorporationStructureRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	result, err := h.svc.RefreshStructures(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, result)
}
