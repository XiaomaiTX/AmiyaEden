package handler

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func contactConflictResponse(c *gin.Context, err error) bool {
	var conflict *service.ContactConflictError
	if !errors.As(err, &conflict) {
		return false
	}
	c.JSON(http.StatusOK, response.Response{Code: 409, Msg: "contact_already_bound", Data: conflict})
	return true
}

func (h *EveSSOHandler) GetRegistration(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	info, err := h.svc.GetRegistration(c.Request.Context(), c.GetHeader("X-Registration-Token"))
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	response.OK(c, info)
}

func (h *EveSSOHandler) CompleteRegistration(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var input service.RegistrationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	result, err := h.svc.CompleteRegistration(c.Request.Context(), c.GetHeader("X-Registration-Token"), input, c.ClientIP())
	if err != nil {
		if contactConflictResponse(c, err) {
			return
		}
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	h.recordSecurityEvent(c, "eve_sso_registration_success", model.AuditResultSuccess, map[string]any{
		"user_id": result.User.ID, "character_id": result.Character.CharacterID,
	})
	response.OK(c, gin.H{"token": result.Token})
}
