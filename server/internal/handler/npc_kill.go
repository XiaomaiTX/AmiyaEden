package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
)

// NpcKillHandler NPC 刷怪报表处理器
type NpcKillHandler struct {
	svc *service.NpcKillService
}

func NewNpcKillHandler() *NpcKillHandler {
	return &NpcKillHandler{
		svc: service.NewNpcKillService(),
	}
}

// GetNpcKills POST /info/npc-kills
// 获取当前用户指定人物的刷怪报表
func (h *NpcKillHandler) GetNpcKills(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req service.NpcKillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误: "+err.Error())
		return
	}
	if err := validateNpcKillRange(req.StartDate, req.EndDate, middleware.GetCorpRuleInt(c, service.CorpRuleNpcKillsMaxRangeDays, 365)); err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}

	result, err := h.svc.GetNpcKills(userID, &req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, result)
}

// GetAllNpcKills POST /info/npc-kills/all
// 获取当前用户名下所有人物的汇总刷怪报表
func (h *NpcKillHandler) GetAllNpcKills(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req service.NpcKillAllRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误: "+err.Error())
		return
	}
	if err := validateNpcKillRange(req.StartDate, req.EndDate, middleware.GetCorpRuleInt(c, service.CorpRuleNpcKillsMaxRangeDays, 365)); err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}

	result, err := h.svc.GetAllNpcKills(userID, &req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, result)
}

// GetCorpNpcKills POST /corp/npc-kills
// 获取公司内所有成员的刷怪报表（管理员）
func (h *NpcKillHandler) GetCorpNpcKills(c *gin.Context) {
	if !middleware.GetCorpRuleBool(c, service.CorpRuleNpcKillsAllowCorpAggregate, true) {
		response.Fail(c, response.CodeForbidden, "当前军团策略禁止公司级刷怪报表聚合")
		return
	}

	var req service.NpcKillCorpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误: "+err.Error())
		return
	}
	if err := validateNpcKillRange(req.StartDate, req.EndDate, middleware.GetCorpRuleInt(c, service.CorpRuleNpcKillsMaxRangeDays, 365)); err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}

	result, err := h.svc.GetCorpNpcKills(&req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, result)
}

func validateNpcKillRange(startDate string, endDate string, maxDays int) error {
	if maxDays <= 0 || startDate == "" || endDate == "" {
		return nil
	}
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil
	}
	if end.Before(start) {
		return nil
	}
	if int(end.Sub(start).Hours()/24)+1 > maxDays {
		return errors.New("查询时间范围超过军团策略限制")
	}
	return nil
}
