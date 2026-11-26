package handler

import (
	"context"
	"errors"
	"math"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

// AnnouncementMaxBodyBytes limits create and update JSON before decoding.
const AnnouncementMaxBodyBytes = 72 << 20

var (
	errAnnouncementID        = errors.New("invalid announcement id")
	errAnnouncementReadParam = errors.New("invalid announcement read parameter")
)

type AnnouncementHandler struct{ svc *service.AnnouncementService }

func NewAnnouncementHandler() *AnnouncementHandler {
	return &AnnouncementHandler{service.NewAnnouncementService()}
}

func announcementError(err error) (int, string) {
	switch {
	case errors.Is(err, service.ErrAnnouncementInvalid):
		return response.CodeParamError, "公告标题或正文无效，请检查长度后重试"
	case errors.Is(err, errAnnouncementID):
		return response.CodeParamError, "无效的announcement"
	case errors.Is(err, errAnnouncementReadParam):
		return response.CodeParamError, "参数错误"
	case errors.Is(err, service.ErrAnnouncementImage):
		return response.CodeParamError, "图片无效：仅支持 8 MiB 以内的 PNG 或 JPEG"
	case errors.Is(err, service.ErrAnnouncementTooManyImages):
		return response.CodeParamError, "每条公告最多包含 6 张图片"
	case errors.Is(err, service.ErrAnnouncementNotFound):
		return response.CodeNotFound, "公告不存在或已隐藏"
	case errors.Is(err, service.ErrAnnouncementForbidden):
		return response.CodeForbidden, "无权管理公告"
	case errors.Is(err, service.ErrAnnouncementConflict):
		return response.CodeBizError, "公告已被其他管理员修改，请取消后重新打开再编辑"
	case errors.Is(err, service.ErrAnnouncementStale):
		return response.CodeBizError, "公告内容已更新，请重新打开查看"
	default:
		return response.CodeBizError, "公告操作失败，请稍后重试"
	}
}

func (h *AnnouncementHandler) listAnnouncements(user uint, roles []string, requestedPage *int) (apicontract.AnnouncementList, error) {
	page := 1
	if requestedPage != nil && *requestedPage >= 1 && *requestedPage <= 100000 {
		page = *requestedPage
	}
	data, err := h.svc.List(user, roles, page)
	if err != nil {
		return apicontract.AnnouncementList{}, err
	}
	rows := make([]apicontract.AnnouncementRow, 0, len(data.List))
	for _, row := range data.List {
		a := announcementSummary(row.Announcement, row.AuthorNickname, row.AuthorCharacterID, row.AuthorCharacterName)
		rows = append(rows, apicontract.AnnouncementRow{
			Id: a.Id, Title: a.Title, AuthorId: a.AuthorId, Hidden: a.Hidden,
			Version: a.Version, Revision: a.Revision, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
			AuthorNickname: a.AuthorNickname, AuthorCharacterId: a.AuthorCharacterId, AuthorCharacterName: a.AuthorCharacterName,
			Unread: row.Unread,
		})
	}
	return apicontract.AnnouncementList{List: rows, Total: data.Total}, nil
}

func (h *AnnouncementHandler) getAnnouncement(id uint, roles []string) (apicontract.Announcement, error) {
	if !validAnnouncementID(id) {
		return apicontract.Announcement{}, errAnnouncementID
	}
	data, err := h.svc.Detail(id, roles)
	if err != nil {
		return apicontract.Announcement{}, err
	}
	return announcementDetail(*data), nil
}

func (h *AnnouncementHandler) saveAnnouncement(user uint, id *uint, roles []string, in *apicontract.AnnouncementInput) (apicontract.Announcement, error) {
	var saveID uint // The service interprets 0 as a create, never an update.
	if id != nil {
		if !validAnnouncementID(*id) {
			return apicontract.Announcement{}, errAnnouncementID
		}
		saveID = *id
	}
	if in == nil {
		return apicontract.Announcement{}, service.ErrAnnouncementInvalid
	}
	data, err := h.svc.Save(user, saveID, roles, service.AnnouncementInput{
		Title: in.Title, BodyHTML: in.BodyHtml, Hidden: in.Hidden, Remind: in.Remind, Revision: in.Revision,
	})
	if err != nil {
		return apicontract.Announcement{}, err
	}
	return announcementDetail(*data), nil
}

func (h *AnnouncementHandler) readAnnouncement(user, id uint, in *apicontract.AnnouncementReadInput) error {
	if !validAnnouncementID(id) {
		return errAnnouncementID
	}
	if in == nil || in.Version == 0 {
		return errAnnouncementReadParam
	}
	return h.svc.Read(user, id, in.Version)
}

func validAnnouncementID(id uint) bool {
	return id > 0 && id <= math.MaxUint32
}

func announcementSummary(a model.Announcement, nickname string, characterID int64, characterName string) apicontract.AnnouncementSummary {
	return apicontract.AnnouncementSummary{
		Id: a.ID, Title: a.Title, AuthorId: a.AuthorID, Hidden: a.Hidden,
		Version: a.Version, Revision: a.Revision, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
		AuthorNickname: nickname, AuthorCharacterId: characterID, AuthorCharacterName: characterName,
	}
}

func announcementDetail(detail service.AnnouncementDetail) apicontract.Announcement {
	a := announcementSummary(detail.Announcement, detail.AuthorNickname, detail.AuthorCharacterID, detail.AuthorCharacterName)
	out := apicontract.Announcement{
		Id: a.Id, Title: a.Title, AuthorId: a.AuthorId, Hidden: a.Hidden,
		Version: a.Version, Revision: a.Revision, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
		AuthorNickname: a.AuthorNickname, AuthorCharacterId: a.AuthorCharacterId, AuthorCharacterName: a.AuthorCharacterName,
	}
	if detail.BodyHTML != "" {
		out.BodyHtml = &detail.BodyHTML
	}
	return out
}

func (h *AnnouncementHandler) ListAnnouncements(ctx context.Context, request apicontract.ListAnnouncementsRequestObject) (apicontract.ListAnnouncementsResponseObject, error) {
	c := ctx.(*gin.Context)
	data, err := h.listAnnouncements(middleware.GetUserID(c), middleware.GetUserRoles(c), request.Params.Page)
	var out apicontract.ListAnnouncements200JSONResponse
	if err != nil {
		code, msg := announcementError(err)
		if code == response.CodeForbidden {
			return apicontract.ListAnnouncements403JSONResponse{ForbiddenJSONResponse: announcementForbidden(msg)}, nil
		}
		return &out, out.FromBusinessError(contractBusinessError(code, msg))
	}
	return &out, out.FromAnnouncementListSuccess(apicontract.AnnouncementListSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AnnouncementHandler) GetAnnouncement(ctx context.Context, request apicontract.GetAnnouncementRequestObject) (apicontract.GetAnnouncementResponseObject, error) {
	c := ctx.(*gin.Context)
	data, err := h.getAnnouncement(request.Id, middleware.GetUserRoles(c))
	var out apicontract.GetAnnouncement200JSONResponse
	if err != nil {
		code, msg := announcementError(err)
		if code == response.CodeForbidden {
			return apicontract.GetAnnouncement403JSONResponse{ForbiddenJSONResponse: announcementForbidden(msg)}, nil
		}
		return &out, out.FromBusinessError(contractBusinessError(code, msg))
	}
	return &out, out.FromAnnouncementSuccess(apicontract.AnnouncementSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AnnouncementHandler) CreateAnnouncement(ctx context.Context, request apicontract.CreateAnnouncementRequestObject) (apicontract.CreateAnnouncementResponseObject, error) {
	c := ctx.(*gin.Context)
	data, err := h.saveAnnouncement(middleware.GetUserID(c), nil, middleware.GetUserRoles(c), request.Body)
	var out apicontract.CreateAnnouncement200JSONResponse
	if err != nil {
		code, msg := announcementError(err)
		if code == response.CodeForbidden {
			return apicontract.CreateAnnouncement403JSONResponse{ForbiddenJSONResponse: announcementForbidden(msg)}, nil
		}
		return &out, out.FromBusinessError(contractBusinessError(code, msg))
	}
	return &out, out.FromAnnouncementSuccess(apicontract.AnnouncementSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AnnouncementHandler) UpdateAnnouncement(ctx context.Context, request apicontract.UpdateAnnouncementRequestObject) (apicontract.UpdateAnnouncementResponseObject, error) {
	c := ctx.(*gin.Context)
	data, err := h.saveAnnouncement(middleware.GetUserID(c), &request.Id, middleware.GetUserRoles(c), request.Body)
	var out apicontract.UpdateAnnouncement200JSONResponse
	if err != nil {
		code, msg := announcementError(err)
		if code == response.CodeForbidden {
			return apicontract.UpdateAnnouncement403JSONResponse{ForbiddenJSONResponse: announcementForbidden(msg)}, nil
		}
		return &out, out.FromBusinessError(contractBusinessError(code, msg))
	}
	return &out, out.FromAnnouncementSuccess(apicontract.AnnouncementSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AnnouncementHandler) ReadAnnouncement(ctx context.Context, request apicontract.ReadAnnouncementRequestObject) (apicontract.ReadAnnouncementResponseObject, error) {
	c := ctx.(*gin.Context)
	err := h.readAnnouncement(middleware.GetUserID(c), request.Id, request.Body)
	var out apicontract.ReadAnnouncement200JSONResponse
	if err != nil {
		code, msg := announcementError(err)
		if code == response.CodeForbidden {
			return apicontract.ReadAnnouncement403JSONResponse{ForbiddenJSONResponse: announcementForbidden(msg)}, nil
		}
		return &out, out.FromBusinessError(contractBusinessError(code, msg))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func announcementForbidden(msg string) apicontract.ForbiddenJSONResponse {
	return apicontract.ForbiddenJSONResponse{Code: response.CodeForbidden, Msg: msg}
}
