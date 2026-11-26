package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"github.com/gin-gonic/gin"
)

// PAPExchangeHandler PAP 兑换汇率 HTTP 处理器
type PAPExchangeHandler struct {
	svc *service.PAPExchangeService
}

func NewPAPExchangeHandler() *PAPExchangeHandler {
	return &PAPExchangeHandler{svc: service.NewPAPExchangeService()}
}

func (h *PAPExchangeHandler) getPapExchangeRates() (apicontract.PapExchangeConfig, error) {
	config, err := h.svc.GetConfig()
	if err != nil {
		return apicontract.PapExchangeConfig{}, err
	}
	return papExchangeDTO(config), nil
}

func (h *PAPExchangeHandler) setPapExchangeRates(operatorID uint, req apicontract.PapExchangeInput) (apicontract.PapExchangeConfig, error) {
	rates := make([]service.SetRateRequest, len(req.Rates))
	for i, rate := range req.Rates {
		rates[i] = service.SetRateRequest{PapType: rate.PapType, DisplayName: rate.DisplayName, Rate: rate.Rate}
	}
	input := service.UpdateConfigRequest{
		Rates: rates, MulticharFullRewardCount: req.MulticharFullRewardCount,
		MulticharReducedRewardCount: req.MulticharReducedRewardCount,
		MulticharReducedRewardPct:   req.MulticharReducedRewardPct,
	}
	config, err := h.svc.UpdateConfigByOperator(&input, operatorID)
	if err != nil {
		return apicontract.PapExchangeConfig{}, err
	}
	return papExchangeDTO(config), nil
}

func papExchangeDTO(config *service.PAPExchangeConfigResponse) apicontract.PapExchangeConfig {
	rates := make([]apicontract.PapExchangeRate, len(config.Rates))
	for i, rate := range config.Rates {
		rates[i] = apicontract.PapExchangeRate{
			PapType: rate.PapType, DisplayName: rate.DisplayName, Rate: rate.Rate, UpdatedAt: rate.UpdatedAt,
		}
	}
	return apicontract.PapExchangeConfig{
		Rates: rates, MulticharFullRewardCount: config.MulticharFullRewardCount,
		MulticharReducedRewardCount: config.MulticharReducedRewardCount,
		MulticharReducedRewardPct:   config.MulticharReducedRewardPct,
	}
}

func (h *PAPExchangeHandler) GetPapExchangeRates(_ context.Context, _ apicontract.GetPapExchangeRatesRequestObject) (apicontract.GetPapExchangeRatesResponseObject, error) {
	var out apicontract.GetPapExchangeRates200JSONResponse
	config, err := h.getPapExchangeRates()
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromPapExchangeSuccess(apicontract.PapExchangeSuccess{Code: response.CodeOK, Msg: "success", Data: config})
}

func (h *PAPExchangeHandler) SetPapExchangeRates(ctx context.Context, req apicontract.SetPapExchangeRatesRequestObject) (apicontract.SetPapExchangeRatesResponseObject, error) {
	var out apicontract.SetPapExchangeRates200JSONResponse
	body := req.Body
	if body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误: EOF"))
	}
	// The generated binder checks these required pointers before calling us.
	// Keep direct calls from passing nil settings through to the service.
	if body.MulticharFullRewardCount == nil || body.MulticharReducedRewardCount == nil || body.MulticharReducedRewardPct == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.SetPapExchangeRates401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	config, err := h.setPapExchangeRates(middleware.GetUserID(c), *body)
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromPapExchangeSuccess(apicontract.PapExchangeSuccess{Code: response.CodeOK, Msg: "success", Data: config})
}
