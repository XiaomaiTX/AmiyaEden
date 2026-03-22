package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

// WelfareHandler 福利 HTTP 处理器
type WelfareHandler struct {
	svc *service.WelfareService
}

func NewWelfareHandler() *WelfareHandler {
	return &WelfareHandler{svc: service.NewWelfareService()}
}

// ─────────────────────────────────────────────
//  管理员端（全部 POST）
// ─────────────────────────────────────────────

// adminWelfareCreateRequest 创建福利请求
type adminWelfareCreateRequest struct {
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description"`
	DistMode         string `json:"dist_mode" binding:"required,oneof=per_user per_character"`
	RequireSkillPlan bool   `json:"require_skill_plan"`
	SkillPlanIDs     []uint `json:"skill_plan_ids"`
	Status           int8   `json:"status"`
}

// AdminCreateWelfare POST /system/welfare/add
func (h *WelfareHandler) AdminCreateWelfare(c *gin.Context) {
	var req adminWelfareCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}

	w := &model.Welfare{
		Name:             req.Name,
		Description:      req.Description,
		DistMode:         req.DistMode,
		RequireSkillPlan: req.RequireSkillPlan,
		SkillPlanIDs:     req.SkillPlanIDs,
		Status:           req.Status,
		CreatedBy:        middleware.GetUserID(c),
	}

	if err := h.svc.AdminCreateWelfare(w); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, w)
}

// adminWelfareUpdateRequest 更新福利请求
type adminWelfareUpdateRequest struct {
	ID uint `json:"id" binding:"required"`
	service.AdminUpdateWelfareRequest
}

// AdminUpdateWelfare POST /system/welfare/edit
func (h *WelfareHandler) AdminUpdateWelfare(c *gin.Context) {
	var req adminWelfareUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}

	w, err := h.svc.AdminUpdateWelfare(req.ID, &req.AdminUpdateWelfareRequest)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, w)
}

// adminWelfareDeleteRequest 删除福利请求
type adminWelfareDeleteRequest struct {
	ID uint `json:"id" binding:"required"`
}

// AdminDeleteWelfare POST /system/welfare/delete
func (h *WelfareHandler) AdminDeleteWelfare(c *gin.Context) {
	var req adminWelfareDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}

	if err := h.svc.AdminDeleteWelfare(req.ID); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}

// adminWelfareListRequest 福利列表请求
type adminWelfareListRequest struct {
	Current int    `json:"current"`
	Size    int    `json:"size"`
	Status  *int8  `json:"status"`
	Name    string `json:"name"`
}

// AdminListWelfares POST /system/welfare/list
func (h *WelfareHandler) AdminListWelfares(c *gin.Context) {
	var req adminWelfareListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Current = 1
		req.Size = 20
	}

	filter := repository.WelfareFilter{
		Status: req.Status,
		Name:   req.Name,
	}

	list, total, err := h.svc.AdminListWelfares(req.Current, req.Size, filter)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OKWithPage(c, list, total, req.Current, req.Size)
}
