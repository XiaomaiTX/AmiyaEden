package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type QQGovernanceAdminHandler struct{ svc *service.QQGovernanceService }

func NewQQGovernanceAdminHandler() *QQGovernanceAdminHandler {
	return &QQGovernanceAdminHandler{svc: service.DefaultQQGovernanceService()}
}
func (h *QQGovernanceAdminHandler) ListPolicies(c *gin.Context) {
	rows, err := h.svc.ListPolicies(c.Request.Context())
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, rows)
}
func (h *QQGovernanceAdminHandler) SavePolicy(c *gin.Context) {
	var req service.QQGovernancePolicyInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	if c.Param("group_id") != "" {
		group, ok := parseQQGovernanceInt64(c, "group_id")
		if !ok {
			return
		}
		req.GroupID = group
	}
	item, err := h.svc.SavePolicy(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, item)
}
func (h *QQGovernanceAdminHandler) DeletePolicy(c *gin.Context) {
	group, ok := parseQQGovernanceInt64(c, "group_id")
	if !ok {
		return
	}
	if err := h.svc.DeletePolicy(group, middleware.GetUserID(c)); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}
func (h *QQGovernanceAdminHandler) ListMembers(c *gin.Context) {
	rows, err := h.svc.ListMemberStates(repository.QQGovernanceMemberFilter{GroupID: queryInt64(c, "group_id"), QQ: queryInt64(c, "qq"), Status: c.Query("status"), QQGovernancePage: queryGovernancePage(c)})
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, rows)
}
func (h *QQGovernanceAdminHandler) ListReviews(c *gin.Context) {
	rows, err := h.svc.ListReviews(repository.QQGovernanceReviewFilter{GroupID: queryInt64(c, "group_id"), QQ: queryInt64(c, "qq"), Decision: c.Query("decision"), QQGovernancePage: queryGovernancePage(c)})
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, rows)
}
func (h *QQGovernanceAdminHandler) ListTasks(c *gin.Context) {
	rows, err := h.svc.ListActionTasks(repository.QQGovernanceTaskFilter{GroupID: queryInt64(c, "group_id"), QQ: queryInt64(c, "qq"), Status: c.Query("status"), ActionType: c.Query("action_type"), QQGovernancePage: queryGovernancePage(c)})
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, rows)
}
func (h *QQGovernanceAdminHandler) RetryTask(c *gin.Context) {
	id := requireUintID(c, "id", "任务ID")
	if id == 0 {
		return
	}
	if err := h.svc.RetryTask(id, middleware.GetUserID(c)); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}
func (h *QQGovernanceAdminHandler) ListAlerts(c *gin.Context) {
	rows, err := h.svc.ListAlerts(repository.QQGovernanceAlertFilter{Status: c.Query("status"), QQGovernancePage: queryGovernancePage(c)})
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, rows)
}
func (h *QQGovernanceAdminHandler) AcknowledgeAlert(c *gin.Context) {
	id := requireUintID(c, "id", "告警ID")
	if id == 0 {
		return
	}
	if err := h.svc.AcknowledgeAlert(id, middleware.GetUserID(c)); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}
func (h *QQGovernanceAdminHandler) Metrics(c *gin.Context) {
	data, err := h.svc.Metrics()
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, data)
}
func (h *QQGovernanceAdminHandler) ListGroups(c *gin.Context) {
	rows, err := h.svc.ListGroupStatuses()
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, rows)
}
func (h *QQGovernanceAdminHandler) SearchCorporations(c *gin.Context) {
	rows, err := h.svc.SearchCorporations(c.Request.Context(), c.Query("query"))
	if err != nil {
		response.Fail(c, response.CodeBizError, "搜索军团失败")
		return
	}
	response.OK(c, rows)
}
func (h *QQGovernanceAdminHandler) GetSettings(c *gin.Context) {
	response.OK(c, h.svc.GovernanceSettings())
}
func (h *QQGovernanceAdminHandler) UpdateSettings(c *gin.Context) {
	var req service.QQGovernanceSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	if err := h.svc.UpdateGovernanceSettings(req, middleware.GetUserID(c)); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, h.svc.GovernanceSettings())
}
func (h *QQGovernanceAdminHandler) TriggerReconcile(c *gin.Context) {
	group, ok := parseQQGovernanceInt64(c, "group_id")
	if !ok {
		return
	}
	if err := h.svc.TriggerReconcile(c.Request.Context(), group, middleware.GetUserID(c)); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}
func (h *QQGovernanceAdminHandler) ResetRisk(c *gin.Context) {
	if err := h.svc.ResetRiskControl(middleware.GetUserID(c)); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}
func (h *QQGovernanceAdminHandler) Connection(c *gin.Context) {
	metrics, err := h.svc.Metrics()
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, gin.H{"connected": metrics.Connected, "risk_level": metrics.RiskLevel})
}
func parseQQGovernanceInt64(c *gin.Context, key string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(key), 10, 64)
	if err != nil || value <= 0 {
		response.Fail(c, response.CodeParamError, "无效的 "+key)
		return 0, false
	}
	return value, true
}
func queryInt64(c *gin.Context, key string) int64 {
	value, _ := strconv.ParseInt(c.Query(key), 10, 64)
	return value
}
func queryGovernancePage(c *gin.Context) repository.QQGovernancePage {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	return repository.QQGovernancePage{Page: page, PageSize: size}
}
