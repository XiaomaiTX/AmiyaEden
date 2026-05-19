package model

const (
	CorpCapabilitySRPUser       = "srp.user"
	CorpCapabilitySRPManage     = "srp.manage"
	CorpCapabilityWelfareUser   = "welfare.user"
	CorpCapabilityWelfareReview = "welfare.approval"
	CorpCapabilityWelfareConfig = "welfare.settings"
	CorpCapabilityMenuSRP       = "menu.srp"
	CorpCapabilityMenuWelfare   = "menu.welfare"
	CorpCapabilityMenuDashboard = "menu.dashboard"
	CorpCapabilityMenuOperation = "menu.operation"
	CorpCapabilityMenuRole      = "menu.role"
	CorpCapabilityMenuNewbro    = "menu.newbro"
	CorpCapabilityMenuFuxiHall  = "menu.fuxi_hall"
	CorpCapabilityMenuTicket    = "menu.ticket"
	CorpCapabilityMenuShop      = "menu.shop"
	CorpCapabilityMenuSystem    = "menu.system"
	CorpCapabilityMenuInfo      = "menu.info"
	CorpCapabilityMenuSkillPlan = "menu.skill_planning"
	CorpCapabilityTicketManage  = "ticket.manage"
	CorpCapabilityShopManage    = "shop.manage"
	CorpCapabilitySystemManage  = "system.manage"

	CorpCapabilityInfoWalletRead              = "info.wallet.read"
	CorpCapabilityInfoNpcKillsSelf            = "info.npc_kills.self"
	CorpCapabilityInfoNpcKillsCorp            = "info.npc_kills.corp"
	CorpCapabilityInfoSkillsRead              = "info.skills.read"
	CorpCapabilityInfoAssetsRead              = "info.assets.read"
	CorpCapabilityInfoContractsRead           = "info.contracts.read"
	CorpCapabilityInfoFittingsManage          = "info.fittings.manage"
	CorpCapabilityShopWalletRead              = "shop.wallet.read"
	CorpCapabilityWalletUserEnabled           = "wallet.user.enabled"
	CorpCapabilityShopOrderCreate             = "shop.order.create"
	CorpCapabilityShopOrderReadSelf           = "shop.order.read_self"
	CorpCapabilityDashboardNpcKillsCorp       = "dashboard.npc_kills.corp"
	CorpCapabilityDashboardCorpStructRead     = "dashboard.corp_structures.read"
	CorpCapabilityDashboardCorpStructManage   = "dashboard.corp_structures.manage"
	CorpCapabilityOperationFleetReadSelf      = "operation.fleet.read_self"
	CorpCapabilityOperationFleetManage        = "operation.fleet.manage"
	CorpCapabilityOperationFleetPapManage     = "operation.fleet.pap.manage"
	CorpCapabilitySkillPlanningCorpRead       = "skill_planning.corp.read"
	CorpCapabilitySkillPlanningCorpManage     = "skill_planning.corp.manage"
	CorpCapabilitySkillPlanningPersonalRead   = "skill_planning.personal.read"
	CorpCapabilitySkillPlanningPersonalManage = "skill_planning.personal.manage"
	CorpCapabilityNewbroUserActions           = "newbro.user.actions"
	CorpCapabilityNewbroCaptainActions        = "newbro.captain.actions"
	CorpCapabilityNewbroAdminRead             = "newbro.admin.read"
	CorpCapabilityNewbroAdminManage           = "newbro.admin.manage"
	CorpCapabilityMentorUserActions           = "mentor.user.actions"
	CorpCapabilityMentorMentorActions         = "mentor.mentor.actions"
	CorpCapabilityMentorAdminManage           = "mentor.admin.manage"
	CorpCapabilitySystemTaskRead              = "system.task.read"
	CorpCapabilitySystemTaskRun               = "system.task.run"
	CorpCapabilitySystemBasicConfigRead       = "system.basic_config.read"
	CorpCapabilitySystemBasicConfigManage     = "system.basic_config.manage"
	CorpCapabilitySystemWalletRead            = "system.wallet.read"
	CorpCapabilitySystemWalletAdjust          = "system.wallet.adjust"
	CorpCapabilitySystemAuditRead             = "system.audit.read"
	CorpCapabilitySystemAuditExport           = "system.audit.export"
	CorpCapabilitySystemToolBookmarkRead      = "system.tool_bookmark.read"
	CorpCapabilitySystemToolBookmarkManage    = "system.tool_bookmark.manage"
	CorpCapabilityTicketUserCreate            = "ticket.user.create"
	CorpCapabilityTicketUserReply             = "ticket.user.reply"
	CorpCapabilityTicketAdminRead             = "ticket.admin.read"
	CorpCapabilityTicketAdminManage           = "ticket.admin.manage"
	CorpCapabilityShopAdminProductManage      = "shop.admin.product.manage"
	CorpCapabilityShopAdminOrderManage        = "shop.admin.order.manage"
	CorpCapabilityFuxiHallPublicRead          = "fuxi_hall.public.read"
	CorpCapabilityFuxiHallAdminManage         = "fuxi_hall.admin.manage"
)

var validCorpCapabilities = map[string]struct{}{
	CorpCapabilitySRPUser:                     {},
	CorpCapabilitySRPManage:                   {},
	CorpCapabilityWelfareUser:                 {},
	CorpCapabilityWelfareReview:               {},
	CorpCapabilityWelfareConfig:               {},
	CorpCapabilityMenuSRP:                     {},
	CorpCapabilityMenuWelfare:                 {},
	CorpCapabilityMenuDashboard:               {},
	CorpCapabilityMenuOperation:               {},
	CorpCapabilityMenuRole:                    {},
	CorpCapabilityMenuNewbro:                  {},
	CorpCapabilityMenuFuxiHall:                {},
	CorpCapabilityMenuTicket:                  {},
	CorpCapabilityMenuShop:                    {},
	CorpCapabilityMenuSystem:                  {},
	CorpCapabilityMenuInfo:                    {},
	CorpCapabilityMenuSkillPlan:               {},
	CorpCapabilityTicketManage:                {},
	CorpCapabilityShopManage:                  {},
	CorpCapabilitySystemManage:                {},
	CorpCapabilityInfoWalletRead:              {},
	CorpCapabilityInfoNpcKillsSelf:            {},
	CorpCapabilityInfoNpcKillsCorp:            {},
	CorpCapabilityInfoSkillsRead:              {},
	CorpCapabilityInfoAssetsRead:              {},
	CorpCapabilityInfoContractsRead:           {},
	CorpCapabilityInfoFittingsManage:          {},
	CorpCapabilityShopWalletRead:              {},
	CorpCapabilityWalletUserEnabled:           {},
	CorpCapabilityShopOrderCreate:             {},
	CorpCapabilityShopOrderReadSelf:           {},
	CorpCapabilityDashboardNpcKillsCorp:       {},
	CorpCapabilityDashboardCorpStructRead:     {},
	CorpCapabilityDashboardCorpStructManage:   {},
	CorpCapabilityOperationFleetReadSelf:      {},
	CorpCapabilityOperationFleetManage:        {},
	CorpCapabilityOperationFleetPapManage:     {},
	CorpCapabilitySkillPlanningCorpRead:       {},
	CorpCapabilitySkillPlanningCorpManage:     {},
	CorpCapabilitySkillPlanningPersonalRead:   {},
	CorpCapabilitySkillPlanningPersonalManage: {},
	CorpCapabilityNewbroUserActions:           {},
	CorpCapabilityNewbroCaptainActions:        {},
	CorpCapabilityNewbroAdminRead:             {},
	CorpCapabilityNewbroAdminManage:           {},
	CorpCapabilityMentorUserActions:           {},
	CorpCapabilityMentorMentorActions:         {},
	CorpCapabilityMentorAdminManage:           {},
	CorpCapabilitySystemTaskRead:              {},
	CorpCapabilitySystemTaskRun:               {},
	CorpCapabilitySystemBasicConfigRead:       {},
	CorpCapabilitySystemBasicConfigManage:     {},
	CorpCapabilitySystemWalletRead:            {},
	CorpCapabilitySystemWalletAdjust:          {},
	CorpCapabilitySystemAuditRead:             {},
	CorpCapabilitySystemAuditExport:           {},
	CorpCapabilitySystemToolBookmarkRead:      {},
	CorpCapabilitySystemToolBookmarkManage:    {},
	CorpCapabilityTicketUserCreate:            {},
	CorpCapabilityTicketUserReply:             {},
	CorpCapabilityTicketAdminRead:             {},
	CorpCapabilityTicketAdminManage:           {},
	CorpCapabilityShopAdminProductManage:      {},
	CorpCapabilityShopAdminOrderManage:        {},
	CorpCapabilityFuxiHallPublicRead:          {},
	CorpCapabilityFuxiHallAdminManage:         {},
}

func IsValidCorpCapability(capability string) bool {
	_, ok := validCorpCapabilities[capability]
	return ok
}
