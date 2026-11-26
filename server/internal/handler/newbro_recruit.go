package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

type NewbroRecruitHandler struct {
	linkSvc  *service.RecruitmentLinkService
	entrySvc *service.RecruitmentEntryService
}

func NewNewbroRecruitHandler() *NewbroRecruitHandler {
	return &NewbroRecruitHandler{
		linkSvc:  service.NewRecruitmentLinkService(),
		entrySvc: service.NewRecruitmentEntryService(),
	}
}

func (h *NewbroRecruitHandler) generateLink(userID uint) (apicontract.NewbroGeneratedLink, error) {
	rec, isNew, err := h.linkSvc.GenerateLink(userID, time.Now())
	if err != nil {
		return apicontract.NewbroGeneratedLink{}, err
	}
	return apicontract.NewbroGeneratedLink{Id: rec.ID, Code: rec.Code, GeneratedAt: rec.GeneratedAt, IsNew: isNew}, nil
}

func (h *NewbroRecruitHandler) getMyLinks(userID uint) ([]apicontract.NewbroRecruitLink, error) {
	rows, err := h.linkSvc.GetMyLinks(userID)
	if err != nil {
		return nil, err
	}
	links := make([]apicontract.NewbroRecruitLink, 0, len(rows))
	for _, row := range rows {
		links = append(links, newbroRecruitLink(row))
	}
	return links, nil
}

func (h *NewbroRecruitHandler) getDirectReferralStatus(userID uint) (apicontract.NewbroDirectReferralStatus, error) {
	status, err := h.entrySvc.GetDirectReferralStatus(userID, time.Now())
	if err != nil {
		return apicontract.NewbroDirectReferralStatus{}, err
	}
	return apicontract.NewbroDirectReferralStatus{ShowCard: status.ShowCard, NeedsProfileQQ: status.NeedsProfileQQ}, nil
}

func (h *NewbroRecruitHandler) checkDirectReferrer(userID uint, req apicontract.NewbroQQInput) (apicontract.NewbroDirectReferrer, error) {
	candidate, err := h.entrySvc.LookupDirectReferrer(userID, req.QQ, time.Now())
	if err != nil {
		return apicontract.NewbroDirectReferrer{}, err
	}
	return newbroDirectReferrer(*candidate), nil
}

func (h *NewbroRecruitHandler) confirmDirectReferrer(userID uint, req apicontract.NewbroConfirmReferrerInput) (apicontract.NewbroDirectReferrer, error) {
	confirmed, err := h.entrySvc.ConfirmDirectReferral(userID, req.ReferrerUserId, time.Now())
	if err != nil {
		return apicontract.NewbroDirectReferrer{}, err
	}
	return newbroDirectReferrer(*confirmed), nil
}

func (h *NewbroRecruitHandler) listAdminRecruitLinks(params apicontract.ListAdminRecruitLinksParams) (apicontract.NewbroAdminRecruitLinkPage, error) {
	page, size := newbroPagination(params.Current, params.Size, 20, 100)
	rows, total, err := h.linkSvc.ListAllLinks(page, size)
	if err != nil {
		return apicontract.NewbroAdminRecruitLinkPage{}, err
	}
	links := make([]apicontract.NewbroAdminRecruitLink, 0, len(rows))
	for _, row := range rows {
		link := newbroRecruitLink(row.RecruitLinkRow)
		links = append(links, apicontract.NewbroAdminRecruitLink{
			Id: link.Id, Code: link.Code, Source: link.Source, GeneratedAt: link.GeneratedAt,
			Entries: link.Entries, UserId: row.UserID, UserCharacterName: row.UserCharacterName,
		})
	}
	return apicontract.NewbroAdminRecruitLinkPage{List: links, Total: total, Page: page, PageSize: size}, nil
}

func (h *NewbroRecruitHandler) submitRecruitQQ(code string, req apicontract.NewbroQQInput) (apicontract.NewbroSubmittedQQ, error) {
	qqURL, err := h.entrySvc.SubmitQQ(code, req.QQ, time.Now())
	return apicontract.NewbroSubmittedQQ{QQURL: qqURL}, err
}

func newbroRecruitLink(row service.RecruitLinkRow) apicontract.NewbroRecruitLink {
	entries := make([]apicontract.NewbroRecruitEntry, 0, len(row.Entries))
	for _, entry := range row.Entries {
		entries = append(entries, apicontract.NewbroRecruitEntry{
			Id: entry.ID, QQ: entry.QQ, EnteredAt: entry.EnteredAt,
			Source: apicontract.NewbroRecruitSource(entry.Source), Status: apicontract.NewbroRecruitEntryStatus(entry.Status),
			MatchedUserId: entry.MatchedUserID, MatchedCharacterName: entry.MatchedCharacterName,
			RewardedAt: entry.RewardedAt,
		})
	}
	return apicontract.NewbroRecruitLink{
		Id: row.ID, Code: row.Code, Source: apicontract.NewbroRecruitSource(row.Source),
		GeneratedAt: row.GeneratedAt, Entries: entries,
	}
}

func newbroDirectReferrer(row service.DirectReferrerCandidate) apicontract.NewbroDirectReferrer {
	return apicontract.NewbroDirectReferrer{
		UserId: row.UserID, Nickname: row.Nickname, PrimaryCharacterId: row.PrimaryCharacterID,
		PrimaryCharacterName: row.PrimaryCharacterName,
	}
}

func newbroPagination(current, size *int, defaultSize, maxSize int) (int, int) {
	page, pageSize := 1, defaultSize
	if current != nil {
		page = *current
	}
	if size != nil {
		pageSize = *size
	}
	return normalizePagination(page, pageSize, defaultSize, maxSize)
}

func newbroStrictUserID(ctx context.Context) uint {
	if c, ok := ctx.(*gin.Context); ok && c != nil {
		return middleware.GetUserID(c)
	}
	return 0
}

func newbroStrictBusinessError(err error) apicontract.BusinessError {
	return contractBusinessError(response.CodeBizError, err.Error())
}

func newbroInvalidRequest() apicontract.BusinessError {
	return contractBusinessError(response.CodeParamError, "invalid request")
}

func (h *NewbroRecruitHandler) SubmitRecruitQQ(_ context.Context, request apicontract.SubmitRecruitQQRequestObject) (apicontract.SubmitRecruitQQResponseObject, error) {
	out := &apicontract.SubmitRecruitQQ200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(newbroInvalidRequest())
	}
	data, err := h.submitRecruitQQ(request.Code, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromNewbroSubmittedQQSuccess(apicontract.NewbroSubmittedQQSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *NewbroRecruitHandler) GenerateLink(ctx context.Context, _ apicontract.GenerateLinkRequestObject) (apicontract.GenerateLinkResponseObject, error) {
	out := &apicontract.GenerateLink200JSONResponse{}
	data, err := h.generateLink(newbroStrictUserID(ctx))
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromNewbroGeneratedLinkSuccess(apicontract.NewbroGeneratedLinkSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *NewbroRecruitHandler) GetMyLinks(ctx context.Context, _ apicontract.GetMyLinksRequestObject) (apicontract.GetMyLinksResponseObject, error) {
	out := &apicontract.GetMyLinks200JSONResponse{}
	data, err := h.getMyLinks(newbroStrictUserID(ctx))
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromNewbroRecruitLinkListSuccess(apicontract.NewbroRecruitLinkListSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *NewbroRecruitHandler) GetDirectReferralStatus(ctx context.Context, _ apicontract.GetDirectReferralStatusRequestObject) (apicontract.GetDirectReferralStatusResponseObject, error) {
	out := &apicontract.GetDirectReferralStatus200JSONResponse{}
	data, err := h.getDirectReferralStatus(newbroStrictUserID(ctx))
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromNewbroDirectReferralStatusSuccess(apicontract.NewbroDirectReferralStatusSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *NewbroRecruitHandler) CheckDirectReferrer(ctx context.Context, request apicontract.CheckDirectReferrerRequestObject) (apicontract.CheckDirectReferrerResponseObject, error) {
	out := &apicontract.CheckDirectReferrer200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(newbroInvalidRequest())
	}
	data, err := h.checkDirectReferrer(newbroStrictUserID(ctx), *request.Body)
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromNewbroDirectReferrerSuccess(apicontract.NewbroDirectReferrerSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *NewbroRecruitHandler) ConfirmDirectReferrer(ctx context.Context, request apicontract.ConfirmDirectReferrerRequestObject) (apicontract.ConfirmDirectReferrerResponseObject, error) {
	out := &apicontract.ConfirmDirectReferrer200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(newbroInvalidRequest())
	}
	data, err := h.confirmDirectReferrer(newbroStrictUserID(ctx), *request.Body)
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromNewbroDirectReferrerSuccess(apicontract.NewbroDirectReferrerSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *NewbroRecruitHandler) ListAdminRecruitLinks(_ context.Context, request apicontract.ListAdminRecruitLinksRequestObject) (apicontract.ListAdminRecruitLinksResponseObject, error) {
	out := &apicontract.ListAdminRecruitLinks200JSONResponse{}
	data, err := h.listAdminRecruitLinks(request.Params)
	if err != nil {
		return out, out.FromBusinessError(newbroStrictBusinessError(err))
	}
	return out, out.FromNewbroAdminRecruitLinkPageSuccess(apicontract.NewbroAdminRecruitLinkPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}
