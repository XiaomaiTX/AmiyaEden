package handler

import (
	"context"

	"amiya-eden/internal/apicontract"
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

func (h *WebhookHandler) GetWebhookConfig(_ context.Context, _ apicontract.GetWebhookConfigRequestObject) (apicontract.GetWebhookConfigResponseObject, error) {
	var out apicontract.GetWebhookConfig200JSONResponse
	cfg, err := h.svc.GetConfig()
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	data := apicontract.WebhookConfig{
		URL: cfg.URL, Enabled: cfg.Enabled, Type: cfg.Type, FleetTemplate: cfg.FleetTemplate,
	}
	return &out, out.FromWebhookConfigSuccess(apicontract.WebhookConfigSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *WebhookHandler) SetWebhookConfig(ctx context.Context, req apicontract.SetWebhookConfigRequestObject) (apicontract.SetWebhookConfigResponseObject, error) {
	var out apicontract.SetWebhookConfig200JSONResponse
	if req.Body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.SetWebhookConfig401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	cfg := service.WebhookConfig{
		URL: req.Body.URL, Enabled: req.Body.Enabled, Type: req.Body.Type, FleetTemplate: req.Body.FleetTemplate,
	}
	if err := h.svc.SetConfigByOperator(&cfg, middleware.GetUserID(c)); err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *WebhookHandler) TestWebhook(_ context.Context, req apicontract.TestWebhookRequestObject) (apicontract.TestWebhookResponseObject, error) {
	var out apicontract.TestWebhook200JSONResponse
	if req.Body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	if err := h.svc.SendTest(&service.WebhookConfig{URL: req.Body.URL, Type: req.Body.Type}, req.Body.Content); err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, "发送失败: "+err.Error()))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}
