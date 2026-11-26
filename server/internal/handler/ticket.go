package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"math"

	"github.com/gin-gonic/gin"
)

type TicketHandler struct {
	svc *service.TicketService
}

func NewTicketHandler() *TicketHandler {
	return &TicketHandler{svc: service.NewTicketService()}
}

type ticketParamError string

func (e ticketParamError) Error() string { return string(e) }

const (
	ticketInvalidUserID     ticketParamError = "无效的 user_id"
	ticketInvalidCategoryID ticketParamError = "无效的 category_id"
)

func ticketErrorCode(err error) int {
	if _, ok := err.(ticketParamError); ok {
		return response.CodeParamError
	}
	return response.CodeBizError
}

func (h *TicketHandler) createTicket(userID uint, req apicontract.TicketCreateInput) (apicontract.TicketRecord, error) {
	ticket, err := h.svc.CreateTicket(userID, req.CategoryId, req.Title, req.Description)
	if err != nil {
		return apicontract.TicketRecord{}, err
	}
	return ticketRecordDTO(*ticket), nil
}

func (h *TicketHandler) listMyTickets(userID uint, params apicontract.ListMyTicketsParams) (apicontract.TicketMemberPage, error) {
	page, pageSize := normalizePagination(ticketValue(params.Current), ticketValue(params.Size), 20, 100)
	list, total, err := h.svc.ListMyTickets(userID, ticketValue(params.Status), page, pageSize)
	if err != nil {
		return apicontract.TicketMemberPage{}, err
	}
	return apicontract.TicketMemberPage{
		List: ticketDTOs(list, ticketRecordDTO), Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func (h *TicketHandler) getMyTicket(userID, ticketID uint) (apicontract.TicketRecord, error) {
	ticket, err := h.svc.GetMyTicket(userID, ticketID)
	if err != nil {
		return apicontract.TicketRecord{}, err
	}
	return ticketRecordDTO(*ticket), nil
}

func (h *TicketHandler) addMyReply(userID, ticketID uint, req apicontract.TicketReplyInput) (apicontract.TicketReply, error) {
	reply, err := h.svc.AddReplyAsUser(userID, ticketID, req.Content)
	if err != nil {
		return apicontract.TicketReply{}, err
	}
	return ticketReplyDTO(*reply), nil
}

func (h *TicketHandler) listMyReplies(userID, ticketID uint) ([]apicontract.TicketReply, error) {
	list, err := h.svc.ListRepliesAsUser(userID, ticketID)
	return ticketDTOs(list, ticketReplyDTO), err
}

func (h *TicketHandler) ticketCategories(enabledOnly bool) ([]apicontract.TicketCategory, error) {
	list, err := h.svc.ListCategories(enabledOnly)
	return ticketDTOs(list, ticketCategoryDTO), err
}

func (h *TicketHandler) adminListTickets(params apicontract.AdminListTicketsParams) (apicontract.TicketAdminPage, error) {
	// The generated query binder decodes uint but does not enforce these schema bounds.
	if id := params.UserId; id != nil && (*id == 0 || *id > math.MaxUint32) {
		return apicontract.TicketAdminPage{}, ticketInvalidUserID
	}
	if id := params.CategoryId; id != nil && (*id == 0 || *id > math.MaxUint32) {
		return apicontract.TicketAdminPage{}, ticketInvalidCategoryID
	}
	page, pageSize := normalizePagination(ticketValue(params.Current), ticketValue(params.Size), 20, 100)
	filter := service.TicketListFilter{
		Status: ticketValue(params.Status), Keyword: ticketValue(params.Keyword),
		UserID: ticketValue(params.UserId), Category: ticketValue(params.CategoryId),
	}
	list, total, err := h.svc.ListTicketsAdmin(filter, page, pageSize)
	if err != nil {
		return apicontract.TicketAdminPage{}, err
	}
	return apicontract.TicketAdminPage{
		List: ticketDTOs(list, ticketAdminItemDTO), Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func (h *TicketHandler) adminGetTicket(ticketID uint) (apicontract.TicketAdminItem, error) {
	ticket, err := h.svc.GetAdminTicketDetail(ticketID)
	if err != nil {
		return apicontract.TicketAdminItem{}, err
	}
	return ticketAdminItemDTO(*ticket), nil
}

func (h *TicketHandler) adminUpdateStatus(userID, ticketID uint, req apicontract.TicketStatusInput) (apicontract.TicketRecord, error) {
	ticket, err := h.svc.UpdateStatusAsAdmin(userID, ticketID, string(req.Status))
	if err != nil {
		return apicontract.TicketRecord{}, err
	}
	return ticketRecordDTO(*ticket), nil
}

func (h *TicketHandler) adminAddReply(userID, ticketID uint, req apicontract.TicketAdminReplyInput) (apicontract.TicketReply, error) {
	reply, err := h.svc.AddReplyAsAdmin(userID, ticketID, req.Content, req.IsInternal)
	if err != nil {
		return apicontract.TicketReply{}, err
	}
	return ticketReplyDTO(*reply), nil
}

func (h *TicketHandler) adminListReplies(ticketID uint) ([]apicontract.TicketReply, error) {
	list, err := h.svc.ListRepliesAsAdmin(ticketID)
	return ticketDTOs(list, ticketReplyDTO), err
}

func (h *TicketHandler) adminListStatusHistory(ticketID uint) ([]apicontract.TicketStatusChange, error) {
	list, err := h.svc.ListStatusHistoryAsAdmin(ticketID)
	return ticketDTOs(list, ticketStatusChangeDTO), err
}

func (h *TicketHandler) adminCreateCategory(userID uint, req apicontract.TicketCategoryInput) (apicontract.TicketCategory, error) {
	category := ticketCategoryModel(req)
	if err := h.svc.CreateCategory(userID, &category); err != nil {
		return apicontract.TicketCategory{}, err
	}
	return ticketCategoryDTO(category), nil
}

func (h *TicketHandler) adminUpdateCategory(userID, categoryID uint, req apicontract.TicketCategoryInput) (apicontract.TicketCategory, error) {
	category := ticketCategoryModel(req)
	result, err := h.svc.UpdateCategory(userID, categoryID, &category)
	if err != nil {
		return apicontract.TicketCategory{}, err
	}
	return ticketCategoryDTO(*result), nil
}

func (h *TicketHandler) adminTicketStatistics() (apicontract.TicketStatistics, error) {
	stats, err := h.svc.GetStatistics()
	if err != nil {
		return apicontract.TicketStatistics{}, err
	}
	total, _ := stats["total"].(int64)
	status, _ := stats["status"].(map[string]int64)
	category, _ := stats["category"].(map[uint]int64)
	recent7d, _ := stats["recent_7d"].(int64)
	recent30d, _ := stats["recent_30d"].(int64)
	pendingCount, _ := stats["pendingCount"].(int64)
	return apicontract.TicketStatistics{
		Total: total, Status: status, Category: category,
		Recent7d: recent7d, Recent30d: recent30d, PendingCount: pendingCount,
	}, nil
}

func ticketValue[T any](value *T) T {
	if value != nil {
		return *value
	}
	var zero T
	return zero
}

func ticketDTOs[M, D any](rows []M, toDTO func(M) D) []D {
	if rows == nil {
		return nil
	}
	dtos := make([]D, len(rows))
	for i, row := range rows {
		dtos[i] = toDTO(row)
	}
	return dtos
}

func ticketRecordDTO(ticket model.Ticket) apicontract.TicketRecord {
	return apicontract.TicketRecord{
		Id: ticket.ID, UserId: ticket.UserID, CategoryId: ticket.CategoryID,
		Title: ticket.Title, Description: ticket.Description, Status: apicontract.TicketStatus(ticket.Status),
		HandledBy: ticket.HandledBy, HandledAt: ticket.HandledAt, ClosedAt: ticket.ClosedAt,
		CreatedAt: ticket.CreatedAt, UpdatedAt: ticket.UpdatedAt,
	}
}

func ticketAdminItemDTO(ticket model.TicketListItem) apicontract.TicketAdminItem {
	return apicontract.TicketAdminItem{
		Id: ticket.ID, UserId: ticket.UserID, CategoryId: ticket.CategoryID,
		Title: ticket.Title, Description: ticket.Description, Status: apicontract.TicketStatus(ticket.Status),
		HandledBy: ticket.HandledBy, HandledAt: ticket.HandledAt, ClosedAt: ticket.ClosedAt,
		CreatedAt: ticket.CreatedAt, UpdatedAt: ticket.UpdatedAt,
		UserNickname: ticket.UserNickname, HandledByNickname: ticket.HandledByNickname,
		RequesterName: ticket.RequesterName, RequesterCharacterName: ticket.RequesterCharacterName,
		CategoryName: ticket.CategoryName, CategoryNameEn: ticket.CategoryNameEN,
	}
}

func ticketCategoryModel(req apicontract.TicketCategoryInput) model.TicketCategory {
	return model.TicketCategory{
		BaseModel: model.BaseModel{ID: req.Id, CreatedAt: req.CreatedAt, UpdatedAt: req.UpdatedAt},
		Name:      req.Name, NameEN: req.NameEn, Description: req.Description,
		SortOrder: req.SortOrder, Enabled: req.Enabled, RetiredAdmission: req.RetiredAdmission,
	}
}

func ticketCategoryDTO(category model.TicketCategory) apicontract.TicketCategory {
	return apicontract.TicketCategory{
		Id: category.ID, Name: category.Name, NameEn: category.NameEN, Description: category.Description,
		SortOrder: category.SortOrder, Enabled: category.Enabled, RetiredAdmission: category.RetiredAdmission,
		CreatedAt: category.CreatedAt, UpdatedAt: category.UpdatedAt,
	}
}

func ticketReplyDTO(reply model.TicketReplyItem) apicontract.TicketReply {
	return apicontract.TicketReply{
		Id: reply.ID, TicketId: reply.TicketID, UserId: reply.UserID, Content: reply.Content,
		UserNickname: reply.UserNickname, ReplyUserNickname: reply.ReplyUserNickname,
		IsInternal: reply.IsInternal, CreatedAt: reply.CreatedAt, UpdatedAt: reply.UpdatedAt,
	}
}

func ticketStatusChangeDTO(change model.TicketStatusHistoryItem) apicontract.TicketStatusChange {
	return apicontract.TicketStatusChange{
		Id: change.ID, TicketId: change.TicketID, FromStatus: change.FromStatus, ToStatus: apicontract.TicketStatus(change.ToStatus),
		ChangedBy: change.ChangedBy, ChangedByNickname: change.ChangedByNickname,
		ChangedByName: change.ChangedByName, ChangedByCharacterName: change.ChangedByCharacterName, ChangedAt: change.ChangedAt,
	}
}

func ticketStrictUserID(ctx context.Context) (uint, bool) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return 0, false
	}
	id := middleware.GetUserID(c)
	return id, id != 0
}

func ticketUnauthorized() apicontract.UnauthorizedJSONResponse {
	return apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}
}

func ticketMissingBody() apicontract.BusinessError {
	return contractBusinessError(response.CodeParamError, "请求参数错误")
}

func ticketStrictError(err error) apicontract.BusinessError {
	return contractBusinessError(ticketErrorCode(err), err.Error())
}

func (h *TicketHandler) CreateTicket(ctx context.Context, request apicontract.CreateTicketRequestObject) (apicontract.CreateTicketResponseObject, error) {
	userID, ok := ticketStrictUserID(ctx)
	if !ok {
		return apicontract.CreateTicket401JSONResponse{UnauthorizedJSONResponse: ticketUnauthorized()}, nil
	}
	out := &apicontract.CreateTicket200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(ticketMissingBody())
	}
	data, err := h.createTicket(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketRecordSuccess(apicontract.TicketRecordSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) ListMyTickets(ctx context.Context, request apicontract.ListMyTicketsRequestObject) (apicontract.ListMyTicketsResponseObject, error) {
	userID, ok := ticketStrictUserID(ctx)
	if !ok {
		return apicontract.ListMyTickets401JSONResponse{UnauthorizedJSONResponse: ticketUnauthorized()}, nil
	}
	out := &apicontract.ListMyTickets200JSONResponse{}
	data, err := h.listMyTickets(userID, request.Params)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketMemberPageSuccess(apicontract.TicketMemberPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) GetMyTicket(ctx context.Context, request apicontract.GetMyTicketRequestObject) (apicontract.GetMyTicketResponseObject, error) {
	userID, ok := ticketStrictUserID(ctx)
	if !ok {
		return apicontract.GetMyTicket401JSONResponse{UnauthorizedJSONResponse: ticketUnauthorized()}, nil
	}
	out := &apicontract.GetMyTicket200JSONResponse{}
	data, err := h.getMyTicket(userID, request.Id)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketRecordSuccess(apicontract.TicketRecordSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) AddMyReply(ctx context.Context, request apicontract.AddMyReplyRequestObject) (apicontract.AddMyReplyResponseObject, error) {
	userID, ok := ticketStrictUserID(ctx)
	if !ok {
		return apicontract.AddMyReply401JSONResponse{UnauthorizedJSONResponse: ticketUnauthorized()}, nil
	}
	out := &apicontract.AddMyReply200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(ticketMissingBody())
	}
	data, err := h.addMyReply(userID, request.Id, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketReplySuccess(apicontract.TicketReplySuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) ListMyReplies(ctx context.Context, request apicontract.ListMyRepliesRequestObject) (apicontract.ListMyRepliesResponseObject, error) {
	userID, ok := ticketStrictUserID(ctx)
	if !ok {
		return apicontract.ListMyReplies401JSONResponse{UnauthorizedJSONResponse: ticketUnauthorized()}, nil
	}
	out := &apicontract.ListMyReplies200JSONResponse{}
	data, err := h.listMyReplies(userID, request.Id)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketRepliesSuccess(apicontract.TicketRepliesSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) ListCategories(_ context.Context, _ apicontract.ListCategoriesRequestObject) (apicontract.ListCategoriesResponseObject, error) {
	out := &apicontract.ListCategories200JSONResponse{}
	data, err := h.ticketCategories(true)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketCategoriesSuccess(apicontract.TicketCategoriesSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) AdminListTickets(_ context.Context, request apicontract.AdminListTicketsRequestObject) (apicontract.AdminListTicketsResponseObject, error) {
	out := &apicontract.AdminListTickets200JSONResponse{}
	data, err := h.adminListTickets(request.Params)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketAdminPageSuccess(apicontract.TicketAdminPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) AdminGetTicket(_ context.Context, request apicontract.AdminGetTicketRequestObject) (apicontract.AdminGetTicketResponseObject, error) {
	out := &apicontract.AdminGetTicket200JSONResponse{}
	data, err := h.adminGetTicket(request.Id)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketAdminItemSuccess(apicontract.TicketAdminItemSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) AdminUpdateStatus(ctx context.Context, request apicontract.AdminUpdateStatusRequestObject) (apicontract.AdminUpdateStatusResponseObject, error) {
	userID, ok := ticketStrictUserID(ctx)
	if !ok {
		return apicontract.AdminUpdateStatus401JSONResponse{UnauthorizedJSONResponse: ticketUnauthorized()}, nil
	}
	out := &apicontract.AdminUpdateStatus200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(ticketMissingBody())
	}
	data, err := h.adminUpdateStatus(userID, request.Id, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketRecordSuccess(apicontract.TicketRecordSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) AdminAddReply(ctx context.Context, request apicontract.AdminAddReplyRequestObject) (apicontract.AdminAddReplyResponseObject, error) {
	userID, ok := ticketStrictUserID(ctx)
	if !ok {
		return apicontract.AdminAddReply401JSONResponse{UnauthorizedJSONResponse: ticketUnauthorized()}, nil
	}
	out := &apicontract.AdminAddReply200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(ticketMissingBody())
	}
	data, err := h.adminAddReply(userID, request.Id, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketReplySuccess(apicontract.TicketReplySuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) AdminListReplies(_ context.Context, request apicontract.AdminListRepliesRequestObject) (apicontract.AdminListRepliesResponseObject, error) {
	out := &apicontract.AdminListReplies200JSONResponse{}
	data, err := h.adminListReplies(request.Id)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketRepliesSuccess(apicontract.TicketRepliesSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) AdminListStatusHistory(_ context.Context, request apicontract.AdminListStatusHistoryRequestObject) (apicontract.AdminListStatusHistoryResponseObject, error) {
	out := &apicontract.AdminListStatusHistory200JSONResponse{}
	data, err := h.adminListStatusHistory(request.Id)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketStatusHistorySuccess(apicontract.TicketStatusHistorySuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) AdminListCategories(_ context.Context, _ apicontract.AdminListCategoriesRequestObject) (apicontract.AdminListCategoriesResponseObject, error) {
	out := &apicontract.AdminListCategories200JSONResponse{}
	data, err := h.ticketCategories(false)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketCategoriesSuccess(apicontract.TicketCategoriesSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) AdminCreateCategory(ctx context.Context, request apicontract.AdminCreateCategoryRequestObject) (apicontract.AdminCreateCategoryResponseObject, error) {
	userID, ok := ticketStrictUserID(ctx)
	if !ok {
		return apicontract.AdminCreateCategory401JSONResponse{UnauthorizedJSONResponse: ticketUnauthorized()}, nil
	}
	out := &apicontract.AdminCreateCategory200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(ticketMissingBody())
	}
	data, err := h.adminCreateCategory(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketCategorySuccess(apicontract.TicketCategorySuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) AdminUpdateCategory(ctx context.Context, request apicontract.AdminUpdateCategoryRequestObject) (apicontract.AdminUpdateCategoryResponseObject, error) {
	userID, ok := ticketStrictUserID(ctx)
	if !ok {
		return apicontract.AdminUpdateCategory401JSONResponse{UnauthorizedJSONResponse: ticketUnauthorized()}, nil
	}
	out := &apicontract.AdminUpdateCategory200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(ticketMissingBody())
	}
	data, err := h.adminUpdateCategory(userID, request.Id, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketCategorySuccess(apicontract.TicketCategorySuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TicketHandler) AdminDeleteCategory(ctx context.Context, request apicontract.AdminDeleteCategoryRequestObject) (apicontract.AdminDeleteCategoryResponseObject, error) {
	userID, ok := ticketStrictUserID(ctx)
	if !ok {
		return apicontract.AdminDeleteCategory401JSONResponse{UnauthorizedJSONResponse: ticketUnauthorized()}, nil
	}
	out := &apicontract.AdminDeleteCategory200JSONResponse{}
	if err := h.svc.DeleteCategory(userID, request.Id); err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *TicketHandler) AdminStatistics(_ context.Context, _ apicontract.AdminStatisticsRequestObject) (apicontract.AdminStatisticsResponseObject, error) {
	out := &apicontract.AdminStatistics200JSONResponse{}
	data, err := h.adminTicketStatistics()
	if err != nil {
		return out, out.FromBusinessError(ticketStrictError(err))
	}
	return out, out.FromTicketStatisticsSuccess(apicontract.TicketStatisticsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}
