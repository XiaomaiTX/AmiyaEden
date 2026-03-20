package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SkillPlanHandler 军团技能计划 HTTP 处理器
type SkillPlanHandler struct {
	svc *service.SkillPlanService
}

func NewSkillPlanHandler() *SkillPlanHandler {
	return &SkillPlanHandler{
		svc: service.NewSkillPlanService(),
	}
}

// CreateSkillPlan 创建技能计划
func (h *SkillPlanHandler) CreateSkillPlan(c *gin.Context) {
	var req service.CreateSkillPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}

	result, err := h.svc.CreateSkillPlan(middleware.GetUserID(c), &req, resolveRequestLang(c))
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}

	response.OK(c, result)
}

// ListSkillPlans 获取技能计划列表
func (h *SkillPlanHandler) ListSkillPlans(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))
	keyword := c.Query("keyword")

	records, total, err := h.svc.ListSkillPlans(page, size, keyword)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}

	response.OK(c, gin.H{
		"list":     records,
		"page":     page,
		"pageSize": size,
		"total":    total,
	})
}

// GetSkillPlan 获取技能计划详情
func (h *SkillPlanHandler) GetSkillPlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeParamError, "无效的技能计划 ID")
		return
	}

	result, err := h.svc.GetSkillPlan(uint(id), resolveRequestLang(c))
	if err != nil {
		response.Fail(c, response.CodeNotFound, err.Error())
		return
	}

	response.OK(c, result)
}

// UpdateSkillPlan 更新技能计划
func (h *SkillPlanHandler) UpdateSkillPlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeParamError, "无效的技能计划 ID")
		return
	}

	var req service.UpdateSkillPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}

	result, err := h.svc.UpdateSkillPlan(
		uint(id),
		middleware.GetUserID(c),
		middleware.GetUserRoles(c),
		&req,
		resolveRequestLang(c),
	)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}

	response.OK(c, result)
}

// DeleteSkillPlan 删除技能计划
func (h *SkillPlanHandler) DeleteSkillPlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeParamError, "无效的技能计划 ID")
		return
	}

	err = h.svc.DeleteSkillPlan(uint(id), middleware.GetUserID(c), middleware.GetUserRoles(c))
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}

	response.OK(c, nil)
}

func resolveRequestLang(c *gin.Context) string {
	lang := c.Query("lang")
	if lang == "" {
		lang = c.GetHeader("Accept-Language")
	}
	if lang == "" {
		if cookieLang, err := c.Cookie("language"); err == nil {
			lang = cookieLang
		}
	}
	if lang == "" {
		lang = "zh"
	}
	return lang
}
