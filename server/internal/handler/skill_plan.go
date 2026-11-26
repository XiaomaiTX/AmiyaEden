package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"math"

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

func resolveRequestLang(c *gin.Context, queryLang *string) string {
	lang := skillPlanValue(queryLang)
	if lang == "" && c != nil {
		lang = c.GetHeader("Accept-Language")
	}
	if lang == "" && c != nil {
		if cookieLang, err := c.Cookie("language"); err == nil {
			lang = cookieLang
		}
	}
	if lang == "" {
		lang = "zh"
	}
	return lang
}

func skillPlanValue[T any](value *T) T {
	if value != nil {
		return *value
	}
	var zero T
	return zero
}

func skillPlanMap[T, U any](values []T, convert func(T) U) []U {
	if values == nil {
		return nil
	}
	mapped := make([]U, len(values))
	for i, value := range values {
		mapped[i] = convert(value)
	}
	return mapped
}

func skillPlanRequirementRequest(input apicontract.SkillPlanWriteRequirement) service.SkillPlanSkillReq {
	return service.SkillPlanSkillReq{SkillTypeID: input.SkillTypeId, RequiredLevel: input.RequiredLevel}
}

func skillPlanCreateRequest(input apicontract.SkillPlanCreateInput) service.CreateSkillPlanRequest {
	return service.CreateSkillPlanRequest{
		Title: input.Title, Description: input.Description, Disabled: input.Disabled,
		ShipTypeID: input.ShipTypeId, SkillsText: input.SkillsText,
		Skills: skillPlanMap(input.Skills, skillPlanRequirementRequest),
	}
}

func skillPlanUpdateRequest(input apicontract.SkillPlanUpdateInput) service.UpdateSkillPlanRequest {
	return service.UpdateSkillPlanRequest{
		Title: input.Title, Description: input.Description, Disabled: input.Disabled,
		ShipTypeID: input.ShipTypeId, SkillsText: input.SkillsText, SortOrder: input.SortOrder,
		Skills: skillPlanMap(input.Skills, skillPlanRequirementRequest),
	}
}

func skillPlanRecordDTO(record service.SkillPlanListItemResp) apicontract.SkillPlanRecord {
	return apicontract.SkillPlanRecord{
		Id: record.ID, Title: record.Title, Description: record.Description, Disabled: record.Disabled,
		PlanScope: apicontract.SkillPlanScope(record.PlanScope), ShipTypeId: record.ShipTypeID,
		SortOrder: record.SortOrder, CreatedBy: record.CreatedBy, CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt, SkillCount: record.SkillCount,
	}
}

func skillPlanDetailDTO(record *service.SkillPlanDetailResp) apicontract.SkillPlanDetail {
	return apicontract.SkillPlanDetail{
		Id: record.ID, Title: record.Title, Description: record.Description, Disabled: record.Disabled,
		PlanScope: apicontract.SkillPlanScope(record.PlanScope), ShipTypeId: record.ShipTypeID, ShipName: record.ShipName,
		SortOrder: record.SortOrder, CreatedBy: record.CreatedBy, CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt, SkillCount: record.SkillCount,
		Skills: skillPlanMap(record.Skills, func(skill service.SkillPlanSkillResp) apicontract.SkillPlanRequirement {
			return apicontract.SkillPlanRequirement{
				Id: skill.ID, SkillPlanId: skill.SkillPlanID, SkillTypeId: skill.SkillTypeID,
				SkillName: skill.SkillName, GroupName: skill.GroupName,
				RequiredLevel: skill.RequiredLevel, Sort: skill.Sort,
			}
		}),
	}
}

func skillPlanCheckDTO(result *service.SkillPlanCheckResultResp) apicontract.SkillPlanCheckResult {
	return apicontract.SkillPlanCheckResult{
		PlanCount: result.PlanCount, Characters: skillPlanMap(result.Characters, skillPlanCheckCharacterDTO),
	}
}

func skillPlanCheckCharacterDTO(character service.SkillPlanCheckCharacterResp) apicontract.SkillPlanCompletionCharacter {
	return apicontract.SkillPlanCompletionCharacter{
		CharacterId: character.CharacterID, CharacterName: character.CharacterName,
		CompletedPlans: character.CompletedPlans, TotalPlans: character.TotalPlans,
		Plans: skillPlanMap(character.Plans, skillPlanCheckPlanDTO),
	}
}

func skillPlanCheckPlanDTO(plan service.SkillPlanCheckPlanResp) apicontract.SkillPlanCompletionPlan {
	return apicontract.SkillPlanCompletionPlan{
		PlanId: plan.PlanID, PlanTitle: plan.PlanTitle, PlanDescription: plan.PlanDescription,
		ShipTypeId: plan.ShipTypeID, MatchedSkills: plan.MatchedSkills, TotalSkills: plan.TotalSkills,
		FullySatisfied: plan.FullySatisfied,
		MissingSkills: skillPlanMap(plan.MissingSkills, func(skill service.SkillPlanCheckMissingSkillResp) apicontract.SkillPlanMissingSkill {
			return apicontract.SkillPlanMissingSkill{
				SkillTypeId: skill.SkillTypeID, SkillName: skill.SkillName, GroupName: skill.GroupName,
				RequiredLevel: skill.RequiredLevel, CurrentLevel: skill.CurrentLevel,
			}
		}),
	}
}

type skillPlanCaller struct {
	gin    *gin.Context
	userID uint
	roles  []string
}

func skillPlanCallerFrom(ctx context.Context) skillPlanCaller {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return skillPlanCaller{}
	}
	return skillPlanCaller{gin: c, userID: middleware.GetUserID(c), roles: middleware.GetUserRoles(c)}
}

func (c skillPlanCaller) language(query *string) string { return resolveRequestLang(c.gin, query) }

func skillPlanValidID(id uint) bool { return id != 0 && id <= math.MaxUint32 }

func skillPlanInvalidID() apicontract.BusinessError {
	return contractBusinessError(response.CodeParamError, "无效的技能计划 ID")
}

func skillPlanInvalidBody() apicontract.BusinessError {
	return contractBusinessError(response.CodeParamError, "请求参数错误")
}

func (h *SkillPlanHandler) ListSkillPlans(ctx context.Context, request apicontract.ListSkillPlansRequestObject) (apicontract.ListSkillPlansResponseObject, error) {
	out := &apicontract.ListSkillPlans200JSONResponse{}
	data, err := h.listSkillPlans(skillPlanCallerFrom(ctx).roles, request.Params)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSkillPlanPageSuccess(apicontract.SkillPlanPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) ListPersonalSkillPlans(ctx context.Context, request apicontract.ListPersonalSkillPlansRequestObject) (apicontract.ListPersonalSkillPlansResponseObject, error) {
	out := &apicontract.ListPersonalSkillPlans200JSONResponse{}
	data, err := h.listPersonalSkillPlans(skillPlanCallerFrom(ctx).userID, request.Params)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSkillPlanPageSuccess(apicontract.SkillPlanPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) CreateSkillPlan(ctx context.Context, request apicontract.CreateSkillPlanRequestObject) (apicontract.CreateSkillPlanResponseObject, error) {
	out := &apicontract.CreateSkillPlan200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(skillPlanInvalidBody())
	}
	caller := skillPlanCallerFrom(ctx)
	data, err := h.createSkillPlan(caller.userID, *request.Body, caller.language(request.Params.Lang))
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSkillPlanDetailSuccess(apicontract.SkillPlanDetailSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) CreatePersonalSkillPlan(ctx context.Context, request apicontract.CreatePersonalSkillPlanRequestObject) (apicontract.CreatePersonalSkillPlanResponseObject, error) {
	out := &apicontract.CreatePersonalSkillPlan200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(skillPlanInvalidBody())
	}
	caller := skillPlanCallerFrom(ctx)
	data, err := h.createPersonalSkillPlan(caller.userID, *request.Body, caller.language(request.Params.Lang))
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSkillPlanDetailSuccess(apicontract.SkillPlanDetailSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) GetSkillPlan(ctx context.Context, request apicontract.GetSkillPlanRequestObject) (apicontract.GetSkillPlanResponseObject, error) {
	out := &apicontract.GetSkillPlan200JSONResponse{}
	if !skillPlanValidID(request.Id) {
		return out, out.FromBusinessError(skillPlanInvalidID())
	}
	caller := skillPlanCallerFrom(ctx)
	data, err := h.getSkillPlan(request.Id, caller.language(request.Params.Lang), caller.roles)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeNotFound, err.Error()))
	}
	return out, out.FromSkillPlanDetailSuccess(apicontract.SkillPlanDetailSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) GetPersonalSkillPlan(ctx context.Context, request apicontract.GetPersonalSkillPlanRequestObject) (apicontract.GetPersonalSkillPlanResponseObject, error) {
	out := &apicontract.GetPersonalSkillPlan200JSONResponse{}
	if !skillPlanValidID(request.Id) {
		return out, out.FromBusinessError(skillPlanInvalidID())
	}
	caller := skillPlanCallerFrom(ctx)
	data, err := h.getPersonalSkillPlan(request.Id, caller.userID, caller.language(request.Params.Lang))
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeNotFound, err.Error()))
	}
	return out, out.FromSkillPlanDetailSuccess(apicontract.SkillPlanDetailSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) UpdateSkillPlan(ctx context.Context, request apicontract.UpdateSkillPlanRequestObject) (apicontract.UpdateSkillPlanResponseObject, error) {
	out := &apicontract.UpdateSkillPlan200JSONResponse{}
	if !skillPlanValidID(request.Id) {
		return out, out.FromBusinessError(skillPlanInvalidID())
	}
	if request.Body == nil {
		return out, out.FromBusinessError(skillPlanInvalidBody())
	}
	caller := skillPlanCallerFrom(ctx)
	data, err := h.updateSkillPlan(request.Id, caller.userID, caller.roles, *request.Body, caller.language(request.Params.Lang))
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSkillPlanDetailSuccess(apicontract.SkillPlanDetailSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) UpdatePersonalSkillPlan(ctx context.Context, request apicontract.UpdatePersonalSkillPlanRequestObject) (apicontract.UpdatePersonalSkillPlanResponseObject, error) {
	out := &apicontract.UpdatePersonalSkillPlan200JSONResponse{}
	if !skillPlanValidID(request.Id) {
		return out, out.FromBusinessError(skillPlanInvalidID())
	}
	if request.Body == nil {
		return out, out.FromBusinessError(skillPlanInvalidBody())
	}
	caller := skillPlanCallerFrom(ctx)
	data, err := h.updatePersonalSkillPlan(request.Id, caller.userID, *request.Body, caller.language(request.Params.Lang))
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSkillPlanDetailSuccess(apicontract.SkillPlanDetailSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) DeleteSkillPlan(ctx context.Context, request apicontract.DeleteSkillPlanRequestObject) (apicontract.DeleteSkillPlanResponseObject, error) {
	out := &apicontract.DeleteSkillPlan200JSONResponse{}
	if !skillPlanValidID(request.Id) {
		return out, out.FromBusinessError(skillPlanInvalidID())
	}
	caller := skillPlanCallerFrom(ctx)
	if err := h.svc.DeleteSkillPlan(request.Id, caller.userID, caller.roles); err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *SkillPlanHandler) DeletePersonalSkillPlan(ctx context.Context, request apicontract.DeletePersonalSkillPlanRequestObject) (apicontract.DeletePersonalSkillPlanResponseObject, error) {
	out := &apicontract.DeletePersonalSkillPlan200JSONResponse{}
	if !skillPlanValidID(request.Id) {
		return out, out.FromBusinessError(skillPlanInvalidID())
	}
	if err := h.svc.DeletePersonalSkillPlan(request.Id, skillPlanCallerFrom(ctx).userID); err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *SkillPlanHandler) ReorderSkillPlans(ctx context.Context, request apicontract.ReorderSkillPlansRequestObject) (apicontract.ReorderSkillPlansResponseObject, error) {
	out := &apicontract.ReorderSkillPlans200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(skillPlanInvalidBody())
	}
	if err := h.svc.ReorderSkillPlans(request.Body.IDs, skillPlanCallerFrom(ctx).roles); err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *SkillPlanHandler) ReorderPersonalSkillPlans(ctx context.Context, request apicontract.ReorderPersonalSkillPlansRequestObject) (apicontract.ReorderPersonalSkillPlansResponseObject, error) {
	out := &apicontract.ReorderPersonalSkillPlans200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(skillPlanInvalidBody())
	}
	if err := h.svc.ReorderPersonalSkillPlans(request.Body.IDs, skillPlanCallerFrom(ctx).userID); err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *SkillPlanHandler) GetSkillPlanCheckSelection(ctx context.Context, _ apicontract.GetSkillPlanCheckSelectionRequestObject) (apicontract.GetSkillPlanCheckSelectionResponseObject, error) {
	out := &apicontract.GetSkillPlanCheckSelection200JSONResponse{}
	data, err := h.getSkillPlanCheckSelection(skillPlanCallerFrom(ctx).userID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSkillPlanCharacterSelectionSuccess(apicontract.SkillPlanCharacterSelectionSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) SaveSkillPlanCheckSelection(ctx context.Context, request apicontract.SaveSkillPlanCheckSelectionRequestObject) (apicontract.SaveSkillPlanCheckSelectionResponseObject, error) {
	out := &apicontract.SaveSkillPlanCheckSelection200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(skillPlanInvalidBody())
	}
	data, err := h.saveSkillPlanCheckSelection(skillPlanCallerFrom(ctx).userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSkillPlanCharacterSelectionSuccess(apicontract.SkillPlanCharacterSelectionSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) GetSkillPlanCheckPlanSelection(ctx context.Context, _ apicontract.GetSkillPlanCheckPlanSelectionRequestObject) (apicontract.GetSkillPlanCheckPlanSelectionResponseObject, error) {
	out := &apicontract.GetSkillPlanCheckPlanSelection200JSONResponse{}
	data, err := h.getSkillPlanCheckPlanSelection(skillPlanCallerFrom(ctx).userID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSkillPlanPlanSelectionSuccess(apicontract.SkillPlanPlanSelectionSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) SaveSkillPlanCheckPlanSelection(ctx context.Context, request apicontract.SaveSkillPlanCheckPlanSelectionRequestObject) (apicontract.SaveSkillPlanCheckPlanSelectionResponseObject, error) {
	out := &apicontract.SaveSkillPlanCheckPlanSelection200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(skillPlanInvalidBody())
	}
	data, err := h.saveSkillPlanCheckPlanSelection(skillPlanCallerFrom(ctx).userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSkillPlanPlanSelectionSuccess(apicontract.SkillPlanPlanSelectionSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) RunSkillPlanCompletionCheck(ctx context.Context, request apicontract.RunSkillPlanCompletionCheckRequestObject) (apicontract.RunSkillPlanCompletionCheckResponseObject, error) {
	out := &apicontract.RunSkillPlanCompletionCheck200JSONResponse{}
	caller := skillPlanCallerFrom(ctx)
	data, err := h.runSkillPlanCompletionCheck(caller.userID, skillPlanValue(request.Body), caller.language(request.Params.Lang))
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSkillPlanCheckSuccess(apicontract.SkillPlanCheckSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillPlanHandler) createSkillPlan(userID uint, input apicontract.SkillPlanCreateInput, lang string) (apicontract.SkillPlanDetail, error) {
	req := skillPlanCreateRequest(input)
	result, err := h.svc.CreateSkillPlan(userID, &req, lang)
	return skillPlanDetailResult(result, err)
}

func (h *SkillPlanHandler) createPersonalSkillPlan(userID uint, input apicontract.SkillPlanCreateInput, lang string) (apicontract.SkillPlanDetail, error) {
	req := skillPlanCreateRequest(input)
	result, err := h.svc.CreatePersonalSkillPlan(userID, &req, lang)
	return skillPlanDetailResult(result, err)
}

func (h *SkillPlanHandler) listSkillPlans(roles []string, params apicontract.ListSkillPlansParams) (apicontract.SkillPlanPage, error) {
	page, size := normalizePagination(skillPlanValue(params.Current), skillPlanValue(params.Size), 50, 100)
	records, total, err := h.svc.ListSkillPlans(page, size, skillPlanValue(params.Keyword), roles)
	return skillPlanPageResult(records, total, page, size, err)
}

func (h *SkillPlanHandler) listPersonalSkillPlans(userID uint, params apicontract.ListPersonalSkillPlansParams) (apicontract.SkillPlanPage, error) {
	page, size := normalizePagination(skillPlanValue(params.Current), skillPlanValue(params.Size), 50, 100)
	records, total, err := h.svc.ListPersonalSkillPlans(userID, page, size, skillPlanValue(params.Keyword))
	return skillPlanPageResult(records, total, page, size, err)
}

func (h *SkillPlanHandler) getSkillPlan(id uint, lang string, roles []string) (apicontract.SkillPlanDetail, error) {
	result, err := h.svc.GetSkillPlan(id, lang, roles)
	return skillPlanDetailResult(result, err)
}

func (h *SkillPlanHandler) getPersonalSkillPlan(id, userID uint, lang string) (apicontract.SkillPlanDetail, error) {
	result, err := h.svc.GetPersonalSkillPlan(id, userID, lang)
	return skillPlanDetailResult(result, err)
}

func (h *SkillPlanHandler) updateSkillPlan(id, userID uint, roles []string, input apicontract.SkillPlanUpdateInput, lang string) (apicontract.SkillPlanDetail, error) {
	req := skillPlanUpdateRequest(input)
	result, err := h.svc.UpdateSkillPlan(id, userID, roles, &req, lang)
	return skillPlanDetailResult(result, err)
}

func (h *SkillPlanHandler) updatePersonalSkillPlan(id, userID uint, input apicontract.SkillPlanUpdateInput, lang string) (apicontract.SkillPlanDetail, error) {
	req := skillPlanUpdateRequest(input)
	result, err := h.svc.UpdatePersonalSkillPlan(id, userID, &req, lang)
	return skillPlanDetailResult(result, err)
}

func (h *SkillPlanHandler) getSkillPlanCheckSelection(userID uint) (apicontract.SkillPlanCharacterSelection, error) {
	result, err := h.svc.GetCheckSelection(userID)
	return skillPlanCharacterSelectionResult(result, err)
}

func (h *SkillPlanHandler) saveSkillPlanCheckSelection(userID uint, input apicontract.SkillPlanCharacterSelectionInput) (apicontract.SkillPlanCharacterSelection, error) {
	result, err := h.svc.SaveCheckSelection(userID, &service.SkillPlanCheckSelectionRequest{CharacterIDs: input.CharacterIDs})
	return skillPlanCharacterSelectionResult(result, err)
}

func (h *SkillPlanHandler) getSkillPlanCheckPlanSelection(userID uint) (apicontract.SkillPlanPlanSelection, error) {
	result, err := h.svc.GetCheckPlanSelection(userID)
	return skillPlanPlanSelectionResult(result, err)
}

func (h *SkillPlanHandler) saveSkillPlanCheckPlanSelection(userID uint, input apicontract.SkillPlanPlanSelectionInput) (apicontract.SkillPlanPlanSelection, error) {
	result, err := h.svc.SaveCheckPlanSelection(userID, &service.SkillPlanCheckPlanSelectionRequest{PlanIDs: input.PlanIDs})
	return skillPlanPlanSelectionResult(result, err)
}

func (h *SkillPlanHandler) runSkillPlanCompletionCheck(userID uint, input apicontract.SkillPlanCheckInput, fallbackLang string) (apicontract.SkillPlanCheckResult, error) {
	if input.Language == "" {
		input.Language = fallbackLang
	}
	result, err := h.svc.RunCompletionCheck(userID, &service.RunSkillPlanCheckRequest{CharacterIDs: input.CharacterIDs, Language: input.Language})
	if err != nil {
		return apicontract.SkillPlanCheckResult{}, err
	}
	return skillPlanCheckDTO(result), nil
}

func skillPlanDetailResult(record *service.SkillPlanDetailResp, err error) (apicontract.SkillPlanDetail, error) {
	if err != nil {
		return apicontract.SkillPlanDetail{}, err
	}
	return skillPlanDetailDTO(record), nil
}

func skillPlanPageResult(records []service.SkillPlanListItemResp, total int64, page, size int, err error) (apicontract.SkillPlanPage, error) {
	if err != nil {
		return apicontract.SkillPlanPage{}, err
	}
	return apicontract.SkillPlanPage{
		List: skillPlanMap(records, skillPlanRecordDTO), Total: total, Page: page, PageSize: size,
	}, nil
}

func skillPlanCharacterSelectionResult(result *service.SkillPlanCheckSelectionResp, err error) (apicontract.SkillPlanCharacterSelection, error) {
	if err != nil {
		return apicontract.SkillPlanCharacterSelection{}, err
	}
	return apicontract.SkillPlanCharacterSelection{CharacterIDs: result.CharacterIDs}, nil
}

func skillPlanPlanSelectionResult(result *service.SkillPlanCheckPlanSelectionResp, err error) (apicontract.SkillPlanPlanSelection, error) {
	if err != nil {
		return apicontract.SkillPlanPlanSelection{}, err
	}
	return apicontract.SkillPlanPlanSelection{PlanIDs: result.PlanIDs}, nil
}
