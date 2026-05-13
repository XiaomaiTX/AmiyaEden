package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

type ToolBookmarkHandler struct {
	svc *service.ToolBookmarkService
}

func NewToolBookmarkHandler() *ToolBookmarkHandler {
	return &ToolBookmarkHandler{svc: service.NewToolBookmarkService()}
}

type toolBookmarkCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	URL         string `json:"url" binding:"required"`
	Description string `json:"description"`
	IsEnabled   *bool  `json:"is_enabled"`
	SortOrder   *int   `json:"sort_order"`
}

type toolBookmarkUpdateRequest struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	IsEnabled   *bool  `json:"is_enabled"`
	SortOrder   *int   `json:"sort_order"`
}

func (h *ToolBookmarkHandler) ListVisible(c *gin.Context) {
	rows, err := h.svc.ListVisibleBookmarks()
	if err != nil {
		response.Fail(c, response.CodeBizError, "获取工具书签失败")
		return
	}
	response.OK(c, rows)
}

func (h *ToolBookmarkHandler) AdminList(c *gin.Context) {
	rows, err := h.svc.AdminList()
	if err != nil {
		response.Fail(c, response.CodeBizError, "获取工具书签失败")
		return
	}
	response.OK(c, rows)
}

func (h *ToolBookmarkHandler) AdminCreate(c *gin.Context) {
	var req toolBookmarkCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	row, err := h.svc.AdminCreate(middleware.GetUserID(c), service.ToolBookmarkUpsertRequest{
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		IsEnabled:   req.IsEnabled,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		if service.IsUserVisibleError(err) {
			response.Fail(c, response.CodeBizError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, "创建工具书签失败")
		return
	}
	response.OK(c, row)
}

func (h *ToolBookmarkHandler) AdminUpdate(c *gin.Context) {
	id := requireUintID(c, "id", "书签 ID")
	if id == 0 {
		return
	}
	var req toolBookmarkUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	row, err := h.svc.AdminUpdate(id, service.ToolBookmarkUpsertRequest{
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		IsEnabled:   req.IsEnabled,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		if service.IsUserVisibleError(err) {
			response.Fail(c, response.CodeBizError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, "更新工具书签失败")
		return
	}
	response.OK(c, row)
}

func (h *ToolBookmarkHandler) AdminDelete(c *gin.Context) {
	id := requireUintID(c, "id", "书签 ID")
	if id == 0 {
		return
	}
	if err := h.svc.AdminDelete(id); err != nil {
		if service.IsUserVisibleError(err) {
			response.Fail(c, response.CodeBizError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, "删除工具书签失败")
		return
	}
	response.OK(c, nil)
}
