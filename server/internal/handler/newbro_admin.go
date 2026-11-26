package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"io"
)

type NewbroAdminHandler struct {
	settingsSvc newbroRecruitSettingsService
}

type newbroRecruitSettingsService interface {
	GetRecruitSettings() service.NewbroRecruitSettings
	UpdateRecruitSettingsByOperator(service.NewbroRecruitSettings, uint) (service.NewbroRecruitSettings, error)
}

func NewNewbroAdminHandler() *NewbroAdminHandler {
	return &NewbroAdminHandler{service.NewNewbroSettingsService()}
}

func (h *NewbroAdminHandler) getRecruitSettings() apicontract.NewbroRecruitSettings {
	return newbroRecruitSettings(h.settingsSvc.GetRecruitSettings())
}

func (h *NewbroAdminHandler) updateRecruitSettings(operatorID uint, req apicontract.UpdateNewbroRecruitSettingsRequest) (apicontract.NewbroRecruitSettings, error) {
	updated, err := h.settingsSvc.UpdateRecruitSettingsByOperator(service.NewbroRecruitSettings{
		RecruitQQURL:        req.RecruitQQURL,
		RecruitRewardAmount: *req.RecruitRewardAmount,
		RecruitCooldownDays: req.RecruitCooldownDays,
	}, operatorID)
	return newbroRecruitSettings(updated), err
}

func newbroRecruitSettings(settings service.NewbroRecruitSettings) apicontract.NewbroRecruitSettings {
	return apicontract.NewbroRecruitSettings{
		RecruitQQURL: settings.RecruitQQURL, RecruitRewardAmount: settings.RecruitRewardAmount,
		RecruitCooldownDays: settings.RecruitCooldownDays,
	}
}

func (h *NewbroAdminHandler) GetRecruitSettings(_ context.Context, _ apicontract.GetRecruitSettingsRequestObject) (apicontract.GetRecruitSettingsResponseObject, error) {
	return apicontract.GetRecruitSettings200JSONResponse{Code: response.CodeOK, Msg: "success", Data: h.getRecruitSettings()}, nil
}

func (h *NewbroAdminHandler) UpdateRecruitSettings(ctx context.Context, request apicontract.UpdateRecruitSettingsRequestObject) (apicontract.UpdateRecruitSettingsResponseObject, error) {
	out := &apicontract.UpdateRecruitSettings200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "invalid request: "+io.EOF.Error()))
	}
	if request.Body.RecruitRewardAmount == nil {
		return out, out.FromBusinessError(newbroInvalidRequest())
	}
	data, err := h.updateRecruitSettings(newbroStrictUserID(ctx), *request.Body)
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromNewbroRecruitSettingsSuccess(apicontract.NewbroRecruitSettingsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}
