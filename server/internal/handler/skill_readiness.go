package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"errors"
)

type SkillReadinessHandler struct {
	svc *service.SkillReadinessService
}

func NewSkillReadinessHandler() *SkillReadinessHandler {
	return &SkillReadinessHandler{svc: service.NewSkillReadinessService()}
}

func readinessFailure(err error) contractOperationFailure {
	switch {
	case errors.Is(err, service.ErrSkillReadinessRequest):
		return contractOperationFailure{response.CodeParamError, "查询参数无效"}
	case errors.Is(err, service.ErrSkillReadinessPlan):
		return contractOperationFailure{response.CodeBizError, "请选择包含技能要求的有效军团方案"}
	case errors.Is(err, service.ErrSkillReadinessCharacter):
		return contractOperationFailure{response.CodeBizError, "角色不存在或不在查询范围内"}
	default:
		return contractOperationFailure{response.CodeBizError, "获取军团技能数据失败"}
	}
}

func (h *SkillReadinessHandler) getSkillReadinessOptions(ctx context.Context) (apicontract.SkillReadinessOptions, error) {
	result, err := h.svc.Options(ctx)
	if err != nil {
		return apicontract.SkillReadinessOptions{}, err
	}
	return readinessOptionsDTO(*result), nil
}

func readinessRequest(req apicontract.SkillReadinessRequest) service.SkillReadinessRequest {
	return service.SkillReadinessRequest{
		CorporationIDs: req.CorporationIDs, PlanID: req.PlanId,
		Page: req.Page, PageSize: req.PageSize, Language: string(req.Language),
	}
}

func (h *SkillReadinessHandler) querySkillReadiness(ctx context.Context, req service.SkillReadinessRequest) (apicontract.SkillReadinessResult, error) {
	result, err := h.svc.Query(ctx, req)
	if err != nil {
		return apicontract.SkillReadinessResult{}, err
	}
	return readinessResultDTO(*result), nil
}

func (h *SkillReadinessHandler) getSkillReadinessUserCharacters(ctx context.Context, id uint, req service.SkillReadinessRequest) ([]apicontract.SkillReadinessCharacter, error) {
	if id == 0 {
		return nil, service.ErrSkillReadinessRequest
	}
	result, err := h.svc.UserCharacters(ctx, id, req)
	if err != nil {
		return nil, err
	}
	return dashboardDTOs(result, readinessCharacterDTO), nil
}

func (h *SkillReadinessHandler) getSkillReadinessCharacterSkills(ctx context.Context, id int64, req service.SkillReadinessRequest) (apicontract.SkillReadinessCharacterDetail, error) {
	if id <= 0 {
		return apicontract.SkillReadinessCharacterDetail{}, service.ErrSkillReadinessRequest
	}
	result, err := h.svc.CharacterSkills(ctx, id, req)
	if err != nil {
		return apicontract.SkillReadinessCharacterDetail{}, err
	}
	return readinessCharacterDetailDTO(*result), nil
}

func readinessOptionsDTO(result service.SkillReadinessOptions) apicontract.SkillReadinessOptions {
	return apicontract.SkillReadinessOptions{
		Corporations: dashboardDTOs(result.Corporations, func(row service.SkillReadinessCorporationOption) apicontract.SkillReadinessCorporationOption {
			return apicontract.SkillReadinessCorporationOption{CorporationId: row.CorporationID, CorporationName: row.CorporationName}
		}),
		Plans: dashboardDTOs(result.Plans, func(row service.SkillReadinessPlanOption) apicontract.SkillReadinessPlanOption {
			return apicontract.SkillReadinessPlanOption{Id: row.ID, Title: row.Title, Description: row.Description, ShipTypeId: row.ShipTypeID}
		}),
	}
}

func readinessResultDTO(result service.SkillReadinessResult) apicontract.SkillReadinessResult {
	return apicontract.SkillReadinessResult{
		Summary: apicontract.SkillReadinessSummary{
			CharacterCount: result.Summary.CharacterCount, UserCount: result.Summary.UserCount,
			SatisfiedUserCount: result.Summary.SatisfiedUserCount, SatisfiedCharacterCount: result.Summary.SatisfiedCharacterCount,
			UnknownCharacterCount: result.Summary.UnknownCharacterCount,
		},
		List: dashboardDTOs(result.List, func(row service.SkillReadinessUser) apicontract.SkillReadinessUser {
			return apicontract.SkillReadinessUser{
				UserId: row.UserID, Nickname: row.Nickname, CharacterCount: row.CharacterCount,
				SatisfiedCharacterCount: row.SatisfiedCharacterCount, UnknownCharacterCount: row.UnknownCharacterCount,
			}
		}),
		Total: result.Total, Page: result.Page, PageSize: result.PageSize,
	}
}

func readinessCharacterDTO(row service.SkillReadinessCharacter) apicontract.SkillReadinessCharacter {
	return apicontract.SkillReadinessCharacter{
		UserId: row.UserID, CharacterId: row.CharacterID, CharacterName: row.CharacterName,
		CorporationId: row.CorporationID, CorporationName: row.CorporationName,
		Status: apicontract.SkillReadinessStatus(row.Status), MatchedSkills: row.MatchedSkills,
		TotalSkills: row.TotalSkills, SkillUpdatedAt: row.SkillUpdatedAt, TokenInvalid: row.TokenInvalid,
	}
}

func readinessCharacterDetailDTO(result service.SkillReadinessCharacterDetail) apicontract.SkillReadinessCharacterDetail {
	skill := func(row service.SkillReadinessSkill) apicontract.SkillReadinessSkill {
		return apicontract.SkillReadinessSkill{
			SkillTypeId: row.SkillTypeID, SkillName: row.SkillName, GroupName: row.GroupName,
			ActiveLevel: row.ActiveLevel, TrainedLevel: row.TrainedLevel, SkillpointsInSkill: row.SkillpointsInSkill,
			RequiredLevel: row.RequiredLevel, CurrentLevel: row.CurrentLevel, Satisfied: row.Satisfied,
		}
	}
	return apicontract.SkillReadinessCharacterDetail{
		Character:    readinessCharacterDTO(result.Character),
		Requirements: dashboardDTOs(result.Requirements, skill), Skills: dashboardDTOs(result.Skills, skill),
	}
}

func readinessBusinessError(err error) apicontract.BusinessError {
	failure := readinessFailure(err)
	return contractBusinessError(failure.code, failure.message)
}

func (h *SkillReadinessHandler) GetSkillReadinessOptions(ctx context.Context, _ apicontract.GetSkillReadinessOptionsRequestObject) (apicontract.GetSkillReadinessOptionsResponseObject, error) {
	out := &apicontract.GetSkillReadinessOptions200JSONResponse{}
	requestCtx, _ := strictGinRequestDetails(ctx)
	data, err := h.getSkillReadinessOptions(requestCtx)
	if err != nil {
		return out, out.FromBusinessError(readinessBusinessError(err))
	}
	return out, out.FromSkillReadinessOptionsSuccess(apicontract.SkillReadinessOptionsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillReadinessHandler) QuerySkillReadiness(ctx context.Context, request apicontract.QuerySkillReadinessRequestObject) (apicontract.QuerySkillReadinessResponseObject, error) {
	out := &apicontract.QuerySkillReadiness200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(readinessBusinessError(service.ErrSkillReadinessRequest))
	}
	requestCtx, _ := strictGinRequestDetails(ctx)
	data, err := h.querySkillReadiness(requestCtx, readinessRequest(*request.Body))
	if err != nil {
		return out, out.FromBusinessError(readinessBusinessError(err))
	}
	return out, out.FromSkillReadinessQuerySuccess(apicontract.SkillReadinessQuerySuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillReadinessHandler) GetSkillReadinessUserCharacters(ctx context.Context, request apicontract.GetSkillReadinessUserCharactersRequestObject) (apicontract.GetSkillReadinessUserCharactersResponseObject, error) {
	out := &apicontract.GetSkillReadinessUserCharacters200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(readinessBusinessError(service.ErrSkillReadinessRequest))
	}
	requestCtx, _ := strictGinRequestDetails(ctx)
	data, err := h.getSkillReadinessUserCharacters(requestCtx, request.UserId, readinessRequest(*request.Body))
	if err != nil {
		return out, out.FromBusinessError(readinessBusinessError(err))
	}
	return out, out.FromSkillReadinessUserCharactersSuccess(apicontract.SkillReadinessUserCharactersSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SkillReadinessHandler) GetSkillReadinessCharacterSkills(ctx context.Context, request apicontract.GetSkillReadinessCharacterSkillsRequestObject) (apicontract.GetSkillReadinessCharacterSkillsResponseObject, error) {
	out := &apicontract.GetSkillReadinessCharacterSkills200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(readinessBusinessError(service.ErrSkillReadinessRequest))
	}
	requestCtx, _ := strictGinRequestDetails(ctx)
	data, err := h.getSkillReadinessCharacterSkills(requestCtx, request.CharacterId, readinessRequest(*request.Body))
	if err != nil {
		return out, out.FromBusinessError(readinessBusinessError(err))
	}
	return out, out.FromSkillReadinessCharacterSkillsSuccess(apicontract.SkillReadinessCharacterSkillsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}
