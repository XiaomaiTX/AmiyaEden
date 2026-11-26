package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"time"
)

type CaptainRefundHandler struct{ svc captainRefundService }

type captainRefundService interface {
	Options(userID uint) (service.CaptainRefundOptions, error)
	Calculate(userID uint, req service.CaptainRefundRequest, now time.Time) (*service.CaptainRefundCalculation, error)
	Create(userID uint, req service.CaptainRefundRequest) (*service.CaptainRefundView, error)
	List(userID uint, admin bool, page, size int) ([]service.CaptainRefundView, int64, error)
	Detail(userID, id uint, admin bool) (*service.CaptainRefundView, error)
	Review(userID, id uint, approve bool, reason string) error
	Settings() service.CaptainRefundSettings
	UpdateSettings(userID uint, settings service.CaptainRefundSettings) (service.CaptainRefundSettings, error)
}

func NewCaptainRefundHandler() *CaptainRefundHandler {
	return &CaptainRefundHandler{service.NewCaptainRefundService()}
}

func (h *CaptainRefundHandler) getCaptainRefundOptions(userID uint) (apicontract.CaptainRefundOptions, error) {
	v, err := h.svc.Options(userID)
	if err != nil {
		return apicontract.CaptainRefundOptions{}, err
	}
	characters := mapRefundRows(v.Characters, func(character service.CaptainRefundCharacter) apicontract.CaptainRefundCharacter {
		return apicontract.CaptainRefundCharacter{CharacterId: character.CharacterID, CharacterName: character.CharacterName}
	})
	return apicontract.CaptainRefundOptions{Characters: characters, Settings: refundSettingsDTO(v.Settings)}, nil
}

func (h *CaptainRefundHandler) previewCaptainRefund(userID uint, req apicontract.CaptainRefundInput) (apicontract.CaptainRefundCalculation, error) {
	v, err := h.svc.Calculate(userID, refundRequestFromInput(req), time.Now())
	if err != nil {
		return apicontract.CaptainRefundCalculation{}, err
	}
	return refundCalculationDTO(*v), nil
}

func (h *CaptainRefundHandler) createCaptainRefund(userID uint, req apicontract.CaptainRefundInput) (apicontract.CaptainRefundApplication, error) {
	v, err := h.svc.Create(userID, refundRequestFromInput(req))
	if err != nil {
		return apicontract.CaptainRefundApplication{}, err
	}
	return refundApplicationDTO(*v), nil
}

func (h *CaptainRefundHandler) listCaptainRefunds(userID uint, admin bool, current, size *int) (apicontract.CaptainRefundPage, error) {
	page, pageSize := newbroPagination(current, size, 200, 200)
	rows, total, err := h.svc.List(userID, admin, page, pageSize)
	if err != nil {
		return apicontract.CaptainRefundPage{}, err
	}
	return apicontract.CaptainRefundPage{
		List: mapRefundRows(rows, refundApplicationDTO), Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func (h *CaptainRefundHandler) detailCaptainRefund(userID, id uint, admin bool) (apicontract.CaptainRefundApplication, error) {
	v, err := h.svc.Detail(userID, id, admin)
	if err != nil {
		return apicontract.CaptainRefundApplication{}, err
	}
	return refundApplicationDTO(*v), nil
}

func (h *CaptainRefundHandler) approveCaptainRefund(userID, id uint) error {
	return h.svc.Review(userID, id, true, "")
}

func (h *CaptainRefundHandler) rejectCaptainRefund(userID, id uint, req apicontract.CaptainRefundReviewInput) error {
	return h.svc.Review(userID, id, false, req.Reason)
}

func (h *CaptainRefundHandler) getCaptainRefundSettings() apicontract.CaptainRefundSettings {
	return refundSettingsDTO(h.svc.Settings())
}

func (h *CaptainRefundHandler) updateCaptainRefundSettings(userID uint, req apicontract.CaptainRefundSettingsInput) (apicontract.CaptainRefundSettings, error) {
	v, err := h.svc.UpdateSettings(userID, service.CaptainRefundSettings{
		MaxTotalSP: req.MaxTotalSP, MaxAgeDays: req.MaxAgeDays, TaxRate: req.TaxRate, AllowedSystemIDs: req.AllowedSystemIDs,
	})
	return refundSettingsDTO(v), err
}

func refundRequestFromInput(req apicontract.CaptainRefundInput) service.CaptainRefundRequest {
	return service.CaptainRefundRequest{
		CaptainCharacterID: req.CaptainCharacterId, MemberNames: req.MemberNames,
		SystemID: req.SystemId, Start: req.Start, End: req.End,
	}
}

func refundSettingsDTO(settings service.CaptainRefundSettings) apicontract.CaptainRefundSettings {
	return apicontract.CaptainRefundSettings{
		MaxTotalSP: settings.MaxTotalSP, MaxAgeDays: settings.MaxAgeDays,
		TaxRate: settings.TaxRate, AllowedSystemIDs: settings.AllowedSystemIDs,
	}
}

func refundApplicationDTO(v service.CaptainRefundView) apicontract.CaptainRefundApplication {
	return apicontract.CaptainRefundApplication{
		Id: v.ID, UserId: v.UserID, Status: apicontract.CaptainRefundStatus(v.Status), Reward: v.Reward,
		CreatedAt: v.CreatedAt, ReviewedAt: v.ReviewedAt, ReviewedBy: v.ReviewedBy,
		RejectReason: v.RejectReason, WalletRefId: v.WalletRefID, Calculation: refundCalculationDTO(v.Calculation),
	}
}

func refundCalculationDTO(v service.CaptainRefundCalculation) apicontract.CaptainRefundCalculation {
	return apicontract.CaptainRefundCalculation{
		Request: apicontract.CaptainRefundRequest{
			CaptainCharacterId: v.Request.CaptainCharacterID, MemberNames: v.Request.MemberNames,
			SystemId: v.Request.SystemID, Start: v.Request.Start, End: v.Request.End,
		},
		Settings: refundSettingsDTO(v.Settings), IgnoredNames: v.IgnoredNames,
		Members: mapRefundRows(v.Members, refundMemberDTO),
		Journals: mapRefundRows(v.Journals, func(j service.RefundJournal) apicontract.CaptainRefundJournal {
			return apicontract.CaptainRefundJournal{
				CharacterId: j.CharacterID, JournalId: j.JournalID, Date: j.Date, Amount: j.Amount,
				MatchJournalId: j.MatchJournalID, MatchDate: j.MatchDate,
			}
		}),
		Conflicts: mapRefundRows(v.Conflicts, func(r service.RefundConflict) apicontract.CaptainRefundConflict {
			return apicontract.CaptainRefundConflict{
				CharacterId: r.CharacterID, JournalId: r.JournalID, JournalAt: r.JournalAt, RefundId: r.RefundID,
			}
		}),
		Bounty: v.Bounty, Reward: v.Reward, HasNewbro: v.HasNewbro, CanSubmit: v.CanSubmit,
	}
}

func refundMemberDTO(member service.RefundMember) apicontract.CaptainRefundMember {
	e := member.Eligibility
	return apicontract.CaptainRefundMember{
		CharacterId: member.CharacterID, CharacterName: member.CharacterName, UserId: member.UserID,
		IsCaptain: member.IsCaptain, Bounty: member.Bounty, WalletUpdatedAt: member.WalletUpdatedAt,
		Eligibility: apicontract.CaptainRefundEligibility{
			IsCurrentlyNewbro: e.IsCurrentlyNewbro, DisqualifiedReason: e.DisqualifiedReason, TotalSP: e.TotalSP,
			OldestBirthday: e.OldestBirthday, MaxTotalSP: e.MaxTotalSP, MaxAgeDays: e.MaxAgeDays, EvaluatedAt: e.EvaluatedAt,
			Characters: mapRefundRows(e.Characters, func(c service.NewbroCharacterSnapshot) apicontract.CaptainRefundEligibilityCharacter {
				return apicontract.CaptainRefundEligibilityCharacter{
					CharacterId: c.CharacterID, TotalSP: c.TotalSP, HasSkillData: c.HasSkillData, Birthday: c.Birthday,
				}
			}),
		},
	}
}

// Preserve nil slices found in stored snapshots without converting empty arrays to null.
func mapRefundRows[S, T any](rows []S, convert func(S) T) []T {
	if rows == nil {
		return nil
	}
	out := make([]T, 0, len(rows))
	for _, row := range rows {
		out = append(out, convert(row))
	}
	return out
}

func (h *CaptainRefundHandler) GetCaptainRefundOptions(ctx context.Context, _ apicontract.GetCaptainRefundOptionsRequestObject) (apicontract.GetCaptainRefundOptionsResponseObject, error) {
	out := &apicontract.GetCaptainRefundOptions200JSONResponse{}
	data, err := h.getCaptainRefundOptions(newbroStrictUserID(ctx))
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromCaptainRefundOptionsSuccess(apicontract.CaptainRefundOptionsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *CaptainRefundHandler) PreviewCaptainRefund(ctx context.Context, request apicontract.PreviewCaptainRefundRequestObject) (apicontract.PreviewCaptainRefundResponseObject, error) {
	out := &apicontract.PreviewCaptainRefund200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(newbroInvalidRequest())
	}
	data, err := h.previewCaptainRefund(newbroStrictUserID(ctx), *request.Body)
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromCaptainRefundCalculationSuccess(apicontract.CaptainRefundCalculationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *CaptainRefundHandler) CreateCaptainRefund(ctx context.Context, request apicontract.CreateCaptainRefundRequestObject) (apicontract.CreateCaptainRefundResponseObject, error) {
	out := &apicontract.CreateCaptainRefund200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(newbroInvalidRequest())
	}
	data, err := h.createCaptainRefund(newbroStrictUserID(ctx), *request.Body)
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromCaptainRefundApplicationSuccess(apicontract.CaptainRefundApplicationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *CaptainRefundHandler) ListMyCaptainRefunds(ctx context.Context, request apicontract.ListMyCaptainRefundsRequestObject) (apicontract.ListMyCaptainRefundsResponseObject, error) {
	out := &apicontract.ListMyCaptainRefunds200JSONResponse{}
	data, err := h.listCaptainRefunds(newbroStrictUserID(ctx), false, request.Params.Current, request.Params.Size)
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromCaptainRefundPageSuccess(apicontract.CaptainRefundPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *CaptainRefundHandler) ListAdminCaptainRefunds(ctx context.Context, request apicontract.ListAdminCaptainRefundsRequestObject) (apicontract.ListAdminCaptainRefundsResponseObject, error) {
	out := &apicontract.ListAdminCaptainRefunds200JSONResponse{}
	data, err := h.listCaptainRefunds(newbroStrictUserID(ctx), true, request.Params.Current, request.Params.Size)
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromCaptainRefundPageSuccess(apicontract.CaptainRefundPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *CaptainRefundHandler) GetMyCaptainRefund(ctx context.Context, request apicontract.GetMyCaptainRefundRequestObject) (apicontract.GetMyCaptainRefundResponseObject, error) {
	out := &apicontract.GetMyCaptainRefund200JSONResponse{}
	data, err := h.detailCaptainRefund(newbroStrictUserID(ctx), request.Id, false)
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromCaptainRefundApplicationSuccess(apicontract.CaptainRefundApplicationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *CaptainRefundHandler) GetAdminCaptainRefund(ctx context.Context, request apicontract.GetAdminCaptainRefundRequestObject) (apicontract.GetAdminCaptainRefundResponseObject, error) {
	out := &apicontract.GetAdminCaptainRefund200JSONResponse{}
	data, err := h.detailCaptainRefund(newbroStrictUserID(ctx), request.Id, true)
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromCaptainRefundApplicationSuccess(apicontract.CaptainRefundApplicationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *CaptainRefundHandler) ApproveCaptainRefund(ctx context.Context, request apicontract.ApproveCaptainRefundRequestObject) (apicontract.ApproveCaptainRefundResponseObject, error) {
	out := &apicontract.ApproveCaptainRefund200JSONResponse{}
	// Approval has always ignored the optional request body, including its reason.
	if err := h.approveCaptainRefund(newbroStrictUserID(ctx), request.Id); err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromCaptainRefundReviewSuccess(captainRefundReviewSuccess())
}

func (h *CaptainRefundHandler) RejectCaptainRefund(ctx context.Context, request apicontract.RejectCaptainRefundRequestObject) (apicontract.RejectCaptainRefundResponseObject, error) {
	out := &apicontract.RejectCaptainRefund200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(newbroInvalidRequest())
	}
	if err := h.rejectCaptainRefund(newbroStrictUserID(ctx), request.Id, *request.Body); err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromCaptainRefundReviewSuccess(captainRefundReviewSuccess())
}

func captainRefundReviewSuccess() apicontract.CaptainRefundReviewSuccess {
	return apicontract.CaptainRefundReviewSuccess{Code: response.CodeOK, Msg: "success", Data: map[string]any{}}
}

func (h *CaptainRefundHandler) GetCaptainRefundSettings(_ context.Context, _ apicontract.GetCaptainRefundSettingsRequestObject) (apicontract.GetCaptainRefundSettingsResponseObject, error) {
	return apicontract.GetCaptainRefundSettings200JSONResponse{Code: response.CodeOK, Msg: "success", Data: h.getCaptainRefundSettings()}, nil
}

func (h *CaptainRefundHandler) UpdateCaptainRefundSettings(ctx context.Context, request apicontract.UpdateCaptainRefundSettingsRequestObject) (apicontract.UpdateCaptainRefundSettingsResponseObject, error) {
	out := &apicontract.UpdateCaptainRefundSettings200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "invalid settings"))
	}
	data, err := h.updateCaptainRefundSettings(newbroStrictUserID(ctx), *request.Body)
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromCaptainRefundSettingsSuccess(apicontract.CaptainRefundSettingsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}
