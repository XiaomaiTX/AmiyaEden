package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"strings"
)

const (
	evidenceMaxBytes = 2048 << 10 // 2MB
)

var evidenceAllowedMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// WelfareHandler 福利 HTTP 处理器
type WelfareHandler struct {
	svc         *service.WelfareService
	settingsSvc welfareSettingsService
}

type welfareSettingsService interface {
	GetSettings() service.WelfareSettings
	UpdateSettingsByOperator(cfg service.WelfareSettings, operatorID uint) (service.WelfareSettings, error)
}

func NewWelfareHandler() *WelfareHandler {
	return &WelfareHandler{
		svc:         service.NewWelfareService(),
		settingsSvc: service.NewWelfareSettingsService(),
	}
}

// ─────────────────────────────────────────────
//  管理员端（全部 POST）
// ─────────────────────────────────────────────

func (h *WelfareHandler) adminCreateWelfare(req apicontract.AdminCreateWelfareJSONRequestBody, userID uint) (apicontract.WelfareItem, error) {
	w := &model.Welfare{
		Name:                   req.Name,
		Description:            req.Description,
		DistMode:               req.DistMode,
		PayByFuxiCoin:          req.PayByFuxiCoin,
		RequireSkillPlan:       req.RequireSkillPlan,
		SkillPlanIDs:           req.SkillPlanIDs,
		MaxCharAgeMonths:       req.MaxCharAgeMonths,
		MinimumPap:             req.MinimumPap,
		MinimumFuxiLegionYears: req.MinimumFuxiLegionYears,
		RequireEvidence:        req.RequireEvidence,
		ExampleEvidence:        req.ExampleEvidence,
		Status:                 req.Status,
		SortOrder:              req.SortOrder,
		CreatedBy:              userID,
	}

	if err := h.svc.AdminCreateWelfare(w); err != nil {
		return apicontract.WelfareItem{}, err
	}
	return welfareItemDTO(*w), nil
}

func (h *WelfareHandler) adminUpdateWelfare(req apicontract.AdminUpdateWelfareJSONRequestBody) (apicontract.WelfareItem, error) {
	w, err := h.svc.AdminUpdateWelfare(req.ID, &service.AdminUpdateWelfareRequest{
		Name:                   req.Name,
		Description:            req.Description,
		DistMode:               req.DistMode,
		PayByFuxiCoin:          req.PayByFuxiCoin,
		RequireSkillPlan:       req.RequireSkillPlan,
		SkillPlanIDs:           req.SkillPlanIDs,
		MaxCharAgeMonths:       req.MaxCharAgeMonths,
		MinimumPap:             req.MinimumPap,
		MinimumFuxiLegionYears: req.MinimumFuxiLegionYears,
		RequireEvidence:        req.RequireEvidence,
		ExampleEvidence:        req.ExampleEvidence,
		Status:                 req.Status,
		SortOrder:              req.SortOrder,
	})
	if err != nil {
		return apicontract.WelfareItem{}, err
	}
	return welfareItemDTO(*w), nil
}

func (h *WelfareHandler) adminDeleteWelfare(req apicontract.AdminDeleteWelfareJSONRequestBody) error {
	return h.svc.AdminDeleteWelfare(req.ID)
}

func (h *WelfareHandler) adminListWelfares(req apicontract.AdminListWelfaresJSONRequestBody, badBody bool) (apicontract.WelfarePage, error) {
	if badBody {
		req.Current = 1
		req.Size = 20
	}
	req.Current, req.Size = normalizePagination(req.Current, req.Size, 20, 100)

	filter := service.WelfareFilter{
		Status: req.Status,
		Name:   req.Name,
	}

	list, total, err := h.svc.AdminListWelfares(req.Current, req.Size, filter)
	if err != nil {
		return apicontract.WelfarePage{}, err
	}
	listDTO := mapContractRows(list, welfareItemDTO)
	return apicontract.WelfarePage{List: &listDTO, Total: total, Page: req.Current, PageSize: req.Size}, nil
}

func (h *WelfareHandler) adminGetWelfareSettings() apicontract.WelfareSettings {
	return welfareSettingsDTO(h.settingsSvc.GetSettings())
}

func (h *WelfareHandler) adminUpdateWelfareSettings(req apicontract.AdminUpdateWelfareSettingsJSONRequestBody, operatorID uint) (apicontract.WelfareSettings, error) {
	updated, err := h.settingsSvc.UpdateSettingsByOperator(service.WelfareSettings{
		AutoApproveFuxiCoinThreshold: *req.AutoApproveFuxiCoinThreshold,
	}, operatorID)
	if err != nil {
		return apicontract.WelfareSettings{}, err
	}

	return welfareSettingsDTO(updated), nil
}

func (h *WelfareHandler) adminReorderWelfares(req apicontract.AdminReorderWelfaresJSONRequestBody) error {
	return h.svc.AdminReorderWelfares(req.IDs)
}

// ─────────────────────────────────────────────
//  管理端 - 导入历史记录
// ─────────────────────────────────────────────

func (h *WelfareHandler) adminImportWelfareRecords(req apicontract.AdminImportWelfareRecordsJSONRequestBody) (apicontract.WelfareImportResult, error) {
	count, err := h.svc.ImportWelfareRecords(&service.ImportWelfareRecordsRequest{
		WelfareID: req.WelfareID,
		CSV:       req.CSV,
	})
	if err != nil {
		return apicontract.WelfareImportResult{}, err
	}
	return apicontract.WelfareImportResult{Count: count}, nil
}

// ─────────────────────────────────────────────
//  管理端 - 福利审批
// ─────────────────────────────────────────────

func (h *WelfareHandler) adminListWelfareApplications(req apicontract.AdminListWelfareApplicationsJSONRequestBody, badBody bool) (apicontract.WelfareAdminApplicationPage, error) {
	if badBody {
		req.Current = 1
		req.Size = 50
	}
	req.Current, req.Size = normalizeLedgerPagination(req.Current, req.Size)

	var filter service.WelfareApplicationFilter
	if strings.Contains(req.Status, ",") {
		filter.StatusIn = strings.Split(req.Status, ",")
	} else {
		filter.Status = req.Status
	}
	filter.Keyword = req.Keyword
	filter.IncludeEvidenceImage = filter.Status == model.WelfareAppStatusRequested

	list, total, err := h.svc.AdminListApplications(req.Current, req.Size, filter)
	if err != nil {
		return apicontract.WelfareAdminApplicationPage{}, err
	}
	return apicontract.WelfareAdminApplicationPage{List: mapContractRows(list, welfareAdminApplicationDTO), Total: total, Page: req.Current, PageSize: req.Size}, nil
}

func (h *WelfareHandler) adminDeleteWelfareApplication(req apicontract.AdminDeleteWelfareApplicationJSONRequestBody) error {
	return h.svc.AdminDeleteApplication(req.ID)
}

func (h *WelfareHandler) adminReviewWelfareApplication(req apicontract.AdminReviewWelfareApplicationJSONRequestBody, reviewerID uint, reviewerRoles []string) (apicontract.WelfareReviewResult, error) {
	result, err := h.svc.AdminReviewApplication(req.ID, reviewerID, reviewerRoles, &service.AdminReviewApplicationRequest{
		Action: req.Action,
	})
	if err != nil {
		return apicontract.WelfareReviewResult{}, err
	}
	return welfareReviewDTO(result), nil
}

// ─────────────────────────────────────────────
//  用户端
// ─────────────────────────────────────────────

func (h *WelfareHandler) getEligibleWelfares(userID uint) ([]apicontract.WelfareEligibleItem, error) {
	result, err := h.svc.GetEligibleWelfares(userID)
	if err != nil {
		return nil, err
	}
	return mapContractRows(result, welfareEligibleItemDTO), nil
}

func (h *WelfareHandler) applyForWelfare(req apicontract.ApplyForWelfareJSONRequestBody, userID uint) (apicontract.WelfareApplication, error) {
	app, err := h.svc.ApplyForWelfare(userID, &service.ApplyForWelfareRequest{
		WelfareID:     req.WelfareID,
		CharacterID:   req.CharacterID,
		EvidenceImage: req.EvidenceImage,
	})
	if err != nil {
		return apicontract.WelfareApplication{}, err
	}
	return welfareApplicationDTO(*app), nil
}

func (h *WelfareHandler) listMyWelfareApplications(req apicontract.ListMyWelfareApplicationsJSONRequestBody, userID uint) (apicontract.WelfareMyApplicationPage, error) {
	req.Current, req.Size = normalizePagination(req.Current, req.Size, 10, 100)

	result, total, err := h.svc.ListMyApplications(userID, req.Current, req.Size, req.Status)
	if err != nil {
		return apicontract.WelfareMyApplicationPage{}, err
	}
	return apicontract.WelfareMyApplicationPage{List: mapContractRows(result, welfareMyApplicationDTO), Total: total, Page: req.Current, PageSize: req.Size}, nil
}

func welfareItemDTO(value model.Welfare) apicontract.WelfareItem {
	return apicontract.WelfareItem{
		ID:                     value.ID,
		CreatedAt:              value.CreatedAt,
		UpdatedAt:              value.UpdatedAt,
		Name:                   value.Name,
		Description:            value.Description,
		DistMode:               value.DistMode,
		PayByFuxiCoin:          value.PayByFuxiCoin,
		RequireSkillPlan:       value.RequireSkillPlan,
		MaxCharAgeMonths:       value.MaxCharAgeMonths,
		MinimumPap:             value.MinimumPap,
		MinimumFuxiLegionYears: value.MinimumFuxiLegionYears,
		RequireEvidence:        value.RequireEvidence,
		ExampleEvidence:        value.ExampleEvidence,
		Status:                 value.Status,
		SortOrder:              value.SortOrder,
		CreatedBy:              value.CreatedBy,
		SkillPlanIDs:           &value.SkillPlanIDs,
		SkillPlanNames:         &value.SkillPlanNames,
	}
}

func welfareApplicationDTO(value model.WelfareApplication) apicontract.WelfareApplication {
	return apicontract.WelfareApplication{
		ID:            value.ID,
		CreatedAt:     value.CreatedAt,
		UpdatedAt:     value.UpdatedAt,
		WelfareID:     value.WelfareID,
		UserID:        value.UserID,
		CharacterID:   value.CharacterID,
		CharacterName: value.CharacterName,
		QQ:            value.QQ,
		DiscordID:     value.DiscordID,
		EvidenceImage: value.EvidenceImage,
		Status:        value.Status,
		ReviewedBy:    value.ReviewedBy,
		ReviewedAt:    value.ReviewedAt,
	}
}

func welfareSettingsDTO(value service.WelfareSettings) apicontract.WelfareSettings {
	return apicontract.WelfareSettings{
		AutoApproveFuxiCoinThreshold: value.AutoApproveFuxiCoinThreshold,
	}
}

func welfareAdminApplicationDTO(value service.AdminApplicationResp) apicontract.WelfareAdminApplication {
	return apicontract.WelfareAdminApplication{
		ID:                value.ID,
		WelfareID:         value.WelfareID,
		WelfareName:       value.WelfareName,
		WelfareDesc:       value.WelfareDesc,
		UserID:            value.UserID,
		ApplicantNickname: value.ApplicantNickname,
		CharacterName:     value.CharacterName,
		QQ:                value.QQ,
		DiscordID:         value.DiscordID,
		EvidenceImage:     value.EvidenceImage,
		Status:            value.Status,
		ReviewedBy:        value.ReviewedBy,
		ReviewerName:      value.ReviewerName,
		CreatedAt:         value.CreatedAt,
		ReviewedAt:        value.ReviewedAt,
	}
}

func welfareMyApplicationDTO(value service.MyApplicationResp) apicontract.WelfareMyApplication {
	return apicontract.WelfareMyApplication{
		ID:            value.ID,
		WelfareID:     value.WelfareID,
		WelfareName:   value.WelfareName,
		CharacterName: value.CharacterName,
		Status:        value.Status,
		ReviewerName:  value.ReviewerName,
		CreatedAt:     value.CreatedAt,
		ReviewedAt:    value.ReviewedAt,
	}
}

func welfareEligibleCharacterDTO(value service.EligibleCharacterResp) apicontract.WelfareEligibleCharacter {
	return apicontract.WelfareEligibleCharacter{
		CharacterID:      value.CharacterID,
		CharacterName:    value.CharacterName,
		CanApplyNow:      value.CanApplyNow,
		IneligibleReason: optionalContractValue(value.IneligibleReason),
	}
}

func welfareEligibleItemDTO(value service.EligibleWelfareResp) apicontract.WelfareEligibleItem {
	characters := mapContractRows(value.EligibleCharacters, welfareEligibleCharacterDTO)
	return apicontract.WelfareEligibleItem{
		ID:                 value.ID,
		Name:               value.Name,
		Description:        value.Description,
		DistMode:           value.DistMode,
		SkillPlanNames:     value.SkillPlanNames,
		RequireEvidence:    value.RequireEvidence,
		ExampleEvidence:    value.ExampleEvidence,
		CanApplyNow:        value.CanApplyNow,
		IneligibleReason:   optionalContractValue(value.IneligibleReason),
		EligibleCharacters: &characters,
	}
}

func welfareReviewDTO(value service.WelfareReviewResult) apicontract.WelfareReviewResult {
	return apicontract.WelfareReviewResult{
		Outcome:                    string(value.Outcome),
		EligibilityReason:          optionalContractValue(string(value.EligibilityReason)),
		MailError:                  optionalContractValue(value.MailError),
		MailID:                     optionalContractValue(value.MailID),
		MailSenderCharacterID:      optionalContractValue(value.MailSenderCharacterID),
		MailSenderCharacterName:    optionalContractValue(value.MailSenderCharacterName),
		MailRecipientCharacterID:   optionalContractValue(value.MailRecipientCharacterID),
		MailRecipientCharacterName: optionalContractValue(value.MailRecipientCharacterName),
	}
}

func (h *WelfareHandler) AdminCreateWelfare(ctx context.Context, request apicontract.AdminCreateWelfareRequestObject) (apicontract.AdminCreateWelfareResponseObject, error) {
	out := &apicontract.AdminCreateWelfare200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	userID, _ := commerceIdentity(ctx)
	data, err := h.adminCreateWelfare(*request.Body, userID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromWelfareItemSuccess(apicontract.WelfareItemSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *WelfareHandler) AdminUpdateWelfare(_ context.Context, request apicontract.AdminUpdateWelfareRequestObject) (apicontract.AdminUpdateWelfareResponseObject, error) {
	out := &apicontract.AdminUpdateWelfare200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	data, err := h.adminUpdateWelfare(*request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromWelfareItemSuccess(apicontract.WelfareItemSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *WelfareHandler) AdminDeleteWelfare(_ context.Context, request apicontract.AdminDeleteWelfareRequestObject) (apicontract.AdminDeleteWelfareResponseObject, error) {
	out := &apicontract.AdminDeleteWelfare200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	if err := h.adminDeleteWelfare(*request.Body); err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *WelfareHandler) AdminListWelfares(ctx context.Context, request apicontract.AdminListWelfaresRequestObject) (apicontract.AdminListWelfaresResponseObject, error) {
	out := &apicontract.AdminListWelfares200JSONResponse{}
	body, badBody := commerceOptionalBody(ctx, request.Body)
	data, err := h.adminListWelfares(body, badBody)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromWelfarePageSuccess(apicontract.WelfarePageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *WelfareHandler) AdminGetWelfareSettings(_ context.Context, _ apicontract.AdminGetWelfareSettingsRequestObject) (apicontract.AdminGetWelfareSettingsResponseObject, error) {
	out := &apicontract.AdminGetWelfareSettings200JSONResponse{}
	data := h.adminGetWelfareSettings()
	return out, out.FromWelfareSettingsSuccess(apicontract.WelfareSettingsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *WelfareHandler) AdminUpdateWelfareSettings(ctx context.Context, request apicontract.AdminUpdateWelfareSettingsRequestObject) (apicontract.AdminUpdateWelfareSettingsResponseObject, error) {
	out := &apicontract.AdminUpdateWelfareSettings200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	operatorID, _ := commerceIdentity(ctx)
	data, err := h.adminUpdateWelfareSettings(*request.Body, operatorID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromWelfareSettingsSuccess(apicontract.WelfareSettingsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *WelfareHandler) AdminReorderWelfares(_ context.Context, request apicontract.AdminReorderWelfaresRequestObject) (apicontract.AdminReorderWelfaresResponseObject, error) {
	out := &apicontract.AdminReorderWelfares200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	if err := h.adminReorderWelfares(*request.Body); err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *WelfareHandler) AdminImportWelfareRecords(_ context.Context, request apicontract.AdminImportWelfareRecordsRequestObject) (apicontract.AdminImportWelfareRecordsResponseObject, error) {
	out := &apicontract.AdminImportWelfareRecords200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	data, err := h.adminImportWelfareRecords(*request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromWelfareImportResultSuccess(apicontract.WelfareImportResultSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *WelfareHandler) AdminListWelfareApplications(ctx context.Context, request apicontract.AdminListWelfareApplicationsRequestObject) (apicontract.AdminListWelfareApplicationsResponseObject, error) {
	out := &apicontract.AdminListWelfareApplications200JSONResponse{}
	body, badBody := commerceOptionalBody(ctx, request.Body)
	data, err := h.adminListWelfareApplications(body, badBody)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromWelfareAdminApplicationPageSuccess(apicontract.WelfareAdminApplicationPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *WelfareHandler) AdminDeleteWelfareApplication(_ context.Context, request apicontract.AdminDeleteWelfareApplicationRequestObject) (apicontract.AdminDeleteWelfareApplicationResponseObject, error) {
	out := &apicontract.AdminDeleteWelfareApplication200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	if err := h.adminDeleteWelfareApplication(*request.Body); err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *WelfareHandler) AdminReviewWelfareApplication(ctx context.Context, request apicontract.AdminReviewWelfareApplicationRequestObject) (apicontract.AdminReviewWelfareApplicationResponseObject, error) {
	out := &apicontract.AdminReviewWelfareApplication200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	reviewerID, reviewerRoles := commerceIdentity(ctx)
	data, err := h.adminReviewWelfareApplication(*request.Body, reviewerID, reviewerRoles)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromWelfareReviewResultSuccess(apicontract.WelfareReviewResultSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *WelfareHandler) GetEligibleWelfares(ctx context.Context, _ apicontract.GetEligibleWelfaresRequestObject) (apicontract.GetEligibleWelfaresResponseObject, error) {
	out := &apicontract.GetEligibleWelfares200JSONResponse{}
	userID, _ := commerceIdentity(ctx)
	data, err := h.getEligibleWelfares(userID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromWelfareEligibleListSuccess(apicontract.WelfareEligibleListSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *WelfareHandler) ApplyForWelfare(ctx context.Context, request apicontract.ApplyForWelfareRequestObject) (apicontract.ApplyForWelfareResponseObject, error) {
	out := &apicontract.ApplyForWelfare200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	userID, _ := commerceIdentity(ctx)
	data, err := h.applyForWelfare(*request.Body, userID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromWelfareApplicationSuccess(apicontract.WelfareApplicationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *WelfareHandler) ListMyWelfareApplications(ctx context.Context, request apicontract.ListMyWelfareApplicationsRequestObject) (apicontract.ListMyWelfareApplicationsResponseObject, error) {
	out := &apicontract.ListMyWelfareApplications200JSONResponse{}
	body, _ := commerceOptionalBody(ctx, request.Body)
	userID, _ := commerceIdentity(ctx)
	data, err := h.listMyWelfareApplications(body, userID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromWelfareMyApplicationPageSuccess(apicontract.WelfareMyApplicationPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}
