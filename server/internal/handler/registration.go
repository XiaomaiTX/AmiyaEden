package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"context"
)

func (h *EveSSOHandler) getEveRegistration(ctx context.Context, ticket string) (apicontract.EveRegistrationInfo, error) {
	info, err := h.svc.GetRegistration(ctx, ticket)
	if err != nil {
		return apicontract.EveRegistrationInfo{}, err
	}
	return apicontract.EveRegistrationInfo{CharacterId: info.CharacterID, CharacterName: info.CharacterName}, nil
}

func (h *EveSSOHandler) completeEveRegistration(request ssoRequestContext, ticket string, input apicontract.EveRegistrationInput) (apicontract.EveRegistrationResult, error) {
	result, err := h.svc.CompleteRegistration(request.ctx, ticket, service.RegistrationInput{
		Nickname: input.Nickname, QQ: input.Qq, DiscordID: input.DiscordId,
	}, request.clientIP)
	if err != nil {
		return apicontract.EveRegistrationResult{}, err
	}
	h.recordSecurityEvent(request, "eve_sso_registration_success", model.AuditResultSuccess, map[string]any{
		"user_id": result.User.ID, "character_id": result.Character.CharacterID,
	})
	return apicontract.EveRegistrationResult{Token: result.Token}, nil
}
