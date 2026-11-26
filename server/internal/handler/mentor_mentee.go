package handler

import (
	"context"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
)

type MentorMenteeHandler struct {
	svc *service.MentorService
}

func NewMentorMenteeHandler() *MentorMenteeHandler {
	return &MentorMenteeHandler{svc: service.NewMentorService()}
}

func (h *MentorMenteeHandler) listMentorCandidates(userID uint) (apicontract.MentorCandidateList, error) {
	result, err := h.svc.ListMentorCandidates(userID)
	if err != nil {
		return nil, err
	}
	return mapContractRows(result, mentorCandidateDTO), nil
}

func (h *MentorMenteeHandler) getMyMentorStatus(userID uint) (apicontract.MentorMyStatus, error) {
	result, err := h.svc.GetMyMenteeStatus(userID)
	if err != nil {
		return apicontract.MentorMyStatus{}, err
	}
	return mentorMyStatusDTO(*result), nil
}

func (h *MentorMenteeHandler) applyForMentor(userID uint, req *apicontract.MentorApplyInput) (apicontract.MentorApplyResult, error) {
	if req == nil || req.MentorUserID == 0 {
		return apicontract.MentorApplyResult{}, errMentorInvalidRequest
	}
	result, err := h.svc.ApplyForMentor(userID, req.MentorUserID)
	if err != nil {
		return apicontract.MentorApplyResult{}, err
	}
	return mentorApplyDTO(*result), nil
}

func mentorCandidateDTO(value service.MentorCandidate) apicontract.MentorCandidate {
	return apicontract.MentorCandidate{
		MentorUserID:        value.MentorUserID,
		MentorCharacterID:   value.MentorCharacterID,
		MentorCharacterName: value.MentorCharacterName,
		MentorNickname:      value.MentorNickname,
		MentorQQ:            value.MentorQQ,
		MentorDiscordID:     value.MentorDiscordID,
		ActiveMenteeCount:   value.ActiveMenteeCount,
		LastOnlineAt:        value.LastOnlineAt,
	}
}

func mentorApplyDTO(value model.MentorMenteeRelationship) apicontract.MentorApplyResult {
	return apicontract.MentorApplyResult{
		ID:                              value.ID,
		CreatedAt:                       value.CreatedAt,
		UpdatedAt:                       value.UpdatedAt,
		MenteeUserID:                    value.MenteeUserID,
		MenteePrimaryCharacterIDAtStart: value.MenteePrimaryCharacterIDAtStart,
		MentorUserID:                    value.MentorUserID,
		Status:                          value.Status,
		AppliedAt:                       value.AppliedAt,
		RespondedAt:                     value.RespondedAt,
		RevokedAt:                       value.RevokedAt,
		RevokedBy:                       value.RevokedBy,
		GraduatedAt:                     value.GraduatedAt,
	}
}

func mentorRelationshipDTO(value service.MentorRelationshipView) apicontract.MentorRelationshipView {
	return apicontract.MentorRelationshipView{
		ID:                  value.ID,
		MenteeUserID:        value.MenteeUserID,
		MentorUserID:        value.MentorUserID,
		Status:              value.Status,
		AppliedAt:           value.AppliedAt,
		RespondedAt:         value.RespondedAt,
		RevokedAt:           value.RevokedAt,
		GraduatedAt:         value.GraduatedAt,
		MentorCharacterID:   value.MentorCharacterID,
		MentorCharacterName: value.MentorCharacterName,
		MentorNickname:      value.MentorNickname,
		MentorQQ:            value.MentorQQ,
		MentorDiscordID:     value.MentorDiscordID,
		MenteeCharacterID:   value.MenteeCharacterID,
		MenteeCharacterName: value.MenteeCharacterName,
		MenteeNickname:      value.MenteeNickname,
	}
}

func mentorMyStatusDTO(value service.MenteeMyStatusResponse) apicontract.MentorMyStatus {
	var relationship *apicontract.MentorRelationshipView
	if value.CurrentRelationship != nil {
		mapped := mentorRelationshipDTO(*value.CurrentRelationship)
		relationship = &mapped
	}
	return apicontract.MentorMyStatus{
		IsEligible: value.IsEligible, DisqualifiedReason: value.DisqualifiedReason,
		CurrentRelationship: relationship,
	}
}

func (h *MentorMenteeHandler) ListMentorCandidates(ctx context.Context, _ apicontract.ListMentorCandidatesRequestObject) (apicontract.ListMentorCandidatesResponseObject, error) {
	data, err := h.listMentorCandidates(mentorUserID(ctx))
	var out apicontract.ListMentorCandidates200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorCandidateListSuccess(apicontract.MentorCandidateListSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *MentorMenteeHandler) GetMyMentorStatus(ctx context.Context, _ apicontract.GetMyMentorStatusRequestObject) (apicontract.GetMyMentorStatusResponseObject, error) {
	data, err := h.getMyMentorStatus(mentorUserID(ctx))
	var out apicontract.GetMyMentorStatus200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorMyStatusSuccess(apicontract.MentorMyStatusSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *MentorMenteeHandler) ApplyForMentor(ctx context.Context, request apicontract.ApplyForMentorRequestObject) (apicontract.ApplyForMentorResponseObject, error) {
	data, err := h.applyForMentor(mentorUserID(ctx), request.Body)
	var out apicontract.ApplyForMentor200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(mentorBusinessError(err))
	}
	return &out, out.FromMentorApplyResultSuccess(apicontract.MentorApplyResultSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}
