package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

// WebhookHandler Webhook 配置 HTTP 处理器
type WebhookHandler struct {
	svc *service.WebhookService
}

func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{svc: service.NewWebhookService()}
}

// GetConfig GET /system/webhook/config
func (h *WebhookHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig()
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, cfg)
}

// SetConfig PUT /system/webhook/config
func (h *WebhookHandler) SetConfig(c *gin.Context) {
	var req service.WebhookConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetConfigByOperator(&req, middleware.GetUserID(c)); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}

// TestWebhook POST /system/webhook/test
func (h *WebhookHandler) TestWebhook(c *gin.Context) {
	var req struct {
		URL     string `json:"url"`
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	cfg := &service.WebhookConfig{
		URL:  req.URL,
		Type: req.Type,
	}
	if err := h.svc.SendTest(cfg, req.Content); err != nil {
		response.Fail(c, response.CodeBizError, "发送失败: "+err.Error())
		return
	}
	response.OK(c, nil)
}
