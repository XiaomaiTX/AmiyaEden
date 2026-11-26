package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"github.com/gin-gonic/gin"
)

const (
	salaryRatesReadMessage     = "读取操作工资失败，请重试"
	salarySettingsReadMessage  = "读取工资配置失败"
	salarySettingsInputMessage = "请填写完整、有效的工资配置"
	salarySettingsSaveMessage  = "保存失败，请检查金额最多两位小数并重试"
)

type SalaryHandler struct{ svc *service.SalaryService }

func NewSalaryHandler() *SalaryHandler { return &SalaryHandler{svc: service.NewSalaryService()} }

func (h *SalaryHandler) operationRates() (apicontract.OperationSalaryRates, error) {
	rates, err := h.svc.OperationRates()
	if err != nil {
		return apicontract.OperationSalaryRates{}, err
	}
	return apicontract.OperationSalaryRates{
		Recruitment: rates.Recruitment, WelfareDelivery: rates.WelfareDelivery,
		ShopDelivery: rates.ShopDelivery, SrpPayout: rates.SRPPayout,
	}, nil
}

func (h *SalaryHandler) salarySettings() (apicontract.SalarySettings, error) {
	settings, err := h.svc.Settings()
	if err != nil {
		return apicontract.SalarySettings{}, err
	}
	return salarySettingsDTO(settings), nil
}

// Callers bind and validate req before invoking the service.
func (h *SalaryHandler) updateSalarySettings(operatorID uint, req apicontract.SalarySettingsInput) (apicontract.SalarySettings, error) {
	settings, err := h.svc.Update(service.SalarySettings{
		Recruitment: *req.Recruitment, WelfareDelivery: *req.WelfareDelivery,
		ShopDelivery: *req.ShopDelivery, SRPPayout: *req.SrpPayout,
		FCSalary: *req.FcSalary, FCSalaryMonthlyLimit: *req.FcSalaryMonthlyLimit,
	}, operatorID)
	if err != nil {
		return apicontract.SalarySettings{}, err
	}
	return salarySettingsDTO(settings), nil
}

func salarySettingsDTO(settings *service.SalarySettings) apicontract.SalarySettings {
	return apicontract.SalarySettings{
		Recruitment: settings.Recruitment, WelfareDelivery: settings.WelfareDelivery,
		ShopDelivery: settings.ShopDelivery, SrpPayout: settings.SRPPayout,
		FcSalary: settings.FCSalary, FcSalaryMonthlyLimit: settings.FCSalaryMonthlyLimit,
	}
}

func (h *SalaryHandler) OperationRates(_ context.Context, _ apicontract.OperationRatesRequestObject) (apicontract.OperationRatesResponseObject, error) {
	var out apicontract.OperationRates200JSONResponse
	data, err := h.operationRates()
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, salaryRatesReadMessage))
	}
	return &out, out.FromOperationSalaryRatesSuccess(apicontract.OperationSalaryRatesSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SalaryHandler) GetSalarySettings(_ context.Context, _ apicontract.GetSalarySettingsRequestObject) (apicontract.GetSalarySettingsResponseObject, error) {
	var out apicontract.GetSalarySettings200JSONResponse
	data, err := h.salarySettings()
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, salarySettingsReadMessage))
	}
	return &out, out.FromSalarySettingsSuccess(apicontract.SalarySettingsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SalaryHandler) UpdateSalarySettings(ctx context.Context, request apicontract.UpdateSalarySettingsRequestObject) (apicontract.UpdateSalarySettingsResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.UpdateSalarySettings401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	var out apicontract.UpdateSalarySettings200JSONResponse
	body := request.Body
	if body == nil || body.Recruitment == nil || body.WelfareDelivery == nil || body.ShopDelivery == nil ||
		body.SrpPayout == nil || body.FcSalary == nil || body.FcSalaryMonthlyLimit == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, salarySettingsInputMessage))
	}
	data, err := h.updateSalarySettings(middleware.GetUserID(c), *body)
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, salarySettingsSaveMessage))
	}
	return &out, out.FromSalarySettingsSuccess(apicontract.SalarySettingsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}
