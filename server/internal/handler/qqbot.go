package handler

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"github.com/gin-gonic/gin"
	"time"
)

type QQBotHandler struct{ svc *service.QQBotService }

func NewQQBotHandler() *QQBotHandler           { return &QQBotHandler{svc: service.DefaultQQBotService()} }
func (h *QQBotHandler) Reverse(c *gin.Context) { h.svc.ServeReverse(c.Writer, c.Request) }
func (h *QQBotHandler) Snapshot(c *gin.Context) {
	v, e := h.svc.Snapshot(c.Request.Context())
	if e != nil {
		response.Fail(c, response.CodeBizError, "qqbot unavailable")
		return
	}
	response.OK(c, v)
}
func (h *QQBotHandler) GetSetting(c *gin.Context) {
	v, e := h.svc.Setting(c.Request.Context())
	if e != nil {
		response.Fail(c, response.CodeBizError, "qqbot unavailable")
		return
	}
	response.OK(c, v)
}
func (h *QQBotHandler) SetSetting(c *gin.Context) {
	var v struct {
		Enabled    bool    `json:"enabled"`
		BotID      int64   `json:"bot_id"`
		CIDRs      string  `json:"cidrs"`
		IntervalMS int     `json:"interval_ms"`
		Token      *string `json:"token"`
	}
	if c.ShouldBindJSON(&v) != nil {
		response.Fail(c, response.CodeParamError, "invalid configuration")
		return
	}
	e := h.svc.SetSetting(c.Request.Context(), model.QQBotSetting{Enabled: v.Enabled, BotID: v.BotID, CIDRs: v.CIDRs, IntervalMS: v.IntervalMS}, v.Token)
	if e != nil {
		response.Fail(c, response.CodeBizError, "invalid configuration")
		return
	}
	response.OK(c, nil)
}
func (h *QQBotHandler) Cancel(c *gin.Context) {
	if e := h.svc.Cancel(c.Request.Context(), c.Param("id")); e != nil {
		response.Fail(c, response.CodeBizError, "cancel failed")
		return
	}
	response.OK(c, nil)
}
func (h *QQBotHandler) Retry(c *gin.Context) {
	v, e := h.svc.RetryOperation(c.Request.Context(), c.Param("id"))
	if e != nil {
		response.Fail(c, response.CodeBizError, "retry not allowed")
		return
	}
	response.OK(c, v)
}
func (h *QQBotHandler) RetryDelivery(c *gin.Context) {
	if e := h.svc.RetryDelivery(c.Request.Context(), c.Param("id")); e != nil {
		response.Fail(c, response.CodeBizError, "retry not allowed")
		return
	}
	response.OK(c, nil)
}
func (h *QQBotHandler) Pause(c *gin.Context) {
	var v struct {
		Paused bool `json:"paused"`
	}
	if c.ShouldBindJSON(&v) != nil {
		response.Fail(c, response.CodeParamError, "invalid pause")
		return
	}
	if e := h.svc.Pause(c.Request.Context(), c.Param("class"), v.Paused); e != nil {
		response.Fail(c, response.CodeBizError, "pause failed")
		return
	}
	response.OK(c, nil)
}

func (h *QQBotHandler) GetOperation(c *gin.Context) {
	v, e := h.svc.GetOperation(c.Request.Context(), c.Param("id"))
	if e != nil {
		response.Fail(c, response.CodeBizError, "operation unavailable")
		return
	}
	response.OK(c, v)
}

func (h *QQBotHandler) NotificationChannels(c *gin.Context) {
	v, e := h.svc.Notifications().Channels(c.Request.Context())
	if e != nil {
		response.Fail(c, response.CodeBizError, "notifications unavailable")
		return
	}
	response.OK(c, v)
}
func (h *QQBotHandler) ConfigureNotifications(c *gin.Context) {
	var v struct {
		Enabled  bool    `json:"enabled"`
		GroupIDs []int64 `json:"group_ids"`
	}
	if c.ShouldBindJSON(&v) != nil {
		response.Fail(c, response.CodeParamError, "invalid configuration")
		return
	}
	if e := h.svc.Notifications().Configure(c.Request.Context(), c.Param("channel"), v.Enabled, v.GroupIDs); e != nil {
		response.Fail(c, response.CodeBizError, "invalid configuration")
		return
	}
	response.OK(c, nil)
}
func (h *QQBotHandler) NotificationTargets(c *gin.Context) {
	v, e := h.svc.Notifications().Targets(c.Request.Context())
	if e != nil {
		response.Fail(c, response.CodeBizError, "notifications unavailable")
		return
	}
	response.OK(c, v)
}
func (h *QQBotHandler) TestNotification(c *gin.Context) {
	var v struct {
		Key       string    `json:"key"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if c.ShouldBindJSON(&v) != nil {
		response.Fail(c, response.CodeParamError, "invalid test")
		return
	}
	targets, e := h.svc.Notifications().Test(c.Request.Context(), v.Key, v.ExpiresAt)
	if e != nil {
		response.Fail(c, response.CodeBizError, "test unavailable")
		return
	}
	response.OK(c, targets)
}

func (h *QQBotHandler) GroupBusiness(c *gin.Context) {
	v, e := h.svc.Groups().Audit(c.Request.Context())
	if e != nil {
		response.Fail(c, response.CodeBizError, "group business unavailable")
		return
	}
	response.OK(c, v)
}
func (h *QQBotHandler) ConfigureGroupBusiness(c *gin.Context) {
	v := model.QQGroupRule{MismatchConfirmations: 3, ObservationHours: 24}
	if c.ShouldBindJSON(&v) != nil {
		response.Fail(c, response.CodeParamError, "invalid rule")
		return
	}
	if e := h.svc.Groups().Configure(c.Request.Context(), v); e != nil {
		response.Fail(c, response.CodeBizError, "invalid rule")
		return
	}
	response.OK(c, nil)
}
