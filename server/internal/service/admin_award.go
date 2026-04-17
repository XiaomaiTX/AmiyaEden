package service

import (
	"amiya-eden/internal/model"
	"fmt"

	"gorm.io/gorm"
)

type adminAwardConfigReader interface {
	GetInt(key string, defaultVal int) int
}

func isAdminAwardEligible(roleCodes []string) bool {
	return model.IsSuperAdmin(roleCodes) || model.ContainsRole(roleCodes, model.RoleAdmin)
}

func configuredAdminAward(cfg adminAwardConfigReader) int {
	if cfg == nil {
		return model.SysConfigDefaultPAPAdminAward
	}

	return cfg.GetInt(model.SysConfigPAPAdminAward, model.SysConfigDefaultPAPAdminAward)
}

func applyConfiguredAdminAwardTx(
	tx *gorm.DB,
	cfg adminAwardConfigReader,
	operatorRoles []string,
	operatorID uint,
	reason string,
	refID string,
) error {
	if tx == nil || operatorID == 0 || !isAdminAwardEligible(operatorRoles) {
		return nil
	}

	award := configuredAdminAward(cfg)
	if award <= 0 {
		return nil
	}

	return NewSysWalletService().ApplyWalletDeltaTx(
		tx,
		operatorID,
		float64(award),
		reason,
		model.WalletRefAdminAward,
		refID,
	)
}

func buildWelfareAdminAwardRefID(appID uint) string {
	return fmt.Sprintf("admin_welfare_delivery:%d", appID)
}

func buildShopAdminAwardRefID(orderID uint) string {
	return fmt.Sprintf("admin_shop_delivery:%d", orderID)
}
