package handler

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
)

type SysConfigHandler struct {
	repo *repository.SysConfigRepository
}

func NewSysConfigHandler() *SysConfigHandler {
	return &SysConfigHandler{
		repo: repository.NewSysConfigRepository(),
	}
}

type BasicConfigResponse struct {
	CorpID    int64  `json:"corp_id"`
	SiteTitle string `json:"site_title"`
}

type UpdateBasicConfigRequest struct {
	CorpID    *int64  `json:"corp_id"`
	SiteTitle *string `json:"site_title"`
}

func (h *SysConfigHandler) GetBasicConfig(c *gin.Context) {
	defaultCorpID := strconv.FormatInt(model.SysConfigDefaultCorpID, 10)
	corpIDStr, _ := h.repo.Get(model.SysConfigCorpID, defaultCorpID)
	corpID, err := strconv.ParseInt(corpIDStr, 10, 64)
	if err != nil {
		corpID = model.SysConfigDefaultCorpID
	}
	siteTitle, _ := h.repo.Get(model.SysConfigSiteTitle, model.SysConfigDefaultSiteTitle)

	response.OK(c, BasicConfigResponse{
		CorpID:    corpID,
		SiteTitle: siteTitle,
	})
}

func (h *SysConfigHandler) UpdateBasicConfig(c *gin.Context) {
	var req UpdateBasicConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}

	if req.CorpID != nil {
		if err := h.repo.Set(model.SysConfigCorpID, strconv.FormatInt(*req.CorpID, 10), "军团ID"); err != nil {
			response.Fail(c, response.CodeBizError, "更新军团ID失败")
			return
		}
	}

	if req.SiteTitle != nil {
		if err := h.repo.Set(model.SysConfigSiteTitle, *req.SiteTitle, "网站标题"); err != nil {
			response.Fail(c, response.CodeBizError, "更新网站标题失败")
			return
		}
	}

	response.OK(c, nil)
}
