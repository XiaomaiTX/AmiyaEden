package handler

import (
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

// FuxiAdminHandler 伏羲管理人员名录处理器
type FuxiAdminHandler struct {
	svc *service.FuxiAdminService
}

func NewFuxiAdminHandler() *FuxiAdminHandler {
	return &FuxiAdminHandler{svc: service.NewFuxiAdminService()}
}

// ─── Public ───

// GetDirectory GET /api/v1/fuxi-admins
func (h *FuxiAdminHandler) GetDirectory(c *gin.Context) {
	dir, err := h.svc.GetDirectory()
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, dir)
}

// ─── Admin: Config ───

// GetConfig GET /api/v1/system/fuxi-admins/config
func (h *FuxiAdminHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig()
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, cfg)
}

// UpdateConfig PUT /api/v1/system/fuxi-admins/config
func (h *FuxiAdminHandler) UpdateConfig(c *gin.Context) {
	var req service.FuxiAdminUpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	cfg, err := h.svc.UpdateConfig(&req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, cfg)
}

// ─── Admin: Tiers ───

// ListTiers GET /api/v1/system/fuxi-admins/tiers
func (h *FuxiAdminHandler) ListTiers(c *gin.Context) {
	tiers, err := h.svc.ListTiers()
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, tiers)
}

// CreateTier POST /api/v1/system/fuxi-admins/tiers
func (h *FuxiAdminHandler) CreateTier(c *gin.Context) {
	var req service.FuxiAdminCreateTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	tier, err := h.svc.CreateTier(&req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, tier)
}

// UpdateTier PUT /api/v1/system/fuxi-admins/tiers/:id
func (h *FuxiAdminHandler) UpdateTier(c *gin.Context) {
	id := requireUintID(c, "id", "层级 ID")
	if id == 0 {
		return
	}
	var req service.FuxiAdminUpdateTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	tier, err := h.svc.UpdateTier(id, &req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, tier)
}

// DeleteTier DELETE /api/v1/system/fuxi-admins/tiers/:id
func (h *FuxiAdminHandler) DeleteTier(c *gin.Context) {
	id := requireUintID(c, "id", "层级 ID")
	if id == 0 {
		return
	}
	if err := h.svc.DeleteTier(id); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}

// ─── Admin: Admins ───

// CreateAdmin POST /api/v1/system/fuxi-admins
func (h *FuxiAdminHandler) CreateAdmin(c *gin.Context) {
	var req service.FuxiAdminCreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	admin, err := h.svc.CreateAdmin(&req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, admin)
}

// UpdateAdmin PUT /api/v1/system/fuxi-admins/:id
func (h *FuxiAdminHandler) UpdateAdmin(c *gin.Context) {
	id := requireUintID(c, "id", "管理员 ID")
	if id == 0 {
		return
	}
	var req service.FuxiAdminUpdateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	admin, err := h.svc.UpdateAdmin(id, &req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, admin)
}

// DeleteAdmin DELETE /api/v1/system/fuxi-admins/:id
func (h *FuxiAdminHandler) DeleteAdmin(c *gin.Context) {
	id := requireUintID(c, "id", "管理员 ID")
	if id == 0 {
		return
	}
	if err := h.svc.DeleteAdmin(id); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}
