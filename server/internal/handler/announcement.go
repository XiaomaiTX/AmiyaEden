package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type AnnouncementHandler struct{ svc *service.AnnouncementService }

func NewAnnouncementHandler() *AnnouncementHandler {
	return &AnnouncementHandler{service.NewAnnouncementService()}
}
func announcementResult(c *gin.Context, data interface{}, err error) {
	switch {
	case err == nil:
		response.OK(c, data)
	case errors.Is(err, service.ErrAnnouncementInvalid):
		response.Fail(c, response.CodeParamError, "公告标题或正文无效，请检查长度后重试")
	case errors.Is(err, service.ErrAnnouncementImage):
		response.Fail(c, response.CodeParamError, "图片无效：仅支持 8 MiB 以内的 PNG 或 JPEG")
	case errors.Is(err, service.ErrAnnouncementTooManyImages):
		response.Fail(c, response.CodeParamError, "每条公告最多包含 6 张图片")
	case errors.Is(err, service.ErrAnnouncementNotFound):
		response.Fail(c, response.CodeNotFound, "公告不存在或已隐藏")
	case errors.Is(err, service.ErrAnnouncementForbidden):
		response.Fail(c, response.CodeForbidden, "无权管理公告")
	case errors.Is(err, service.ErrAnnouncementConflict):
		response.Fail(c, response.CodeBizError, "公告已被其他管理员修改，请取消后重新打开再编辑")
	case errors.Is(err, service.ErrAnnouncementStale):
		response.Fail(c, response.CodeBizError, "公告内容已更新，请重新打开查看")
	default:
		response.Fail(c, response.CodeBizError, "公告操作失败，请稍后重试")
	}
}
func (h *AnnouncementHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 || page > 100000 {
		page = 1
	}
	data, err := h.svc.List(middleware.GetUserID(c), middleware.GetUserRoles(c), page)
	announcementResult(c, data, err)
}
func (h *AnnouncementHandler) Detail(c *gin.Context) {
	id := requireUintID(c, "id", "announcement")
	if id == 0 {
		return
	}
	data, err := h.svc.Detail(id, middleware.GetUserRoles(c))
	announcementResult(c, data, err)
}
func (h *AnnouncementHandler) Save(c *gin.Context) {
	var id uint
	if c.Param("id") != "" {
		id = requireUintID(c, "id", "announcement")
		if id == 0 {
			return
		}
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 72<<20)
	var in service.AnnouncementInput
	if err := c.ShouldBindJSON(&in); err != nil {
		announcementResult(c, nil, service.ErrAnnouncementInvalid)
		return
	}
	data, err := h.svc.Save(middleware.GetUserID(c), id, middleware.GetUserRoles(c), in)
	announcementResult(c, data, err)
}
func (h *AnnouncementHandler) Read(c *gin.Context) {
	id := requireUintID(c, "id", "announcement")
	if id == 0 {
		return
	}
	var in struct {
		Version uint `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误")
		return
	}
	announcementResult(c, nil, h.svc.Read(middleware.GetUserID(c), id, in.Version))
}
