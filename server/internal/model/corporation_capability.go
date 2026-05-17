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
)

var validCorpCapabilities = map[string]struct{}{
	CorpCapabilitySRPUser:       {},
	CorpCapabilitySRPManage:     {},
	CorpCapabilityWelfareUser:   {},
	CorpCapabilityWelfareReview: {},
	CorpCapabilityWelfareConfig: {},
	CorpCapabilityMenuSRP:       {},
	CorpCapabilityMenuWelfare:   {},
	CorpCapabilityMenuDashboard: {},
	CorpCapabilityMenuOperation: {},
	CorpCapabilityMenuRole:      {},
	CorpCapabilityMenuNewbro:    {},
	CorpCapabilityMenuFuxiHall:  {},
	CorpCapabilityMenuTicket:    {},
	CorpCapabilityMenuShop:      {},
	CorpCapabilityMenuSystem:    {},
	CorpCapabilityMenuInfo:      {},
	CorpCapabilityMenuSkillPlan: {},
	CorpCapabilityTicketManage:  {},
	CorpCapabilityShopManage:    {},
	CorpCapabilitySystemManage:  {},
}

func IsValidCorpCapability(capability string) bool {
	_, ok := validCorpCapabilities[capability]
	return ok
}
