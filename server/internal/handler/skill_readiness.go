package handler

import (
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SkillReadinessHandler struct {
	svc *service.SkillReadinessService
}

func NewSkillReadinessHandler() *SkillReadinessHandler {
	return &SkillReadinessHandler{svc: service.NewSkillReadinessService()}
}

func readinessError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrSkillReadinessRequest):
		response.Fail(c, response.CodeParamError, "查询参数无效")
	case errors.Is(err, service.ErrSkillReadinessPlan):
		response.Fail(c, response.CodeBizError, "请选择包含技能要求的有效军团方案")
	case errors.Is(err, service.ErrSkillReadinessCharacter):
		response.Fail(c, response.CodeBizError, "角色不存在或不在查询范围内")
	default:
		response.Fail(c, response.CodeBizError, "获取军团技能数据失败")
	}
}

func (h *SkillReadinessHandler) Options(c *gin.Context) {
	result, err := h.svc.Options(c.Request.Context())
	if err != nil {
		readinessError(c, err)
		return
	}
	response.OK(c, result)
}

func bindReadinessRequest(c *gin.Context) (service.SkillReadinessRequest, bool) {
	var req service.SkillReadinessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "查询参数无效")
		return req, false
	}
	return req, true
}

func (h *SkillReadinessHandler) Query(c *gin.Context) {
	req, ok := bindReadinessRequest(c)
	if !ok {
		return
	}
	result, err := h.svc.Query(c.Request.Context(), req)
	if err != nil {
		readinessError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *SkillReadinessHandler) UserCharacters(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("user_id"), 10, strconv.IntSize)
	if err != nil || id == 0 {
		readinessError(c, service.ErrSkillReadinessRequest)
		return
	}
	req, ok := bindReadinessRequest(c)
	if !ok {
		return
	}
	result, err := h.svc.UserCharacters(c.Request.Context(), uint(id), req)
	if err != nil {
		readinessError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *SkillReadinessHandler) CharacterSkills(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("character_id"), 10, 64)
	if err != nil || id <= 0 {
		readinessError(c, service.ErrSkillReadinessRequest)
		return
	}
	req, ok := bindReadinessRequest(c)
	if !ok {
		return
	}
	result, err := h.svc.CharacterSkills(c.Request.Context(), id, req)
	if err != nil {
		readinessError(c, err)
		return
	}
	response.OK(c, result)
}
