package handler

import (
	"context"
	"errors"
	"time"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

type MentorAdminHandler struct {
	svc       *service.MentorService
	rewardSvc *service.MentorRewardService
}

func NewMentorAdminHandler() *MentorAdminHandler {
	return &MentorAdminHandler{
		svc:       service.NewMentorService(),
		rewardSvc: service.NewMentorRewardService(),
	}
}

func (h *MentorAdminHandler) adminListMentorRelationships(params apicontract.AdminListMentorRelationshipsParams) (apicontract.MentorRelationshipPage, error) {
	page, pageSize := normalizePagination(mentorParamValue(params.Current), mentorParamValue(params.Size), 20, 200)
	result, total, err := h.svc.AdminListAllRelationships(mentorParamValue(params.Status), mentorParamValue(params.Keyword), page, pageSize)
	if err != nil {
		return apicontract.MentorRelationshipPage{}, err
	}
	return apicontract.MentorRelationshipPage{
		List: mapContractRows(result, mentorRelationshipDTO), Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func (h *MentorAdminHandler) adminListMentorRewardDistributions(params apicontract.AdminListMentorRewardDistributionsParams) (apicontract.MentorDistributionPage, error) {
	page, pageSize := normalizeLedgerPagination(mentorParamValue(params.Current), mentorParamValue(params.Size))
	result, total, err := h.rewardSvc.ListAdminRewardDistributions(page, pageSize, mentorParamValue(params.Keyword))
	if err != nil {
		return apicontract.MentorDistributionPage{}, err
	}
	return apicontract.MentorDistributionPage{
		List: mapContractRows(result, mentorDistributionDTO), Total: total, Page: page, PageSize: pageSize,
	}, nil
}

var errMentorInvalidRequest = errors.New("invalid request")

func mentorErrorCode(err error) int {
	if errors.Is(err, errMentorInvalidRequest) {
		return response.CodeParamError
	}
	return response.CodeBizError
}

func mentorParamValue[T any](param *T) T {
	if param == nil {
		var zero T
		return zero
	}
	return *param
}

func listMentorRewardStages(svc *service.MentorRewardService) (apicontract.MentorRewardStageList, error) {
	stages, err := svc.GetStages()
	if err != nil {
		return nil, err
	}
	return mapContractRows(stages, mentorRewardStageDTO), nil
}

func (h *MentorAdminHandler) adminUpdateMentorRewardStages(req *apicontract.MentorRewardStagesInput) (apicontract.MentorRewardStageList, error) {
	if req == nil || req.Stages == nil {
		return nil, errMentorInvalidRequest
	}
	stages, err := h.rewardSvc.UpdateStages(mapContractRows(req.Stages, mentorRewardStageInput))
	if err != nil {
		return nil, err
	}
	return mapContractRows(stages, mentorRewardStageDTO), nil
}

func (h *MentorAdminHandler) adminProcessMentorRewards() (apicontract.MentorRewardProcessResult, error) {
	result, err := h.rewardSvc.ProcessRewards(time.Now())
	if err != nil {
		return apicontract.MentorRewardProcessResult{}, err
	}
	return mentorRewardProcessDTO(*result), nil
}

func mentorDistributionDTO(value service.MentorRewardDistributionView) apicontract.MentorDistribution {
	return apicontract.MentorDistribution{
		ID:                  value.ID,
		RelationshipID:      value.RelationshipID,
		StageID:             value.StageID,
		StageOrder:          value.StageOrder,
		MentorUserID:        value.MentorUserID,
		MentorCharacterName: value.MentorCharacterName,
		MentorNickname:      value.MentorNickname,
		MenteeUserID:        value.MenteeUserID,
		MenteeCharacterName: value.MenteeCharacterName,
		MenteeNickname:      value.MenteeNickname,
		RewardAmount:        value.RewardAmount,
		DistributedAt:       value.DistributedAt,
		WalletRefID:         value.WalletRefID,
	}
}

func mentorRewardStageDTO(value model.MentorRewardStage) apicontract.MentorRewardStage {
	return apicontract.MentorRewardStage{
		ID:            value.ID,
		CreatedAt:     value.CreatedAt,
		UpdatedAt:     value.UpdatedAt,
		StageOrder:    value.StageOrder,
		Name:          value.Name,
		ConditionType: value.ConditionType,
		Threshold:     value.Threshold,
		RewardAmount:  value.RewardAmount,
	}
}

func mentorRewardProcessDTO(value service.MentorRewardProcessResult) apicontract.MentorRewardProcessResult {
	return apicontract.MentorRewardProcessResult{
		ProcessedRelationships: value.ProcessedRelationships,
		RewardsDistributed:     value.RewardsDistributed,
		TotalCoinAwarded:       value.TotalCoinAwarded,
		GraduatedCount:         value.GraduatedCount,
	}
}

func mentorRewardStageInput(value apicontract.MentorRewardStageInput) service.MentorRewardStageInput {
	return service.MentorRewardStageInput{
		StageOrder:    value.StageOrder,
		Name:          value.Name,
		ConditionType: value.ConditionType,
		Threshold:     value.Threshold,
		RewardAmount:  value.RewardAmount,
	}
}

func (h *MentorAdminHandler) AdminListMentorRelationships(_ context.Context, request apicontract.AdminListMentorRelationshipsRequestObject) (apicontract.AdminListMentorRelationshipsResponseObject, error) {
	data, err := h.adminListMentorRelationships(request.Params)
	var out apicontract.AdminListMentorRelationships200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorRelationshipPageSuccess(apicontract.MentorRelationshipPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *MentorAdminHandler) AdminListMentorRewardDistributions(_ context.Context, request apicontract.AdminListMentorRewardDistributionsRequestObject) (apicontract.AdminListMentorRewardDistributionsResponseObject, error) {
	data, err := h.adminListMentorRewardDistributions(request.Params)
	var out apicontract.AdminListMentorRewardDistributions200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorDistributionPageSuccess(apicontract.MentorDistributionPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *MentorAdminHandler) AdminRevokeMentorRelationship(ctx context.Context, request apicontract.AdminRevokeMentorRelationshipRequestObject) (apicontract.AdminRevokeMentorRelationshipResponseObject, error) {
	err := applyMentorRelationshipAction(mentorUserID(ctx), request.Body, h.svc.AdminRevokeRelationship)
	var out apicontract.AdminRevokeMentorRelationship200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorEmptySuccess(mentorEmptySuccess())
}

func (h *MentorAdminHandler) AdminGetMentorRewardStages(_ context.Context, _ apicontract.AdminGetMentorRewardStagesRequestObject) (apicontract.AdminGetMentorRewardStagesResponseObject, error) {
	data, err := listMentorRewardStages(h.rewardSvc)
	var out apicontract.AdminGetMentorRewardStages200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorRewardStageListSuccess(mentorRewardStageListSuccess(data))
}

func (h *MentorAdminHandler) AdminUpdateMentorRewardStages(_ context.Context, request apicontract.AdminUpdateMentorRewardStagesRequestObject) (apicontract.AdminUpdateMentorRewardStagesResponseObject, error) {
	data, err := h.adminUpdateMentorRewardStages(request.Body)
	var out apicontract.AdminUpdateMentorRewardStages200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorRewardStageListSuccess(mentorRewardStageListSuccess(data))
}

func (h *MentorAdminHandler) AdminProcessMentorRewards(_ context.Context, _ apicontract.AdminProcessMentorRewardsRequestObject) (apicontract.AdminProcessMentorRewardsResponseObject, error) {
	data, err := h.adminProcessMentorRewards()
	var out apicontract.AdminProcessMentorRewards200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorRewardProcessResultSuccess(apicontract.MentorRewardProcessResultSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func mentorUserID(ctx context.Context) uint {
	return middleware.GetUserID(ctx.(*gin.Context))
}

func mentorBusinessError(err error) apicontract.BusinessError {
	return contractBusinessError(mentorErrorCode(err), err.Error())
}

func mentorEmptySuccess() apicontract.MentorEmptySuccess {
	return apicontract.MentorEmptySuccess{Code: response.CodeOK, Msg: "success", Data: apicontract.MentorEmpty{}}
}

func mentorRewardStageListSuccess(data apicontract.MentorRewardStageList) apicontract.MentorRewardStageListSuccess {
	return apicontract.MentorRewardStageListSuccess{Code: response.CodeOK, Msg: "success", Data: data}
}
