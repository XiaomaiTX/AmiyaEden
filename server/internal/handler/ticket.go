package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

type TicketHandler struct {
	svc *service.TicketService
}

func NewTicketHandler() *TicketHandler {
	return &TicketHandler{svc: service.NewTicketService()}
}

func (h *TicketHandler) CreateTicket(c *gin.Context) {
	var req struct {
		CategoryID  uint   `json:"category_id" binding:"required"`
		Title       string `json:"title" binding:"required"`
		Description string `json:"description" binding:"required"`
		Priority    string `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	ticket, err := h.svc.CreateTicket(middleware.GetUserID(c), req.CategoryID, req.Title, req.Description, req.Priority)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, ticket)
}

func (h *TicketHandler) ListMyTickets(c *gin.Context) {
	page, pageSize, err := parsePaginationQuery(c, 20, 100)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	list, total, err := h.svc.ListMyTickets(middleware.GetUserID(c), c.Query("status"), page, pageSize)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OKWithPage(c, list, total, page, pageSize)
}

func (h *TicketHandler) GetMyTicket(c *gin.Context) {
	ticketID := requireUintID(c, "id", "工单 ID")
	if ticketID == 0 {
		return
	}
	ticket, err := h.svc.GetMyTicket(middleware.GetUserID(c), ticketID)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, ticket)
}

func (h *TicketHandler) AddMyReply(c *gin.Context) {
	ticketID := requireUintID(c, "id", "工单 ID")
	if ticketID == 0 {
		return
	}
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	reply, err := h.svc.AddReplyAsUser(middleware.GetUserID(c), ticketID, req.Content)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, reply)
}

func (h *TicketHandler) ListMyReplies(c *gin.Context) {
	ticketID := requireUintID(c, "id", "工单 ID")
	if ticketID == 0 {
		return
	}
	list, err := h.svc.ListRepliesAsUser(middleware.GetUserID(c), ticketID)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *TicketHandler) ListCategories(c *gin.Context) {
	list, err := h.svc.ListCategories(true)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *TicketHandler) AdminListTickets(c *gin.Context) {
	page, pageSize, err := parsePaginationQuery(c, 20, 100)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	filter := repository.TicketListFilter{
		Status:  c.Query("status"),
		Keyword: c.Query("keyword"),
	}
	if raw := c.Query("user_id"); raw != "" {
		userID, convErr := parseRequiredUintQueryParam("user_id", raw)
		if convErr != nil {
			response.Fail(c, response.CodeParamError, "无效的 user_id")
			return
		}
		filter.UserID = userID
	}
	if raw := c.Query("category_id"); raw != "" {
		categoryID, convErr := parseRequiredUintQueryParam("category_id", raw)
		if convErr != nil {
			response.Fail(c, response.CodeParamError, "无效的 category_id")
			return
		}
		filter.Category = categoryID
	}
	list, total, err := h.svc.ListTicketsAdmin(filter, page, pageSize)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OKWithPage(c, list, total, page, pageSize)
}

func (h *TicketHandler) AdminGetTicket(c *gin.Context) {
	ticketID := requireUintID(c, "id", "工单 ID")
	if ticketID == 0 {
		return
	}
	ticket, err := h.svc.GetAdminTicket(ticketID)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, ticket)
}

func (h *TicketHandler) AdminUpdateStatus(c *gin.Context) {
	ticketID := requireUintID(c, "id", "工单 ID")
	if ticketID == 0 {
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	ticket, err := h.svc.UpdateStatusAsAdmin(middleware.GetUserID(c), ticketID, req.Status)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, ticket)
}

func (h *TicketHandler) AdminUpdatePriority(c *gin.Context) {
	ticketID := requireUintID(c, "id", "工单 ID")
	if ticketID == 0 {
		return
	}
	var req struct {
		Priority string `json:"priority" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	ticket, err := h.svc.UpdatePriorityAsAdmin(ticketID, req.Priority)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, ticket)
}

func (h *TicketHandler) AdminAddReply(c *gin.Context) {
	ticketID := requireUintID(c, "id", "工单 ID")
	if ticketID == 0 {
		return
	}
	var req struct {
		Content    string `json:"content" binding:"required"`
		IsInternal bool   `json:"is_internal"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	reply, err := h.svc.AddReplyAsAdmin(middleware.GetUserID(c), ticketID, req.Content, req.IsInternal)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, reply)
}

func (h *TicketHandler) AdminListReplies(c *gin.Context) {
	ticketID := requireUintID(c, "id", "工单 ID")
	if ticketID == 0 {
		return
	}
	list, err := h.svc.ListRepliesAsAdmin(ticketID)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *TicketHandler) AdminListStatusHistory(c *gin.Context) {
	ticketID := requireUintID(c, "id", "工单 ID")
	if ticketID == 0 {
		return
	}
	list, err := h.svc.ListStatusHistoryAsAdmin(ticketID)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *TicketHandler) AdminListCategories(c *gin.Context) {
	list, err := h.svc.ListCategories(false)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *TicketHandler) AdminCreateCategory(c *gin.Context) {
	var req model.TicketCategory
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	if err := h.svc.CreateCategory(&req); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, req)
}

func (h *TicketHandler) AdminUpdateCategory(c *gin.Context) {
	id := requireUintID(c, "id", "分类 ID")
	if id == 0 {
		return
	}
	var req model.TicketCategory
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	result, err := h.svc.UpdateCategory(id, &req)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *TicketHandler) AdminDeleteCategory(c *gin.Context) {
	id := requireUintID(c, "id", "分类 ID")
	if id == 0 {
		return
	}
	if err := h.svc.DeleteCategory(id); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *TicketHandler) AdminStatistics(c *gin.Context) {
	stats, err := h.svc.GetStatistics()
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, stats)
}
