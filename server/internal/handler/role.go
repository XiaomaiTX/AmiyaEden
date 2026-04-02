package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	svc *service.RoleService
}

func NewRoleHandler() *RoleHandler {
	return &RoleHandler{svc: service.NewRoleService()}
}

// ListRoleDefinitions 返回系统职权定义列表（纯内存，不查库）
func (h *RoleHandler) ListRoleDefinitions(c *gin.Context) {
	response.OK(c, h.svc.ListRoleDefinitions())
}

// ─── 用户职权管理 ───

func (h *RoleHandler) GetUserRoles(c *gin.Context) {
	id := requireUintID(c, "id", "用户ID")
	if id == 0 {
		return
	}
	roles, err := h.svc.GetUserRoles(id)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, roles)
}

type setUserRolesRequest struct {
	RoleCodes []string `json:"role_codes"`
}

func (h *RoleHandler) SetUserRoles(c *gin.Context) {
	userID := requireUintID(c, "id", "用户ID")
	if userID == 0 {
		return
	}
	var req setUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	operatorID := middleware.GetUserID(c)
	operatorRoles := middleware.GetUserRoles(c)
	if err := h.svc.SetUserRoles(c.Request.Context(), operatorID, operatorRoles, userID, req.RoleCodes); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}
