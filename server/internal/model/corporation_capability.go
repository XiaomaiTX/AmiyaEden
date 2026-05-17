package model

const (
	CorpCapabilitySRPUser       = "srp.user"
	CorpCapabilitySRPManage     = "srp.manage"
	CorpCapabilityWelfareUser   = "welfare.user"
	CorpCapabilityWelfareReview = "welfare.approval"
	CorpCapabilityWelfareConfig = "welfare.settings"
	CorpCapabilityMenuSRP       = "menu.srp"
	CorpCapabilityMenuWelfare   = "menu.welfare"
)

var validCorpCapabilities = map[string]struct{}{
	CorpCapabilitySRPUser:       {},
	CorpCapabilitySRPManage:     {},
	CorpCapabilityWelfareUser:   {},
	CorpCapabilityWelfareReview: {},
	CorpCapabilityWelfareConfig: {},
	CorpCapabilityMenuSRP:       {},
	CorpCapabilityMenuWelfare:   {},
}

func IsValidCorpCapability(capability string) bool {
	_, ok := validCorpCapabilities[capability]
	return ok
}
