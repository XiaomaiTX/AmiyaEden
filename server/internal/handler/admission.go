package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"strconv"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

type AdmissionHandler struct{ svc *service.AdmissionService }

func NewAdmissionHandler() *AdmissionHandler {
	return &AdmissionHandler{svc: service.NewAdmissionService()}
}

func classifyAdmissionFailure(err error) *contractOperationFailure {
	code := response.CodeBizError
	if errors.Is(err, service.ErrAdmissionDenied) {
		code = response.CodeForbidden
	}
	return &contractOperationFailure{code, err.Error()}
}

func (h *AdmissionHandler) loadAdmissionContext(userID uint) (apicontract.AdmissionContext, error) {
	result, err := h.svc.Context(userID)
	if err != nil {
		return apicontract.AdmissionContext{}, service.ErrAdmissionInternal
	}
	return admissionContextDTO(*result), nil
}

func (h *AdmissionHandler) listAdmissions(userID uint, reviewer bool, params apicontract.ListMyAdmissionsParams) (apicontract.AdmissionPage, *contractOperationFailure) {
	page, size := normalizePagination(admissionValue(params.Current), admissionValue(params.Size), 20, 100)
	var target int64
	if raw := admissionValue(params.TargetCorporationId); raw != "" {
		var err error
		target, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || target <= 0 {
			return apicontract.AdmissionPage{}, &contractOperationFailure{response.CodeParamError, service.ErrAdmissionInvalid.Error()}
		}
	}
	list, total, err := h.svc.List(userID, reviewer, admissionValue(params.Status), admissionValue(params.Kind), target, page, size)
	if err != nil {
		return apicontract.AdmissionPage{}, classifyAdmissionFailure(err)
	}
	return apicontract.AdmissionPage{
		List: admissionMap(list, admissionListItemDTO), Total: total, Page: page, PageSize: size,
	}, nil
}

func (h *AdmissionHandler) loadAdmission(userID, id uint) (apicontract.AdmissionDetail, error) {
	result, err := h.svc.Detail(userID, id)
	if err != nil {
		return apicontract.AdmissionDetail{}, err
	}
	return admissionDetailDTO(*result), nil
}

func (h *AdmissionHandler) loadAdmissionEvidence(userID, id uint, kind apicontract.AdmissionEvidenceKind) ([]byte, error) {
	result, err := h.svc.Evidence(userID, id, string(kind))
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

const admissionMultipartMemory = 5 << 20

func decodeAdmissionForm(form *multipart.Form) (service.AdmissionWriteInput, map[string][]byte, error) {
	var input service.AdmissionWriteInput
	var payload apicontract.AdmissionWriteInput
	values := form.Value["payload"]
	if len(values) == 0 || json.Unmarshal([]byte(values[0]), &payload) != nil {
		return input, nil, service.ErrAdmissionInvalid
	}
	input = service.AdmissionWriteInput{
		Kind: string(payload.Kind), TargetCorporationID: payload.TargetCorporationId,
		CharacterIDs: payload.CharacterIds, Note: payload.Note, ExpectedVersion: payload.ExpectedVersion,
	}
	uploads := map[string][]byte{}
	for kind, files := range form.File {
		if len(files) != 1 {
			return input, nil, service.ErrAdmissionEvidence
		}
		file, err := files[0].Open()
		if err != nil {
			return input, nil, service.ErrAdmissionEvidence
		}
		data, readErr := io.ReadAll(io.LimitReader(file, service.AdmissionEvidenceMaxBytes+1))
		_ = file.Close()
		if readErr != nil || len(data) > service.AdmissionEvidenceMaxBytes {
			return input, nil, service.ErrAdmissionEvidence
		}
		uploads[kind] = data
	}
	return input, uploads, nil
}

func (h *AdmissionHandler) writeAdmission(userID, id uint, input service.AdmissionWriteInput, uploads map[string][]byte, edit, resubmit bool) (apicontract.AdmissionApplication, error) {
	var result *model.AdmissionApplication
	var err error
	if edit {
		result, err = h.svc.Edit(userID, id, input, uploads, resubmit)
	} else {
		result, err = h.svc.Create(userID, input, uploads)
	}
	if err != nil {
		return apicontract.AdmissionApplication{}, err
	}
	return admissionApplicationDTO(*result), nil
}

func (h *AdmissionHandler) processAdmission(userID, id uint, input apicontract.AdmissionProcessInput, invited bool) (apicontract.AdmissionApplication, error) {
	result, err := h.svc.Process(userID, id, input.ExpectedVersion, invited, input.Reason)
	if err != nil {
		return apicontract.AdmissionApplication{}, err
	}
	return admissionApplicationDTO(*result), nil
}

func admissionValue[T any](value *T) T {
	if value != nil {
		return *value
	}
	var zero T
	return zero
}

func admissionMap[T, U any](values []T, convert func(T) U) []U {
	if values == nil {
		return nil
	}
	result := make([]U, len(values))
	for i, value := range values {
		result[i] = convert(value)
	}
	return result
}

func admissionContextDTO(value service.AdmissionContext) apicontract.AdmissionContext {
	return apicontract.AdmissionContext{
		IsMember: value.IsMember, CanCreateNewMember: value.CanCreateNewMember, CanCreateAlt: value.CanCreateAlt,
		Qq: value.QQ, DiscordId: value.DiscordID, ReviewableCorporationIds: value.ReviewableCorporationIDs,
		Targets: admissionMap(value.Targets, func(target service.AdmissionTarget) apicontract.AdmissionTarget {
			return apicontract.AdmissionTarget{CorporationId: target.CorporationID, Name: target.Name}
		}),
		Characters: admissionMap(value.Characters, func(character service.AdmissionCharacterOption) apicontract.AdmissionCharacterOption {
			return apicontract.AdmissionCharacterOption{
				CharacterId: character.CharacterID, CharacterName: character.CharacterName, CorporationId: character.CorporationID,
			}
		}),
	}
}

func admissionApplicationDTO(value model.AdmissionApplication) apicontract.AdmissionApplication {
	return apicontract.AdmissionApplication{
		Id: value.ID, UserId: value.UserID, Kind: apicontract.AdmissionKind(value.Kind), TargetCorporationId: value.TargetCorporationID,
		Status: apicontract.AdmissionStatus(value.Status), Version: value.Version, ApplicantName: value.ApplicantName,
		PrimaryCharacterId: value.PrimaryCharacterID, Qq: value.QQ, DiscordId: value.DiscordID, Note: value.Note,
		SubmittedAt: value.SubmittedAt, HandledBy: value.HandledBy, HandlerName: value.HandlerName, ClosedAt: value.ClosedAt,
		RejectionReason: value.RejectionReason, EvidenceExpiresAt: value.EvidenceExpiresAt, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
		Characters: admissionMap(value.Characters, func(character model.AdmissionCharacter) apicontract.AdmissionCharacter {
			return apicontract.AdmissionCharacter{
				ApplicationId: character.ApplicationID, CharacterId: character.CharacterID, CharacterName: character.CharacterName,
			}
		}),
	}
}

func admissionListItemDTO(value service.AdmissionListItem) apicontract.AdmissionListItem {
	app := admissionApplicationDTO(value.AdmissionApplication)
	return apicontract.AdmissionListItem{
		Id: app.Id, UserId: app.UserId, Kind: app.Kind, TargetCorporationId: app.TargetCorporationId,
		Status: app.Status, Version: app.Version, ApplicantName: app.ApplicantName, PrimaryCharacterId: app.PrimaryCharacterId,
		Qq: app.Qq, DiscordId: app.DiscordId, Note: app.Note, SubmittedAt: app.SubmittedAt, HandledBy: app.HandledBy,
		HandlerName: app.HandlerName, ClosedAt: app.ClosedAt, RejectionReason: app.RejectionReason,
		EvidenceExpiresAt: app.EvidenceExpiresAt, Characters: app.Characters, CreatedAt: app.CreatedAt, UpdatedAt: app.UpdatedAt,
		Blacklisted: value.Blacklisted,
	}
}

func admissionDetailDTO(value service.AdmissionDetail) apicontract.AdmissionDetail {
	result := apicontract.AdmissionDetail{
		Application: admissionApplicationDTO(*value.Application), CanReview: value.CanReview, IsOwner: value.IsOwner,
		Evidence: admissionMap(value.Evidence, func(evidence model.AdmissionEvidence) apicontract.AdmissionEvidence {
			return apicontract.AdmissionEvidence{
				ApplicationId: evidence.ApplicationID, Kind: apicontract.AdmissionEvidenceKind(evidence.Kind),
				Width: evidence.Width, Height: evidence.Height, ByteSize: evidence.ByteSize, UpdatedAt: evidence.UpdatedAt,
			}
		}),
		Events: admissionMap(value.Events, func(event model.AdmissionProcessingEvent) apicontract.AdmissionEvent {
			return apicontract.AdmissionEvent{
				Id: event.ID, ApplicationId: event.ApplicationID, Version: event.Version, ActorId: event.ActorID,
				ActorName: event.ActorName, Action: event.Action, FromStatus: event.FromStatus,
				ToStatus: apicontract.AdmissionStatus(event.ToStatus), Reason: event.Reason, Snapshot: event.Snapshot, CreatedAt: event.CreatedAt,
			}
		}),
		BlacklistHits: admissionMap(value.BlacklistHits, func(hit service.BlacklistHit) apicontract.AdmissionBlacklistHit {
			return apicontract.AdmissionBlacklistHit{Kind: apicontract.BlacklistKind(hit.Kind), Value: hit.Value, Display: hit.Display, Reason: hit.Reason}
		}),
	}
	if salary := value.Salary; salary != nil {
		result.Salary = &apicontract.AdmissionSalary{
			ApplicationId: salary.ApplicationID, UserId: salary.UserID, Amount: salary.Amount,
			WalletTransactionId: salary.WalletTransactionID, CreatedAt: salary.CreatedAt,
		}
	}
	return result
}

func admissionForbidden(failure *contractOperationFailure) apicontract.ForbiddenJSONResponse {
	return apicontract.ForbiddenJSONResponse{Code: response.CodeForbidden, Msg: failure.message}
}

func (h *AdmissionHandler) GetAdmissionContext(ctx context.Context, _ apicontract.GetAdmissionContextRequestObject) (apicontract.GetAdmissionContextResponseObject, error) {
	data, err := h.loadAdmissionContext(middleware.GetUserID(ctx.(*gin.Context)))
	out := &apicontract.GetAdmissionContext200JSONResponse{}
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromAdmissionContextSuccess(apicontract.AdmissionContextSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AdmissionHandler) ListMyAdmissions(ctx context.Context, request apicontract.ListMyAdmissionsRequestObject) (apicontract.ListMyAdmissionsResponseObject, error) {
	data, failure := h.listAdmissions(middleware.GetUserID(ctx.(*gin.Context)), false, request.Params)
	out := &apicontract.ListMyAdmissions200JSONResponse{}
	if failure != nil {
		if failure.code == response.CodeForbidden {
			return apicontract.ListMyAdmissions403JSONResponse{ForbiddenJSONResponse: admissionForbidden(failure)}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromAdmissionPageSuccess(apicontract.AdmissionPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AdmissionHandler) GetAdmission(ctx context.Context, request apicontract.GetAdmissionRequestObject) (apicontract.GetAdmissionResponseObject, error) {
	data, err := h.loadAdmission(middleware.GetUserID(ctx.(*gin.Context)), request.Id)
	out := &apicontract.GetAdmission200JSONResponse{}
	if err != nil {
		failure := classifyAdmissionFailure(err)
		if failure.code == response.CodeForbidden {
			return apicontract.GetAdmission403JSONResponse{ForbiddenJSONResponse: admissionForbidden(failure)}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromAdmissionDetailSuccess(apicontract.AdmissionDetailSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AdmissionHandler) GetAdmissionEvidence(ctx context.Context, request apicontract.GetAdmissionEvidenceRequestObject) (apicontract.GetAdmissionEvidenceResponseObject, error) {
	data, err := h.loadAdmissionEvidence(middleware.GetUserID(ctx.(*gin.Context)), request.Id, request.Kind)
	headers := apicontract.GetAdmissionEvidence200ResponseHeaders{CacheControl: "private, no-store", XContentTypeOptions: "nosniff"}
	if err != nil {
		failure := classifyAdmissionFailure(err)
		if failure.code == response.CodeForbidden {
			return apicontract.GetAdmissionEvidence403JSONResponse{ForbiddenJSONResponse: admissionForbidden(failure)}, nil
		}
		return apicontract.GetAdmissionEvidence200JSONResponse{Headers: headers, Body: contractBusinessError(failure.code, failure.message)}, nil
	}
	return apicontract.GetAdmissionEvidence200ImagejpegResponse{Headers: headers, Body: bytes.NewReader(data), ContentLength: int64(len(data))}, nil
}

func (h *AdmissionHandler) CreateAdmission(ctx context.Context, request apicontract.CreateAdmissionRequestObject) (apicontract.CreateAdmissionResponseObject, error) {
	data, err := h.writeMultipart(ctx, request.Body, 0, false, false)
	out := &apicontract.CreateAdmission200JSONResponse{}
	if err != nil {
		failure := classifyAdmissionFailure(err)
		if failure.code == response.CodeForbidden {
			return apicontract.CreateAdmission403JSONResponse{ForbiddenJSONResponse: admissionForbidden(failure)}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromAdmissionApplicationSuccess(apicontract.AdmissionApplicationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AdmissionHandler) EditAdmission(ctx context.Context, request apicontract.EditAdmissionRequestObject) (apicontract.EditAdmissionResponseObject, error) {
	data, err := h.writeMultipart(ctx, request.Body, request.Id, true, false)
	out := &apicontract.EditAdmission200JSONResponse{}
	if err != nil {
		failure := classifyAdmissionFailure(err)
		if failure.code == response.CodeForbidden {
			return apicontract.EditAdmission403JSONResponse{ForbiddenJSONResponse: admissionForbidden(failure)}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromAdmissionApplicationSuccess(apicontract.AdmissionApplicationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AdmissionHandler) ResubmitAdmission(ctx context.Context, request apicontract.ResubmitAdmissionRequestObject) (apicontract.ResubmitAdmissionResponseObject, error) {
	data, err := h.writeMultipart(ctx, request.Body, request.Id, true, true)
	out := &apicontract.ResubmitAdmission200JSONResponse{}
	if err != nil {
		failure := classifyAdmissionFailure(err)
		if failure.code == response.CodeForbidden {
			return apicontract.ResubmitAdmission403JSONResponse{ForbiddenJSONResponse: admissionForbidden(failure)}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromAdmissionApplicationSuccess(apicontract.AdmissionApplicationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AdmissionHandler) writeMultipart(ctx context.Context, reader *multipart.Reader, id uint, edit, resubmit bool) (apicontract.AdmissionApplication, error) {
	// The generated binder caps the entire request at 5 MiB; decoding the form
	// still enforces the separate 2 MiB limit on each evidence file.
	if reader == nil {
		return apicontract.AdmissionApplication{}, service.ErrAdmissionEvidence
	}
	form, err := reader.ReadForm(admissionMultipartMemory)
	if err != nil {
		return apicontract.AdmissionApplication{}, service.ErrAdmissionEvidence
	}
	defer func() { _ = form.RemoveAll() }()
	input, uploads, err := decodeAdmissionForm(form)
	if err != nil {
		return apicontract.AdmissionApplication{}, err
	}
	return h.writeAdmission(middleware.GetUserID(ctx.(*gin.Context)), id, input, uploads, edit, resubmit)
}

func (h *AdmissionHandler) ListAdmissionReviews(ctx context.Context, request apicontract.ListAdmissionReviewsRequestObject) (apicontract.ListAdmissionReviewsResponseObject, error) {
	data, failure := h.listAdmissions(middleware.GetUserID(ctx.(*gin.Context)), true, apicontract.ListMyAdmissionsParams(request.Params))
	out := &apicontract.ListAdmissionReviews200JSONResponse{}
	if failure != nil {
		if failure.code == response.CodeForbidden {
			return apicontract.ListAdmissionReviews403JSONResponse{ForbiddenJSONResponse: admissionForbidden(failure)}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromAdmissionPageSuccess(apicontract.AdmissionPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AdmissionHandler) InviteAdmission(ctx context.Context, request apicontract.InviteAdmissionRequestObject) (apicontract.InviteAdmissionResponseObject, error) {
	data, err := h.reviewAdmission(ctx, request.Id, request.Body, true)
	out := &apicontract.InviteAdmission200JSONResponse{}
	if err != nil {
		failure := classifyAdmissionFailure(err)
		if failure.code == response.CodeForbidden {
			return apicontract.InviteAdmission403JSONResponse{ForbiddenJSONResponse: admissionForbidden(failure)}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromAdmissionApplicationSuccess(apicontract.AdmissionApplicationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AdmissionHandler) RejectAdmission(ctx context.Context, request apicontract.RejectAdmissionRequestObject) (apicontract.RejectAdmissionResponseObject, error) {
	data, err := h.reviewAdmission(ctx, request.Id, request.Body, false)
	out := &apicontract.RejectAdmission200JSONResponse{}
	if err != nil {
		failure := classifyAdmissionFailure(err)
		if failure.code == response.CodeForbidden {
			return apicontract.RejectAdmission403JSONResponse{ForbiddenJSONResponse: admissionForbidden(failure)}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromAdmissionApplicationSuccess(apicontract.AdmissionApplicationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AdmissionHandler) reviewAdmission(ctx context.Context, id uint, input *apicontract.AdmissionProcessInput, invited bool) (apicontract.AdmissionApplication, error) {
	if input == nil {
		return apicontract.AdmissionApplication{}, service.ErrAdmissionInvalid
	}
	return h.processAdmission(middleware.GetUserID(ctx.(*gin.Context)), id, *input, invited)
}
