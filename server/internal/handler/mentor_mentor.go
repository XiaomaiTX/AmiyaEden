package handler

import (
	"context"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
)

type MentorMentorHandler struct {
	svc       *service.MentorService
	rewardSvc *service.MentorRewardService
}

func NewMentorMentorHandler() *MentorMentorHandler {
	return &MentorMentorHandler{
		svc:       service.NewMentorService(),
		rewardSvc: service.NewMentorRewardService(),
	}
}

func (h *MentorMentorHandler) listMentorApplications(userID uint) (apicontract.MentorMenteeList, error) {
	result, err := h.svc.ListPendingApplications(userID)
	if err != nil {
		return nil, err
	}
	return mapContractRows(result, mentorMenteeDTO), nil
}

func (h *MentorMentorHandler) listMentorMentees(userID uint, params apicontract.ListMentorMenteesParams) (apicontract.MentorMenteePage, error) {
	page, pageSize := normalizePagination(mentorParamValue(params.Current), mentorParamValue(params.Size), 20, 100)
	status := "active"
	if params.Status != nil {
		status = *params.Status
	}
	result, total, err := h.svc.ListMyMentees(userID, status, page, pageSize)
	if err != nil {
		return apicontract.MentorMenteePage{}, err
	}
	return apicontract.MentorMenteePage{
		List: mapContractRows(result, mentorMenteeDTO), Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func applyMentorRelationshipAction(userID uint, req *apicontract.MentorRelationshipActionInput, action func(uint, uint) error) error {
	if req == nil || req.RelationshipID == 0 {
		return errMentorInvalidRequest
	}
	return action(userID, req.RelationshipID)
}

func mentorMenteeDTO(value service.MentorMenteeListItem) apicontract.MentorMenteeItem {
	return apicontract.MentorMenteeItem{
		RelationshipID:          value.RelationshipID,
		MenteeUserID:            value.MenteeUserID,
		MenteeCharacterID:       value.MenteeCharacterID,
		MenteeCharacterName:     value.MenteeCharacterName,
		MenteeNickname:          value.MenteeNickname,
		MenteeQQ:                value.MenteeQQ,
		MenteeDiscordID:         value.MenteeDiscordID,
		MenteeTotalSP:           value.MenteeTotalSP,
		MenteeTotalPap:          value.MenteeTotalPap,
		MenteeDaysActive:        value.MenteeDaysActive,
		Status:                  value.Status,
		AppliedAt:               value.AppliedAt,
		RespondedAt:             value.RespondedAt,
		GraduatedAt:             value.GraduatedAt,
		DistributedStages:       value.DistributedStages,
		DistributedRewardAmount: value.DistributedRewardAmount,
	}
}

func (h *MentorMentorHandler) ListMentorApplications(ctx context.Context, _ apicontract.ListMentorApplicationsRequestObject) (apicontract.ListMentorApplicationsResponseObject, error) {
	data, err := h.listMentorApplications(mentorUserID(ctx))
	var out apicontract.ListMentorApplications200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorMenteeListSuccess(apicontract.MentorMenteeListSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *MentorMentorHandler) ListMentorMentees(ctx context.Context, request apicontract.ListMentorMenteesRequestObject) (apicontract.ListMentorMenteesResponseObject, error) {
	data, err := h.listMentorMentees(mentorUserID(ctx), request.Params)
	var out apicontract.ListMentorMentees200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorMenteePageSuccess(apicontract.MentorMenteePageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *MentorMentorHandler) GetMentorDashboardRewardStages(_ context.Context, _ apicontract.GetMentorDashboardRewardStagesRequestObject) (apicontract.GetMentorDashboardRewardStagesResponseObject, error) {
	data, err := listMentorRewardStages(h.rewardSvc)
	var out apicontract.GetMentorDashboardRewardStages200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorRewardStageListSuccess(mentorRewardStageListSuccess(data))
}

func (h *MentorMentorHandler) AcceptMentorApplication(ctx context.Context, request apicontract.AcceptMentorApplicationRequestObject) (apicontract.AcceptMentorApplicationResponseObject, error) {
	err := applyMentorRelationshipAction(mentorUserID(ctx), request.Body, h.svc.AcceptApplication)
	var out apicontract.AcceptMentorApplication200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorEmptySuccess(mentorEmptySuccess())
}

func (h *MentorMentorHandler) RejectMentorApplication(ctx context.Context, request apicontract.RejectMentorApplicationRequestObject) (apicontract.RejectMentorApplicationResponseObject, error) {
	err := applyMentorRelationshipAction(mentorUserID(ctx), request.Body, h.svc.RejectApplication)
	var out apicontract.RejectMentorApplication200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorEmptySuccess(mentorEmptySuccess())
}
