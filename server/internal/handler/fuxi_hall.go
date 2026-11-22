package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"

	"github.com/gin-gonic/gin"
)

type fuxiHallService interface {
	RefreshContributions(ctx context.Context, operatorID uint) error
	PublicHall(viewerID uint) (*service.FuxiHallPublicPageResponse, error)
	UpdateOwnCard(userID, id uint, req *service.FuxiHallOwnCardRequest) (*model.FuxiHallCard, error)
	GetPageConfig(pageKey string) (*model.FuxiHallPage, error)
	UpdatePageConfig(operatorID uint, pageKey string, req *service.FuxiHallUpdatePageRequest) (*model.FuxiHallPage, error)
	ListCards(pageKey string, visibleOnly bool) ([]model.FuxiHallCard, error)
	ListManageCards(pageKey string) ([]service.FuxiHallManageCard, error)
	CreateCard(operatorID uint, req *service.FuxiHallCreateCardRequest) (*model.FuxiHallCard, error)
	UpdateCard(operatorID, id uint, operatorRoles []string, req *service.FuxiHallUpdateCardRequest) (*model.FuxiHallCard, error)
	ReorderCards(operatorID uint, req *service.FuxiHallReorderRequest) error
	DeleteCard(operatorID, id uint) error
}

func (h *FuxiHallHandler) RefreshContributions(c *gin.Context) {
	if err := h.svc.RefreshContributions(c.Request.Context(), middleware.GetUserID(c)); err != nil {
		respondFuxiHallError(c, err, "fuxiHall.errors.refresh")
		return
	}
	response.OK(c, nil)
}

// FuxiHallHandler 伏羲大厅处理器
type FuxiHallHandler struct {
	svc fuxiHallService
}

func NewFuxiHallHandler() *FuxiHallHandler {
	return &FuxiHallHandler{svc: service.NewFuxiHallService()}
}

func respondFuxiHallError(c *gin.Context, err error, fallback string) {
	message := fallback
	if service.IsUserVisibleError(err) {
		message = err.Error()
	}
	response.Fail(c, response.CodeBizError, message)
}

func (h *FuxiHallHandler) GetHall(c *gin.Context) {
	data, err := h.svc.PublicHall(middleware.GetUserID(c))
	if err != nil {
		respondFuxiHallError(c, err, "fuxiHall.errors.load")
		return
	}
	response.OK(c, data)
}

func (h *FuxiHallHandler) UpdateOwnCard(c *gin.Context) {
	id := requireUintID(c, "id", "card")
	if id == 0 {
		return
	}
	var req service.FuxiHallOwnCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "fuxiHall.errors.invalid")
		return
	}
	card, err := h.svc.UpdateOwnCard(middleware.GetUserID(c), id, &req)
	if err != nil {
		respondFuxiHallError(c, err, "fuxiHall.errors.save")
		return
	}
	response.OK(c, card)
}

// GetPageConfig GET /api/v1/system/fuxi-hall/pages/:page_key
func (h *FuxiHallHandler) GetPageConfig(c *gin.Context) {
	page, err := h.svc.GetPageConfig(c.Param("page_key"))
	if err != nil {
		respondFuxiHallError(c, err, "获取页面配置失败")
		return
	}
	response.OK(c, page)
}

// UpdatePageConfig PUT /api/v1/system/fuxi-hall/pages/:page_key
func (h *FuxiHallHandler) UpdatePageConfig(c *gin.Context) {
	var req service.FuxiHallUpdatePageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	page, err := h.svc.UpdatePageConfig(middleware.GetUserID(c), c.Param("page_key"), &req)
	if err != nil {
		respondFuxiHallError(c, err, "更新页面配置失败")
		return
	}
	response.OK(c, page)
}

// ListCards GET /api/v1/system/fuxi-hall/cards/:page_key
func (h *FuxiHallHandler) ListCards(c *gin.Context) {
	cards, err := h.svc.ListManageCards(c.Param("page_key"))
	if err != nil {
		respondFuxiHallError(c, err, "获取卡片列表失败")
		return
	}
	response.OK(c, cards)
}

// CreateCard POST /api/v1/system/fuxi-hall/cards
func (h *FuxiHallHandler) CreateCard(c *gin.Context) {
	var req service.FuxiHallCreateCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	card, err := h.svc.CreateCard(middleware.GetUserID(c), &req)
	if err != nil {
		respondFuxiHallError(c, err, "创建卡片失败")
		return
	}
	response.OK(c, card)
}

// UpdateCard PUT /api/v1/system/fuxi-hall/cards/:id
func (h *FuxiHallHandler) UpdateCard(c *gin.Context) {
	id := requireUintID(c, "id", "卡片 ID")
	if id == 0 {
		return
	}

	var req service.FuxiHallUpdateCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	card, err := h.svc.UpdateCard(middleware.GetUserID(c), id, middleware.GetUserRoles(c), &req)
	if err != nil {
		respondFuxiHallError(c, err, "更新卡片失败")
		return
	}
	response.OK(c, card)
}

// ReorderCards PUT /api/v1/system/fuxi-hall/cards/reorder
func (h *FuxiHallHandler) ReorderCards(c *gin.Context) {
	var req service.FuxiHallReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return
	}
	if err := h.svc.ReorderCards(middleware.GetUserID(c), &req); err != nil {
		respondFuxiHallError(c, err, "更新排序失败")
		return
	}
	response.OK(c, nil)
}

// DeleteCard DELETE /api/v1/system/fuxi-hall/cards/:id
func (h *FuxiHallHandler) DeleteCard(c *gin.Context) {
	id := requireUintID(c, "id", "卡片 ID")
	if id == 0 {
		return
	}
	if err := h.svc.DeleteCard(middleware.GetUserID(c), id); err != nil {
		respondFuxiHallError(c, err, "删除卡片失败")
		return
	}
	response.OK(c, nil)
}
