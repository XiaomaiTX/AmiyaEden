package router

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/handler"
	"amiya-eden/internal/service"

	"github.com/gin-gonic/gin"
)

// registerOpenAPIRoutes assembles module implementations. The compiler checks
// the complete strict interface, and the generator owns the route table.
func registerOpenAPIRoutes(r gin.IRouter, taskSvc *service.TaskService) {
	strict := &struct {
		*handler.AnnouncementHandler
		*handler.MumbleHandler
		*handler.SdeHandler
		*handler.MeHandler
		*handler.CaptainRefundHandler
		*handler.NewbroAdminHandler
		*handler.NewbroRecruitHandler
		*handler.GalaxyRegistryHandler
		*handler.SkillPlanHandler
		*handler.CorporationStructureHandler
		*handler.SrpHandler
		*handler.FuxiHallHandler
		*handler.WelfareHandler
		*handler.SysWalletHandler
		*handler.ShopHandler
		*handler.FleetHandler
		*handler.AlliancePAPHandler
		*handler.FleetConfigHandler
		*handler.PAPExchangeHandler
		*handler.WebhookHandler
		*handler.AuditEventHandler
		*handler.UploadHandler
		*handler.AdmissionHandler
		*handler.TaskHandler
		*handler.ESIRefreshHandler
		*handler.SysConfigHandler
		*handler.SkillReadinessHandler
		*handler.DashboardHandler
		*handler.MentorMenteeHandler
		*handler.MentorMentorHandler
		*handler.MentorAdminHandler
		*handler.NpcKillHandler
		*handler.FittingsHandler
		*handler.EveInfoHandler
		*handler.BlacklistHandler
		*handler.ToolBookmarkHandler
		*handler.SalaryHandler
		*handler.BadgeHandler
		*handler.AutoRoleHandler
		*handler.EveSSOHandler
		*handler.TicketHandler
		*handler.UserHandler
		*handler.RoleHandler
	}{
		AnnouncementHandler:         handler.NewAnnouncementHandler(),
		MumbleHandler:               handler.NewMumbleHandler(),
		SdeHandler:                  handler.NewSdeHandler(),
		MeHandler:                   handler.NewMeHandler(),
		CaptainRefundHandler:        handler.NewCaptainRefundHandler(),
		NewbroAdminHandler:          handler.NewNewbroAdminHandler(),
		NewbroRecruitHandler:        handler.NewNewbroRecruitHandler(),
		GalaxyRegistryHandler:       handler.NewGalaxyRegistryHandler(),
		SkillPlanHandler:            handler.NewSkillPlanHandler(),
		CorporationStructureHandler: handler.NewCorporationStructureHandler(),
		SrpHandler:                  handler.NewSrpHandler(),
		FuxiHallHandler:             handler.NewFuxiHallHandler(),
		WelfareHandler:              handler.NewWelfareHandler(),
		SysWalletHandler:            handler.NewSysWalletHandler(),
		ShopHandler:                 handler.NewShopHandler(),
		FleetHandler:                handler.NewFleetHandler(),
		AlliancePAPHandler:          handler.NewAlliancePAPHandler(),
		FleetConfigHandler:          handler.NewFleetConfigHandler(),
		PAPExchangeHandler:          handler.NewPAPExchangeHandler(),
		WebhookHandler:              handler.NewWebhookHandler(),
		AuditEventHandler:           handler.NewAuditEventHandler(),
		UploadHandler:               handler.NewUploadHandler(),
		AdmissionHandler:            handler.NewAdmissionHandler(),
		TaskHandler:                 handler.NewTaskHandler(taskSvc),
		ESIRefreshHandler:           handler.NewESIRefreshHandler(),
		SysConfigHandler:            handler.NewSysConfigHandler(),
		SkillReadinessHandler:       handler.NewSkillReadinessHandler(),
		DashboardHandler:            handler.NewDashboardHandler(),
		MentorMenteeHandler:         handler.NewMentorMenteeHandler(),
		MentorMentorHandler:         handler.NewMentorMentorHandler(),
		MentorAdminHandler:          handler.NewMentorAdminHandler(),
		NpcKillHandler:              handler.NewNpcKillHandler(),
		FittingsHandler:             handler.NewFittingsHandler(),
		EveInfoHandler:              handler.NewEveInfoHandler(),
		BlacklistHandler:            handler.NewBlacklistHandler(),
		ToolBookmarkHandler:         handler.NewToolBookmarkHandler(),
		SalaryHandler:               handler.NewSalaryHandler(),
		BadgeHandler:                handler.NewBadgeHandler(),
		AutoRoleHandler:             handler.NewAutoRoleHandler(),
		EveSSOHandler:               handler.NewEveSSOHandler(),
		TicketHandler:               handler.NewTicketHandler(),
		UserHandler:                 handler.NewUserHandler(),
		RoleHandler:                 handler.NewRoleHandler(),
	}
	apicontract.RegisterHandlersWithOptions(r, apicontract.NewStrictHandler(strict), apicontract.GinServerOptions{
		BaseURL:                  "/api/v1",
		MiddlewaresForOperation:  newOpenAPIMiddlewareResolver(),
		ErrorHandlerForOperation: handleOpenAPIError,
	})
}
