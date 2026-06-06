package router

import (
	"amiya-eden/internal/handler"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"

	"github.com/gin-gonic/gin"
)

var (
	srpPriceManageRoles          = []string{model.RoleAdmin, model.RoleSeniorFC}
	srpManageRoles               = []string{model.RoleSRP, model.RoleSeniorFC, model.RoleAdmin}
	srpPayoutRoles               = []string{model.RoleSRP, model.RoleSeniorFC, model.RoleAdmin}
	shopOrderManageRoles         = []string{model.RoleAdmin, model.RoleShopOrder}
	welfareApprovalRoles         = []string{model.RoleAdmin, model.RoleWelfare}
	skillPlanManageRoles         = []string{model.RoleAdmin, model.RoleSeniorFC}
	autoRoleManageRoles          = []string{model.RoleSuperAdmin}
	systemBasicConfigManageRoles = []string{model.RoleSuperAdmin}
	systemWebhookManageRoles     = []string{model.RoleSuperAdmin}
)

// RegisterRoutes 注册所有业务路由
func RegisterRoutes(r *gin.Engine, taskSvc *service.TaskService) {
	// ─── 上传文件静态目录 ───
	r.Static("/uploads", "./uploads")

	api := r.Group("/api/v1")

	// ─── 无需认证 ───
	ssoH := handler.NewEveSSOHandler()
	sso := api.Group("/sso/eve")
	{
		sso.GET("/login", ssoH.Login)
		sso.GET("/callback", ssoH.Callback)
		sso.GET("/scopes", ssoH.GetScopes)
	}

	// ─── SDE 公开查询 ──
	sdeH := handler.NewSdeHandler()
	sde := api.Group("/sde")
	{
		sde.GET("/version", sdeH.GetVersion)
		sde.POST("/types", sdeH.GetTypes)
		sde.POST("/names", sdeH.GetNames)
		sde.POST("/search", sdeH.FuzzySearch)
	}

	// ─── 招募链接（公开）───
	recruitH := handler.NewNewbroRecruitHandler()
	recruit := api.Group("/recruit")
	{
		recruit.POST("/:code/submit", recruitH.SubmitQQ)
	}

	// ─── 需要登录 ───
	auth := api.Group("", middleware.JWTAuth())
	login := auth.Group("", middleware.RequireLoginUser())

	// 福利（用户端 + 管理端共用 handler）
	welfareH := handler.NewWelfareHandler()
	requireMenuDashboard := middleware.RequireCorpCapability(model.CorpCapabilityMenuDashboard)
	requireMenuOperation := middleware.RequireCorpCapability(model.CorpCapabilityMenuOperation)
	requireMenuNewbro := middleware.RequireCorpCapability(model.CorpCapabilityMenuNewbro)
	requireMenuFuxiHall := middleware.RequireCorpCapability(model.CorpCapabilityMenuFuxiHall)
	requireMenuTicket := middleware.RequireCorpCapability(model.CorpCapabilityMenuTicket)
	requireMenuShop := middleware.RequireCorpCapability(model.CorpCapabilityMenuShop)
	requireMenuInfo := middleware.RequireCorpCapability(model.CorpCapabilityMenuInfo)
	requireMenuSkillPlan := middleware.RequireCorpCapability(model.CorpCapabilityMenuSkillPlan)
	requireTicketManage := middleware.RequireCorpCapability(model.CorpCapabilityTicketManage)
	requireShopManage := middleware.RequireCorpCapability(model.CorpCapabilityShopManage)
	requireSystemManage := middleware.RequireCorpCapability(model.CorpCapabilitySystemManage)
	requireInfoWalletRead := middleware.RequireCorpCapability(model.CorpCapabilityInfoWalletRead)
	requireInfoNpcKillsSelf := middleware.RequireCorpCapability(model.CorpCapabilityInfoNpcKillsSelf)
	requireInfoNpcKillsCorp := middleware.RequireCorpCapability(model.CorpCapabilityInfoNpcKillsCorp)
	requireInfoSkillsRead := middleware.RequireCorpCapability(model.CorpCapabilityInfoSkillsRead)
	requireInfoAssetsRead := middleware.RequireCorpCapability(model.CorpCapabilityInfoAssetsRead)
	requireInfoContractsRead := middleware.RequireCorpCapability(model.CorpCapabilityInfoContractsRead)
	requireInfoFittingsManage := middleware.RequireCorpCapability(model.CorpCapabilityInfoFittingsManage)
	requireShopWalletRead := middleware.RequireCorpCapability(model.CorpCapabilityShopWalletRead)
	requireShopOrderCreate := middleware.RequireCorpCapability(model.CorpCapabilityShopOrderCreate)
	requireShopOrderReadSelf := middleware.RequireCorpCapability(model.CorpCapabilityShopOrderReadSelf)
	requireDashboardNpcKillsCorp := middleware.RequireCorpCapability(model.CorpCapabilityDashboardNpcKillsCorp)
	requireSystemTaskRead := middleware.RequireCorpCapability(model.CorpCapabilitySystemTaskRead)
	requireSystemTaskRun := middleware.RequireCorpCapability(model.CorpCapabilitySystemTaskRun)
	requireSystemWalletRead := middleware.RequireCorpCapability(model.CorpCapabilitySystemWalletRead)
	requireSystemWalletAdjust := middleware.RequireCorpCapability(model.CorpCapabilitySystemWalletAdjust)
	requireSystemAuditRead := middleware.RequireCorpCapability(model.CorpCapabilitySystemAuditRead)
	requireSystemAuditExport := middleware.RequireCorpCapability(model.CorpCapabilitySystemAuditExport)
	requireSystemBasicConfigRead := middleware.RequireCorpCapability(model.CorpCapabilitySystemBasicConfigRead)
	requireSystemBasicConfigManage := middleware.RequireCorpCapability(model.CorpCapabilitySystemBasicConfigManage)
	requireTicketUserCreate := middleware.RequireCorpCapability(model.CorpCapabilityTicketUserCreate)
	requireTicketUserReply := middleware.RequireCorpCapability(model.CorpCapabilityTicketUserReply)
	requireTicketAdminRead := middleware.RequireCorpCapability(model.CorpCapabilityTicketAdminRead)
	requireTicketAdminManage := middleware.RequireCorpCapability(model.CorpCapabilityTicketAdminManage)
	requireShopAdminProductManage := middleware.RequireCorpCapability(model.CorpCapabilityShopAdminProductManage)
	requireShopAdminOrderManage := middleware.RequireCorpCapability(model.CorpCapabilityShopAdminOrderManage)

	// SSO 人物管理（绑定/解绑/设主人物）
	// guest 也应可访问，用于完成初次登录后的人物管理与补充授权。
	ssoAuth := auth.Group("/sso/eve")
	{
		// ssoAuth.GET("/scopes", ssoH.GetScopes)
		ssoAuth.GET("/characters", ssoH.GetMyCharacters)
		ssoAuth.GET("/bind", ssoH.BindLogin)
		ssoAuth.PUT("/primary/:character_id", ssoH.SetPrimary)
		ssoAuth.DELETE("/characters/:character_id", ssoH.Unbind)
	}

	// ─── 当前用户 ───
	meH := handler.NewMeHandler()
	auth.GET("/me", meH.GetMe)
	auth.PUT("/me", meH.UpdateMe)
	auth.DELETE("/me", meH.DeleteMe)

	dashboardH := handler.NewDashboardHandler()
	galaxyRegistryH := handler.NewGalaxyRegistryHandler()
	auth.POST("/dashboard", requireMenuDashboard, dashboardH.GetDashboard)
	corpStructureH := handler.NewCorporationStructureHandler()
	dashboard := login.Group("/dashboard")
	{
		corpStructures := dashboard.Group("/corporation-structures", requireMenuDashboard, middleware.RequireRole(model.RoleAdmin))
		corpStructures.GET("/settings", corpStructureH.GetSettings)
		corpStructures.PUT("/settings/authorizations", corpStructureH.UpdateAuthorizations)
		corpStructures.GET("/filter-options", corpStructureH.GetFilterOptions)
		corpStructures.POST("/list", corpStructureH.ListStructures)
		corpStructures.POST("/run-task", corpStructureH.RunTask)
		corpStructures.GET("/assignments", corpStructureH.GetAssignments)
		corpStructures.PUT("/assignments", corpStructureH.UpdateAssignments)
		corpStructures.GET("/fuel-salary-settings", corpStructureH.GetFuelSalarySettings)
		corpStructures.PUT("/fuel-salary-settings", corpStructureH.UpdateFuelSalarySettings)
		corpStructures.POST("/fuel-salary-payouts/run", corpStructureH.RunFuelSalaryPayout)
	}
	galaxyRegistry := dashboard.Group("/galaxy-registry", requireMenuDashboard)
	{
		galaxyRegistry.GET("/systems", galaxyRegistryH.ListSystems)

		galaxyRegistryCaptain := galaxyRegistry.Group("", middleware.RequireRole(model.RoleCaptain))
		galaxyRegistryCaptain.POST("/entries", galaxyRegistryH.CreateEntry)
		galaxyRegistryCaptain.POST("/entries/:id/end", galaxyRegistryH.EndMyEntry)
		galaxyRegistryCaptain.PUT("/entries/:id/expected-end-at", galaxyRegistryH.UpdateMyExpectedEndAt)
		galaxyRegistryCaptain.GET("/my-entries", galaxyRegistryH.ListMyEntries)

		galaxyRegistryAdmin := galaxyRegistry.Group("/admin", middleware.RequireRole(model.RoleAdmin))
		galaxyRegistryAdmin.GET("/sde-systems", galaxyRegistryH.SearchAdminSdeSystems)
		galaxyRegistryAdmin.GET("/systems", galaxyRegistryH.ListAdminSystems)
		galaxyRegistryAdmin.POST("/systems", galaxyRegistryH.CreateAdminSystem)
		galaxyRegistryAdmin.PUT("/systems/:id", galaxyRegistryH.UpdateAdminSystem)
		galaxyRegistryAdmin.DELETE("/systems/:id", galaxyRegistryH.DeleteAdminSystem)
		galaxyRegistryAdmin.GET("/entries", galaxyRegistryH.ListAdminEntries)
		galaxyRegistryAdmin.POST("/entries/:id/force-end", galaxyRegistryH.ForceEndAdminEntry)
		galaxyRegistryAdmin.POST("/entries/:id/revalidate", galaxyRegistryH.RevalidateAdminEntry)
		galaxyRegistryAdmin.PUT("/entries/:id/validation", galaxyRegistryH.UpdateAdminEntryValidation)
		galaxyRegistryAdmin.GET("/analytics", galaxyRegistryH.GetAdminAnalytics)
	}
	fuelOfficerCorpStructures := dashboard.Group(
		"/corporation-structures",
		requireMenuDashboard,
		middleware.RequireRole(model.RoleFuelOfficer),
	)
	{
		fuelOfficerCorpStructures.POST("/my-assigned-list", corpStructureH.ListMyAssignedStructures)
	}
	badgeH := handler.NewBadgeHandler()
	login.GET("/badge-counts", badgeH.GetBadgeCounts)

	// ─── 通知 ───
	notifH := handler.NewNotificationHandler()
	notification := auth.Group("/notification")
	{
		notification.POST("/list", notifH.ListNotifications)
		notification.POST("/unread-count", notifH.GetUnreadCount)
	}
	notificationWrite := login.Group("/notification")
	{
		notificationWrite.POST("/read", notifH.MarkAsRead)
		notificationWrite.POST("/read-all", notifH.MarkAllAsRead)
	}

	// ─── 舰队 ───
	fleetH := handler.NewFleetHandler()
	operation := login.Group("/operation", requireMenuOperation)
	fleet := operation.Group("/fleets")
	{
		manageFleets := middleware.RequireRole(model.RoleAdmin, model.RoleFC, model.RoleSeniorFC)
		deleteFleets := middleware.RequireRole(model.RoleAdmin)

		fleet.POST("", manageFleets, fleetH.CreateFleet)
		fleet.GET("", manageFleets, fleetH.ListFleets)
		fleet.GET("/me", fleetH.GetMyFleets)
		fleet.GET("/:id", manageFleets, fleetH.GetFleet)
		fleet.PUT("/:id", manageFleets, fleetH.UpdateFleet)
		fleet.DELETE("/:id", deleteFleets, fleetH.DeleteFleet)
		fleet.POST("/:id/refresh-esi", manageFleets, fleetH.RefreshFleetESI)

		// 成员
		fleet.GET("/:id/members", manageFleets, fleetH.GetMembers)
		fleet.GET("/:id/members-pap", manageFleets, fleetH.GetMembersWithPap)
		fleet.POST("/:id/members/manual", manageFleets, fleetH.ManualAddMembers)
		fleet.POST("/:id/members/sync", manageFleets, fleetH.SyncESIMembers)

		// ――― PAP
		fleet.POST("/:id/pap", manageFleets, fleetH.IssuePap)
		fleet.GET("/:id/pap", manageFleets, fleetH.GetPapLogs)
		fleet.GET("/pap/me", fleetH.GetMyPapLogs)
		fleet.GET("/pap/corporation", fleetH.GetCorporationPapSummary)

		// ――― 联盟 PAP
		alliancePAPH := handler.NewAlliancePAPHandler()
		fleet.GET("/pap/alliance", alliancePAPH.GetMyAlliancePAP)

		// 邀请
		fleet.POST("/:id/invites", manageFleets, fleetH.CreateInvite)
		fleet.GET("/:id/invites", manageFleets, fleetH.GetInvites)
		fleet.DELETE("/invites/:invite_id", manageFleets, fleetH.DeactivateInvite)
		fleet.POST("/join", fleetH.JoinFleet)

		// 查人物所在舰队
		fleet.GET("/esi/:character_id", fleetH.GetCharacterFleetInfo)

		// Webhook Ping（FC 或管理员手动触发）
		fleet.POST("/:id/ping", manageFleets, fleetH.PingFleet)
	}

	// ─── 舰队配置 ───
	fleetConfigH := handler.NewFleetConfigHandler()
	fleetConfig := operation.Group("/fleet-configs")
	{
		viewFleetConfigs := middleware.RequireLoginUser()
		manageFleetConfigs := middleware.RequireRole(model.RoleAdmin, model.RoleSeniorFC)

		fleetConfig.GET("", viewFleetConfigs, fleetConfigH.ListFleetConfigs)
		fleetConfig.GET("/:id", viewFleetConfigs, fleetConfigH.GetFleetConfig)
		fleetConfig.GET("/:id/eft", viewFleetConfigs, fleetConfigH.GetFittingEFT)
		fleetConfig.POST("", manageFleetConfigs, fleetConfigH.CreateFleetConfig)
		fleetConfig.PUT("/:id", manageFleetConfigs, fleetConfigH.UpdateFleetConfig)
		fleetConfig.DELETE("/:id", manageFleetConfigs, fleetConfigH.DeleteFleetConfig)
		fleetConfig.POST("/import-fitting", manageFleetConfigs, fleetConfigH.ImportFromUserFitting)
		fleetConfig.POST("/export-esi", viewFleetConfigs, fleetConfigH.ExportToESI)
		fleetConfig.GET("/:id/fittings/:fitting_id/items", viewFleetConfigs, fleetConfigH.GetFittingItems)
		fleetConfig.PUT("/:id/fittings/:fitting_id/items/settings", manageFleetConfigs, fleetConfigH.UpdateFittingItemsSettings)
	}

	// ─── 军团技能计划 ───
	skillPlanH := handler.NewSkillPlanHandler()
	skillPlanning := login.Group("/skill-planning", requireMenuSkillPlan)
	skillPlan := skillPlanning.Group("/skill-plans")
	personalSkillPlan := skillPlanning.Group("/personal-skill-plans")
	{
		viewSkillPlans := middleware.RequireLoginUser()
		manageSkillPlans := middleware.RequireRole(skillPlanManageRoles...)
		viewSkillPlanChecks := middleware.RequireLoginUser()

		skillPlan.GET("/check/selection", viewSkillPlanChecks, skillPlanH.GetCheckSelection)
		skillPlan.PUT("/check/selection", viewSkillPlanChecks, skillPlanH.SaveCheckSelection)
		skillPlan.GET("/check/plan-selection", viewSkillPlanChecks, skillPlanH.GetCheckPlanSelection)
		skillPlan.PUT("/check/plan-selection", viewSkillPlanChecks, skillPlanH.SaveCheckPlanSelection)
		skillPlan.POST("/check/run", viewSkillPlanChecks, skillPlanH.RunCompletionCheck)
		skillPlan.GET("", viewSkillPlans, skillPlanH.ListSkillPlans)
		skillPlan.GET("/:id", viewSkillPlans, skillPlanH.GetSkillPlan)
		skillPlan.POST("", manageSkillPlans, skillPlanH.CreateSkillPlan)
		skillPlan.PUT("/reorder", manageSkillPlans, skillPlanH.ReorderSkillPlans)
		skillPlan.PUT("/:id", manageSkillPlans, skillPlanH.UpdateSkillPlan)
		skillPlan.DELETE("/:id", manageSkillPlans, skillPlanH.DeleteSkillPlan)

		personalSkillPlan.GET("", viewSkillPlans, skillPlanH.ListPersonalSkillPlans)
		personalSkillPlan.GET("/:id", viewSkillPlans, skillPlanH.GetPersonalSkillPlan)
		personalSkillPlan.POST("", viewSkillPlans, skillPlanH.CreatePersonalSkillPlan)
		personalSkillPlan.PUT("/reorder", viewSkillPlans, skillPlanH.ReorderPersonalSkillPlans)
		personalSkillPlan.PUT("/:id", viewSkillPlans, skillPlanH.UpdatePersonalSkillPlan)
		personalSkillPlan.DELETE("/:id", viewSkillPlans, skillPlanH.DeletePersonalSkillPlan)
	}

	// ─── EVE 人物信息 ───
	infoH := handler.NewEveInfoHandler()
	toolBookmarkH := handler.NewToolBookmarkHandler()
	esiH := handler.NewESIRefreshHandler()
	taskH := handler.NewTaskHandler(taskSvc)
	info := login.Group("/info", requireMenuInfo)
	{
		info.GET("/tool-bookmarks", toolBookmarkH.ListVisible)
		info.POST("/wallet", requireInfoWalletRead, infoH.GetWalletJournal)
		info.POST("/skills", requireInfoSkillsRead, infoH.GetCharacterSkills)
		info.POST("/ships", infoH.GetCharacterShips)
		info.POST("/implants", infoH.GetCharacterImplants)
		info.POST("/assets", requireInfoAssetsRead, infoH.GetAssets)
			info.POST("/assets/locations", requireInfoAssetsRead, infoH.GetAssetLocations)
			info.POST("/assets/location-items", requireInfoAssetsRead, infoH.GetAssetLocationItems)
			info.POST("/assets/children", requireInfoAssetsRead, infoH.GetAssetChildren)
		info.POST("/contracts", requireInfoContractsRead, infoH.GetContracts)
		info.POST("/contracts/detail", requireInfoContractsRead, infoH.GetContractDetail)
		info.POST("/esi-refresh", esiH.RunMyCharacterTask)
	}

	// ─── 装配 ───
	fittingsH := handler.NewFittingsHandler()
	info.POST("/fittings", requireInfoFittingsManage, fittingsH.GetFittings)
	info.POST("/fittings/save", requireInfoFittingsManage, fittingsH.SaveFitting)

	// ─── NPC 刷怪报表 ───
	npcKillH := handler.NewNpcKillHandler()
	info.POST("/npc-kills", requireInfoNpcKillsSelf, npcKillH.GetNpcKills)
	info.POST("/npc-kills/all", requireInfoNpcKillsSelf, npcKillH.GetAllNpcKills)

	// ─── 新人帮扶（用户/队长） ───
	newbroUserH := handler.NewNewbroUserHandler()
	newbro := login.Group("/newbro", requireMenuNewbro)
	{
		newbro.GET("/captains", newbroUserH.ListCaptains)
		newbro.GET("/affiliation/me", newbroUserH.GetMyAffiliation)
		newbro.GET("/affiliations/history", newbroUserH.ListMyAffiliationHistory)
		newbro.POST("/affiliation/select", newbroUserH.SelectCaptain)
		newbro.POST("/affiliation/end", newbroUserH.EndAffiliation)
	}

	// ─── 招募链接（用户）───
	recruitUser := login.Group("/newbro/recruit", requireMenuNewbro)
	{
		recruitUser.POST("/link", recruitH.GenerateLink)
		recruitUser.GET("/links", recruitH.GetMyLinks)
		recruitUser.GET("/direct-referral", recruitH.GetDirectReferralStatus)
		recruitUser.POST("/direct-referral/check", recruitH.CheckDirectReferrer)
		recruitUser.POST("/direct-referral/confirm", recruitH.ConfirmDirectReferrer)
	}

	newbroCaptainH := handler.NewNewbroCaptainHandler()
	newbroCaptain := login.Group("/newbro/captain", requireMenuNewbro, middleware.RequireRole(model.RoleCaptain))
	{
		newbroCaptain.GET("/overview", newbroCaptainH.GetOverview)
		newbroCaptain.GET("/players", newbroCaptainH.GetPlayers)
		newbroCaptain.GET("/attributions", newbroCaptainH.GetAttributions)
		newbroCaptain.GET("/rewards", newbroCaptainH.GetRewardSettlements)
		newbroCaptain.GET("/eligible-players", newbroCaptainH.ListEligiblePlayers)
		newbroCaptain.POST("/enroll", newbroCaptainH.EnrollPlayer)
		newbroCaptain.POST("/affiliation/end", newbroCaptainH.EndAffiliation)
	}

	mentorMenteeH := handler.NewMentorMenteeHandler()
	mentorMentee := login.Group("/mentor", requireMenuNewbro)
	{
		mentorMentee.GET("/mentors", mentorMenteeH.ListMentors)
		mentorMentee.GET("/me", mentorMenteeH.GetMyStatus)
		mentorMentee.POST("/apply", mentorMenteeH.ApplyForMentor)
	}

	mentorMentorH := handler.NewMentorMentorHandler()
	mentorDashboard := login.Group("/mentor/dashboard", requireMenuNewbro, middleware.RequireRole(model.RoleMentor))
	{
		mentorDashboard.GET("/applications", mentorMentorH.ListPendingApplications)
		mentorDashboard.GET("/mentees", mentorMentorH.ListMyMentees)
		mentorDashboard.GET("/reward-stages", mentorMentorH.GetRewardStages)
		mentorDashboard.POST("/accept", mentorMentorH.AcceptApplication)
		mentorDashboard.POST("/reject", mentorMentorH.RejectApplication)
	}

	// ─── 商店（用户端）───
	shopH := handler.NewShopHandler()
	shop := login.Group("/shop", requireMenuShop)
	walletH := handler.NewSysWalletHandler()
	shopWallet := shop.Group("/wallet")
	{
		shop.POST("/products", shopH.ListProducts)
		shop.POST("/product/detail", shopH.GetProductDetail)
		shop.POST("/buy", requireShopOrderCreate, shopH.BuyProduct)
		shop.POST("/orders", requireShopOrderReadSelf, shopH.GetMyOrders)

		shopWallet.POST("/my", requireShopWalletRead, walletH.GetMyWallet)
		shopWallet.POST("/my/transactions", requireShopWalletRead, walletH.GetMyTransactions)
	}

	// ─── 文件上传（需要登录）───
	uploadH := handler.NewUploadHandler()
	login.POST("/upload/image", uploadH.UploadImage)

	// ─── 工单（用户端）───
	ticketH := handler.NewTicketHandler()
	ticket := login.Group("/ticket", requireMenuTicket)
	{
		ticket.POST("/tickets", requireTicketUserCreate, ticketH.CreateTicket)
		ticket.GET("/tickets/me", ticketH.ListMyTickets)
		ticket.GET("/tickets/:id", ticketH.GetMyTicket)
		ticket.POST("/tickets/:id/replies", requireTicketUserReply, ticketH.AddMyReply)
		ticket.GET("/tickets/:id/replies", ticketH.ListMyReplies)
		ticket.GET("/categories", ticketH.ListCategories)
	}

	// ─── SRP 补损 ───
	srpH := handler.NewSrpHandler()
	srp := login.Group("/srp")
	{
		requireSRPUser := middleware.RequireCorpCapability(model.CorpCapabilitySRPUser)
		requireSRPManage := middleware.RequireCorpCapability(model.CorpCapabilitySRPManage)

		// 价格表（登录用户可查看，修改仅 admin）
		srp.GET("/prices", requireSRPUser, srpH.ListShipPrices)
		srp.POST("/prices", requireSRPManage, middleware.RequireRole(srpPriceManageRoles...), srpH.UpsertShipPrice)
		srp.DELETE("/prices/:id", requireSRPManage, middleware.RequireRole(srpPriceManageRoles...), srpH.DeleteShipPrice)

		// SRP 配置（仅 admin）
		srp.GET("/config", requireSRPManage, middleware.RequireRole(model.RoleAdmin), srpH.GetSrpConfig)
		srp.PUT("/config", requireSRPManage, middleware.RequireRole(model.RoleAdmin), srpH.UpdateSrpConfig)

		// 个人申请
		srp.POST("/applications", requireSRPUser, srpH.SubmitApplication)
		srp.GET("/applications/me", requireSRPUser, srpH.ListMyApplications)
		srp.GET("/killmails/me", requireSRPUser, srpH.GetMyKillmails)
		srp.GET("/killmails/fleet/:fleet_id", requireSRPUser, srpH.GetFleetKillmails)
		srp.POST("/killmails/detail", requireSRPUser, srpH.GetKillmailDetail)
		srp.POST("/open-info-window", requireSRPUser, srpH.OpenInfoWindow)

		// 管理端操作（srp / senior_fc / admin 可查看列表 / 详情 / 审批 / 发放 / 自动审批）
		reviewSRP := middleware.RequireRole(srpManageRoles...)
		payoutSRP := middleware.RequireRole(srpPayoutRoles...)
		srp.GET("/applications", requireSRPManage, reviewSRP, srpH.ListApplications)
		srp.GET("/applications/fleet-options", requireSRPManage, reviewSRP, srpH.ListFleetOptions)
		srp.GET("/applications/:id", requireSRPManage, reviewSRP, srpH.GetApplication)
		srp.PUT("/applications/:id/review", requireSRPManage, reviewSRP, srpH.ReviewApplication)
		srp.PUT("/applications/auto-approve", requireSRPManage, payoutSRP, srpH.RunFleetAutoApproval)
		srp.GET("/applications/batch-payout-summary", requireSRPManage, payoutSRP, srpH.ListBatchPayoutSummary)
		srp.PUT("/applications/fuxi-payout", requireSRPManage, payoutSRP, srpH.BatchPayoutAsFuxiCoin)
		srp.PUT("/applications/:id/payout", requireSRPManage, payoutSRP, srpH.Payout)
		srp.PUT("/applications/users/:user_id/payout", requireSRPManage, payoutSRP, srpH.BatchPayoutByUser)
	}

	// ─── 任务管理 ───
	tasks := login.Group("/tasks", requireSystemTaskRead, middleware.RequireRole(model.RoleAdmin))
	{
		tasks.GET("", taskH.GetTasks)
		tasks.GET("/history", taskH.GetHistory)
		tasks.POST("/:name/run", requireSystemTaskRun, taskH.RunTask)
		tasks.PUT("/:name/schedule", middleware.RequireRole(model.RoleSuperAdmin), taskH.UpdateSchedule)

		esiTasks := tasks.Group("/esi")
		{
			esiTasks.GET("/tasks", esiH.GetTasks)
			esiTasks.GET("/statuses", esiH.GetStatuses)
			esiTasks.GET("/monitor", esiH.GetMonitor)
			esiTasks.POST("/run", requireSystemTaskRun, esiH.RunTask)
			esiTasks.POST("/run-task", requireSystemTaskRun, esiH.RunTaskByName)
			esiTasks.POST("/run-all", requireSystemTaskRun, esiH.RunAll)
			esiTasks.PUT("/tasks/:name/interval", middleware.RequireRole(model.RoleSuperAdmin), esiH.UpdateInterval)
		}
	}

	// ─── 系统管理（需要 admin 职权）───
	admin := login.Group("/system", middleware.RequireRole(model.RoleAdmin))

	// 系统基础配置
	sysConfigH := handler.NewSysConfigHandler()
	adminConfig := admin.Group("", requireSystemBasicConfigRead, middleware.RequireRole(systemBasicConfigManageRoles...))
	adminBasicConfig := adminConfig.Group("/basic-config")
	adminBasicConfig.GET("", sysConfigH.GetBasicConfig)

	// SDE 配置管理
	adminConfig.GET("/sde-config", sysConfigH.GetSDEConfig)
	adminConfig.PUT("/sde-config", sysConfigH.UpdateSDEConfig)
	adminConfig.GET("/sde-config/status", sysConfigH.GetSDEStatus)
	adminConfig.POST("/sde-config/check", sysConfigH.CheckSDEVersion)
	adminConfig.POST("/sde-config/update", sysConfigH.TriggerSDEUpdate)

	// 允许访问的军团列表
	adminBasicConfig.GET("/allow-corporations", sysConfigH.GetAllowCorporations)
	adminBasicConfig.PUT("/allow-corporations", requireSystemBasicConfigManage, sysConfigH.UpdateAllowCorporations)
	adminBasicConfig.GET("/corporation-access-policies", sysConfigH.GetCorporationAccessPolicies)
	adminBasicConfig.PUT("/corporation-access-policies", requireSystemBasicConfigManage, sysConfigH.UpdateCorporationAccessPolicies)
	adminBasicConfig.GET("/character-esi-restriction", sysConfigH.GetCharacterESIRestrictionConfig)
	adminBasicConfig.PUT("/character-esi-restriction", requireSystemBasicConfigManage, sysConfigH.UpdateCharacterESIRestrictionConfig)

	// NPC 刷怪报表（管理员 — 公司级）
	admin.POST("/npc-kills", requireMenuDashboard, requireDashboardNpcKillsCorp, requireInfoNpcKillsCorp, npcKillH.GetCorpNpcKills)

	// 联盟 PAP 管理（管理员）
	alliancePAPAdminH := handler.NewAlliancePAPHandler()
	alliancePAPAdmin := admin.Group("/pap", requireSystemManage)
	{
		alliancePAPAdmin.GET("", alliancePAPAdminH.GetAllAlliancePAP)
		alliancePAPAdmin.POST("/fetch", alliancePAPAdminH.TriggerFetch)
		alliancePAPAdmin.POST("/import", alliancePAPAdminH.ImportAlliancePAP)
		// 月度归档
		alliancePAPAdmin.POST("/settle", alliancePAPAdminH.SettleMonth)
	}

	// PAP 兑换汇率管理（管理员）
	papExchangeH := handler.NewPAPExchangeHandler()
	admin.GET("/pap-exchange/rates", requireSystemManage, papExchangeH.GetRates)
	admin.PUT("/pap-exchange/rates", requireSystemManage, papExchangeH.SetRates)

	// 职权定义（只读）
	roleH := handler.NewRoleHandler()
	admin.GET("/role/definitions", requireSystemManage, roleH.ListRoleDefinitions)

	// 用户管理
	userH := handler.NewUserHandler()
	adminUser := admin.Group("/user", requireSystemManage)
	{
		adminUser.GET("", userH.ListUsers)
		adminUser.GET("/:id", userH.GetUser)
		adminUser.PUT("/:id", userH.UpdateUser)
		adminUser.DELETE("/:id", userH.DeleteUser)

		// 用户职权分配
		adminUser.GET("/:id/roles", roleH.GetUserRoles)
		adminUser.PUT("/:id/roles", roleH.SetUserRoles)

		// 模拟登录（仅超级管理员）
		adminUser.POST("/:id/impersonate", middleware.RequireRole(model.RoleSuperAdmin), userH.ImpersonateUser)
	}

	newbroAdminH := handler.NewNewbroAdminHandler()
	adminNewbro := admin.Group("/newbro", requireMenuNewbro)
	{
		adminNewbro.GET("/support-settings", newbroAdminH.GetSupportSettings)
		adminNewbro.PUT("/support-settings", newbroAdminH.UpdateSupportSettings)
		adminNewbro.GET("/recruit-settings", newbroAdminH.GetRecruitSettings)
		adminNewbro.PUT("/recruit-settings", newbroAdminH.UpdateRecruitSettings)
		adminNewbro.GET("/captains", newbroAdminH.ListCaptains)
		adminNewbro.GET("/captains/:user_id", newbroAdminH.GetCaptainDetail)
		// 招募链接管理
		adminNewbro.GET("/recruit/links", recruitH.GetAdminLinks)
	}

	// 帮扶记录只读接口：管理员或队长均可访问
	newbroRecords := login.Group("/system/newbro", requireMenuNewbro, middleware.RequireRole(model.RoleAdmin, model.RoleCaptain))
	{
		newbroRecords.GET("/affiliations/history", newbroAdminH.ListAffiliationHistory)
		newbroRecords.GET("/rewards", newbroAdminH.ListRewardSettlements)
	}

	mentorAdminH := handler.NewMentorAdminHandler()
	adminMentor := admin.Group("/mentor", requireMenuNewbro)
	{
		adminMentor.GET("/relationships", mentorAdminH.ListAllRelationships)
		adminMentor.GET("/reward-distributions", mentorAdminH.ListRewardDistributions)
		adminMentor.POST("/revoke", mentorAdminH.RevokeRelationship)
		adminMentor.GET("/settings", mentorAdminH.GetSettings)
		adminMentor.PUT("/settings", mentorAdminH.UpdateSettings)
		adminMentor.GET("/reward-stages", mentorAdminH.GetRewardStages)
		adminMentor.PUT("/reward-stages", mentorAdminH.UpdateRewardStages)
		adminMentor.POST("/reward/process", mentorAdminH.RunRewardProcessing)
	}

	// 伏羲币管理（管理员）
	adminWalletH := handler.NewSysWalletHandler()
	adminWallet := admin.Group("/wallet", requireSystemWalletRead)
	{
		adminWallet.POST("/list", adminWalletH.AdminListWallets)
		adminWallet.POST("/detail", adminWalletH.AdminGetWallet)
		adminWallet.POST("/adjust", requireSystemWalletAdjust, adminWalletH.AdminAdjust)
		adminWallet.POST("/transactions", adminWalletH.AdminListTransactions)
		adminWallet.POST("/logs", adminWalletH.AdminListLogs)
		adminWallet.POST("/analytics", adminWalletH.AdminAnalytics)
	}
	auditEventH := handler.NewAuditEventHandler()
	adminAudit := admin.Group("/audit", requireSystemAuditRead)
	{
		adminAudit.POST("/events", auditEventH.AdminList)
		adminAudit.POST("/export", requireSystemAuditExport, auditEventH.CreateExportTask)
		adminAudit.GET("/export/:task_id", auditEventH.GetExportTaskStatus)
		adminAudit.POST("/export/list", auditEventH.ListExportTasks)
	}

	adminToolBookmark := admin.Group("/tool-bookmarks", requireSystemManage)
	{
		adminToolBookmark.GET("", toolBookmarkH.AdminList)
		adminToolBookmark.POST("", toolBookmarkH.AdminCreate)
		adminToolBookmark.PUT("/:id", toolBookmarkH.AdminUpdate)
		adminToolBookmark.DELETE("/:id", toolBookmarkH.AdminDelete)
	}

	// 商店管理（管理员）
	adminShopH := handler.NewShopHandler()
	adminShopProduct := admin.Group("/shop/product", requireShopManage, requireShopAdminProductManage)
	{
		adminShopProduct.POST("/list", adminShopH.AdminListProducts)
		adminShopProduct.POST("/add", adminShopH.AdminCreateProduct)
		adminShopProduct.POST("/edit", adminShopH.AdminUpdateProduct)
		adminShopProduct.POST("/delete", adminShopH.AdminDeleteProduct)
	}
	// 商店订单（仅管理员）
	shopOrder := login.Group(
		"/system/shop/order",
		requireShopManage,
		requireShopAdminOrderManage,
		middleware.RequireRole(shopOrderManageRoles...),
	)
	{
		shopOrder.POST("/list", adminShopH.AdminListOrders)
		shopOrder.POST("/deliver", adminShopH.AdminDeliverOrder)
		shopOrder.POST("/reject", adminShopH.AdminRejectOrder)
	}

	// 福利管理（列表：admin + welfare 可读；写操作仅 admin）
	welfareListGroup := login.Group(
		"/system/welfare",
		middleware.RequireCorpCapability(model.CorpCapabilityWelfareUser),
		middleware.RequireRole(model.RoleAdmin, model.RoleWelfare),
	)
	welfareListGroup.POST("/list", welfareH.AdminListWelfares)

	welfareApproval := login.Group(
		"/system/welfare",
		middleware.RequireCorpCapability(model.CorpCapabilityWelfareReview),
		middleware.RequireRole(welfareApprovalRoles...),
	)
	{
		welfareApproval.POST("/applications", welfareH.AdminListApplications)
		welfareApproval.POST("/review", welfareH.AdminReviewApplication)
	}

	adminWelfare := admin.Group("/welfare")
	{
		adminWelfare.GET("/settings", welfareH.AdminGetSettings)
		adminWelfare.PUT("/settings", middleware.RequireCorpCapability(model.CorpCapabilityWelfareConfig), welfareH.AdminUpdateSettings)
		adminWelfare.POST("/add", middleware.RequireCorpCapability(model.CorpCapabilityWelfareConfig), welfareH.AdminCreateWelfare)
		adminWelfare.POST("/edit", middleware.RequireCorpCapability(model.CorpCapabilityWelfareConfig), welfareH.AdminUpdateWelfare)
		adminWelfare.POST("/delete", middleware.RequireCorpCapability(model.CorpCapabilityWelfareConfig), welfareH.AdminDeleteWelfare)
		adminWelfare.POST("/applications/delete", middleware.RequireCorpCapability(model.CorpCapabilityWelfareReview), welfareH.AdminDeleteApplication)
		adminWelfare.POST("/import", middleware.RequireCorpCapability(model.CorpCapabilityWelfareConfig), welfareH.AdminImportRecords)
		adminWelfare.POST("/reorder", middleware.RequireCorpCapability(model.CorpCapabilityWelfareConfig), welfareH.AdminReorderWelfares)
	}

	// ─── 用户端福利 ───
	welfareUser := login.Group("/welfare")
	{
		welfareUser.POST("/eligible", middleware.RequireCorpCapability(model.CorpCapabilityWelfareUser), welfareH.GetEligibleWelfares)
		welfareUser.POST("/apply", middleware.RequireCorpCapability(model.CorpCapabilityWelfareUser), welfareH.ApplyForWelfare)
		welfareUser.POST("/my-applications", middleware.RequireCorpCapability(model.CorpCapabilityWelfareUser), welfareH.ListMyApplications)
		welfareUser.POST("/upload-evidence", middleware.RequireCorpCapability(model.CorpCapabilityWelfareUser), welfareH.UploadEvidence)
	}

	// 自动权限映射管理（管理员）
	autoRoleH := handler.NewAutoRoleHandler()
	registerAdminAutoRoleRoutes(admin.Group("", requireSystemManage), autoRoleH)

	// Webhook 配置（仅 super_admin）
	webhookH := handler.NewWebhookHandler()
	adminWebhook := admin.Group("/webhook", requireSystemManage, middleware.RequireRole(systemWebhookManageRoles...))
	{
		adminWebhook.GET("/config", webhookH.GetConfig)
		adminWebhook.PUT("/config", webhookH.SetConfig)
		adminWebhook.POST("/test", webhookH.TestWebhook)
	}

	// ─── 伏羲大厅 ───
	fuxiHallH := handler.NewFuxiHallHandler()

	// 登录可见展示页
	loginFuxiHall := login.Group("/fuxi-hall", requireMenuFuxiHall)
	{
		loginFuxiHall.GET("/leadership", fuxiHallH.GetLeadership)
		loginFuxiHall.GET("/contributors", fuxiHallH.GetContributors)
	}

	// 管理端点（admin group 已要求 RoleAdmin）
	adminFuxiHall := admin.Group("/fuxi-hall", requireMenuFuxiHall)
	{
		adminFuxiHall.GET("/pages/:page_key", fuxiHallH.GetPageConfig)
		adminFuxiHall.PUT("/pages/:page_key", fuxiHallH.UpdatePageConfig)
		adminFuxiHall.GET("/cards/:page_key", fuxiHallH.ListCards)
		adminFuxiHall.POST("/cards", fuxiHallH.CreateCard)
		adminFuxiHall.PUT("/cards/reorder", fuxiHallH.ReorderCards)
		adminFuxiHall.PUT("/cards/:id", fuxiHallH.UpdateCard)
		adminFuxiHall.DELETE("/cards/:id", fuxiHallH.DeleteCard)
	}

	// ─── 工单管理（管理员）───
	adminTicket := admin.Group("/ticket", requireTicketManage)
	{
		adminTicket.GET("/tickets", requireTicketAdminRead, ticketH.AdminListTickets)
		adminTicket.GET("/tickets/:id", requireTicketAdminRead, ticketH.AdminGetTicket)
		adminTicket.PUT("/tickets/:id/status", requireTicketAdminManage, ticketH.AdminUpdateStatus)
		adminTicket.POST("/tickets/:id/replies", requireTicketAdminManage, ticketH.AdminAddReply)
		adminTicket.GET("/tickets/:id/replies", ticketH.AdminListReplies)
		adminTicket.GET("/tickets/:id/status-history", ticketH.AdminListStatusHistory)

		adminTicket.GET("/categories", requireTicketAdminRead, ticketH.AdminListCategories)
		adminTicket.POST("/categories", requireTicketAdminManage, ticketH.AdminCreateCategory)
		adminTicket.PUT("/categories/:id", requireTicketAdminManage, ticketH.AdminUpdateCategory)
		adminTicket.DELETE("/categories/:id", requireTicketAdminManage, ticketH.AdminDeleteCategory)
		adminTicket.GET("/statistics", requireTicketAdminRead, ticketH.AdminStatistics)
	}
}

func registerAdminAutoRoleRoutes(admin *gin.RouterGroup, autoRoleH *handler.AutoRoleHandler) {
	adminAutoRole := admin.Group("/auto-role", middleware.RequireRole(autoRoleManageRoles...))
	{
		// ESI 军团职权映射
		adminAutoRole.GET("/esi-roles", autoRoleH.GetAllEsiRoles)
		adminAutoRole.GET("/esi-role-mappings", autoRoleH.ListEsiRoleMappings)
		adminAutoRole.POST("/esi-role-mappings", autoRoleH.CreateEsiRoleMapping)
		adminAutoRole.DELETE("/esi-role-mappings/:id", autoRoleH.DeleteEsiRoleMapping)

		// ESI 头衔映射
		adminAutoRole.GET("/corp-titles", autoRoleH.ListCorpTitles)
		adminAutoRole.GET("/esi-title-mappings", autoRoleH.ListEsiTitleMappings)
		adminAutoRole.POST("/esi-title-mappings", autoRoleH.CreateEsiTitleMapping)
		adminAutoRole.DELETE("/esi-title-mappings/:id", autoRoleH.DeleteEsiTitleMapping)

		// 手动触发同步
		adminAutoRole.POST("/sync", autoRoleH.TriggerSync)
	}
}
