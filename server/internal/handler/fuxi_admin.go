package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

type fuxiAdminService interface {
	GetDirectory() (*service.FuxiAdminDirectoryResponse, error)
	GetManageDirectory() (*service.FuxiAdminManageDirectoryResponse, error)
	GetManageAdmin(id uint) (*service.FuxiAdminManageAdmin, error)
	GetConfig() (*model.FuxiAdminConfig, error)
	UpdateConfig(req *service.FuxiAdminUpdateConfigRequest) (*model.FuxiAdminConfig, error)
	ListTiers() ([]model.FuxiAdminTier, error)
	CreateTier(req *service.FuxiAdminCreateTierRequest) (*model.FuxiAdminTier, error)
	UpdateTier(id uint, req *service.FuxiAdminUpdateTierRequest) (*model.FuxiAdminTier, error)
	DeleteTier(id uint) error
	CreateAdmin(req *service.FuxiAdminCreateAdminRequest) (*model.FuxiAdmin, error)
	UpdateAdmin(id uint, req *service.FuxiAdminUpdateAdminRequest) (*model.FuxiAdmin, error)
	DeleteAdmin(id uint) error
}

// FuxiAdminHandler 伏羲管理人员名录处理器
type FuxiAdminHandler struct {
	svc fuxiAdminService
}

func NewFuxiAdminHandler() *FuxiAdminHandler {
	return &FuxiAdminHandler{svc: service.NewFuxiAdminService()}
}

func respondFuxiAdminError(c *gin.Context, err error, fallback string) {
	message := fallback
	if service.IsUserVisibleError(err) {
		message = err.Error()
	}
	response.Fail(c, response.CodeBizError, message)
}

func buildFallbackManageAdmin(admin *model.FuxiAdmin) *service.FuxiAdminManageAdmin {
	return &service.FuxiAdminManageAdmin{
		FuxiAdmin:             *admin,
		WelfareDeliveryOffset: admin.WelfareDeliveryOffset,
		FleetLedCount:         0,
		WelfareDeliveryCount:  int64(admin.WelfareDeliveryOffset),
	}
}

// ─── Public ───

// GetDirectory GET /api/v1/fuxi-admins
func (h *FuxiAdminHandler) GetDirectory(c *gin.Context) {
	dir, err := h.svc.GetDirectory()
	if err != nil {
		respondFuxiAdminError(c, err, "获取伏羲管理名录失败")
		return
	}
	response.OK(c, dir)
}

// GetManageDirectory GET /api/v1/system/fuxi-admins/manage-directory
func (h *FuxiAdminHandler) GetManageDirectory(c *gin.Context) {
	dir, err := h.svc.GetManageDirectory()
	if err != nil {
		respondFuxiAdminError(c, err, "获取伏羲管理名录管理视图失败")
		return
	}
	response.OK(c, dir)
}

// ─── Admin: Config ───

// GetConfig GET /api/v1/system/fuxi-admins/config
func (h *FuxiAdminHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig()
	if err != nil {
		respondFuxiAdminError(c, err, "获取伏羲管理名录配置失败")
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
		respondFuxiAdminError(c, err, "更新伏羲管理名录配置失败")
		return
	}
	response.OK(c, cfg)
}

// ─── Admin: Tiers ───

// ListTiers GET /api/v1/system/fuxi-admins/tiers
func (h *FuxiAdminHandler) ListTiers(c *gin.Context) {
	tiers, err := h.svc.ListTiers()
	if err != nil {
		respondFuxiAdminError(c, err, "获取伏羲管理层级失败")
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
		respondFuxiAdminError(c, err, "创建伏羲管理层级失败")
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
		respondFuxiAdminError(c, err, "更新伏羲管理层级失败")
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
		respondFuxiAdminError(c, err, "删除伏羲管理层级失败")
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
		respondFuxiAdminError(c, err, "创建伏羲管理员失败")
		return
	}
	manageAdmin, err := h.svc.GetManageAdmin(admin.ID)
	if err != nil {
		response.OK(c, buildFallbackManageAdmin(admin))
		return
	}
	response.OK(c, manageAdmin)
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
	if req.WelfareDeliveryOffset != nil && !model.IsSuperAdmin(middleware.GetUserRoles(c)) {
		response.Fail(c, response.CodeForbidden, "仅超级管理员可修改福利发放次数偏移")
		return
	}
	admin, err := h.svc.UpdateAdmin(id, &req)
	if err != nil {
		respondFuxiAdminError(c, err, "更新伏羲管理员失败")
		return
	}
	manageAdmin, err := h.svc.GetManageAdmin(admin.ID)
	if err != nil {
		response.OK(c, buildFallbackManageAdmin(admin))
		return
	}
	response.OK(c, manageAdmin)
}

// DeleteAdmin DELETE /api/v1/system/fuxi-admins/:id
func (h *FuxiAdminHandler) DeleteAdmin(c *gin.Context) {
	id := requireUintID(c, "id", "管理员 ID")
	if id == 0 {
		return
	}
	if err := h.svc.DeleteAdmin(id); err != nil {
		respondFuxiAdminError(c, err, "删除伏羲管理员失败")
		return
	}
	response.OK(c, nil)
}
