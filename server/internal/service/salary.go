package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"

	"gorm.io/gorm"
)

type SalarySettings struct {
	Recruitment          float64 `json:"recruitment"`
	WelfareDelivery      float64 `json:"welfare_delivery"`
	ShopDelivery         float64 `json:"shop_delivery"`
	SRPPayout            float64 `json:"srp_payout"`
	FCSalary             float64 `json:"fc_salary"`
	FCSalaryMonthlyLimit int     `json:"fc_salary_monthly_limit"`
}

type SalaryService struct {
	repo *repository.SysConfigRepository
}

type OperationSalaryRates struct {
	Recruitment     float64 `json:"recruitment"`
	WelfareDelivery float64 `json:"welfare_delivery"`
	ShopDelivery    float64 `json:"shop_delivery"`
	SRPPayout       float64 `json:"srp_payout"`
}

// Read-only operation rates are available to signed-in operators for their
// confirmation dialogs. FC limits and configuration writes remain admin-only.
func (s *SalaryService) OperationRates() (*OperationSalaryRates, error) {
	rates := &OperationSalaryRates{}
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range []struct {
			key    string
			target *float64
		}{
			{model.SysConfigSalaryRecruitment, &rates.Recruitment},
			{model.SysConfigSalaryWelfare, &rates.WelfareDelivery},
			{model.SysConfigSalaryShop, &rates.ShopDelivery},
			{model.SysConfigSalarySRP, &rates.SRPPayout},
		} {
			amount, err := salaryAmountTx(tx, item.key, model.DefaultOperationSalary)
			if err != nil {
				return err
			}
			*item.target = amount
		}
		return nil
	})
	return rates, err
}

func NewSalaryService() *SalaryService {
	return &SalaryService{repo: repository.NewSysConfigRepository()}
}

func (s *SalaryService) Settings() (*SalarySettings, error) {
	settings := &SalarySettings{}
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range []struct {
			key      string
			target   *float64
			fallback float64
		}{
			{model.SysConfigSalaryRecruitment, &settings.Recruitment, model.DefaultOperationSalary},
			{model.SysConfigSalaryWelfare, &settings.WelfareDelivery, model.DefaultOperationSalary},
			{model.SysConfigSalaryShop, &settings.ShopDelivery, model.DefaultOperationSalary},
			{model.SysConfigSalarySRP, &settings.SRPPayout, model.DefaultOperationSalary},
			{model.SysConfigPAPFCSalary, &settings.FCSalary, model.SysConfigDefaultPAPFCSalary},
		} {
			amount, err := salaryAmountTx(tx, item.key, item.fallback)
			if err != nil {
				return err
			}
			*item.target = amount
		}
		settings.FCSalaryMonthlyLimit = s.repo.GetIntTx(tx, model.SysConfigPAPFCSalaryLimit, model.SysConfigDefaultPAPFCSalaryLimit)
		return nil
	})
	return settings, err
}

func (s *SalaryService) Update(settings SalarySettings, operatorID uint) (*SalarySettings, error) {
	for _, amount := range []float64{settings.Recruitment, settings.WelfareDelivery, settings.ShopDelivery, settings.SRPPayout, settings.FCSalary} {
		if !validSalaryAmount(amount) {
			return nil, errors.New("工资必须为非负金额，最多保留两位小数")
		}
	}
	if settings.FCSalaryMonthlyLimit < 0 {
		return nil, errors.New("FC 工资月度次数上限不能小于 0")
	}
	items := newSysConfigBatch(6).
		AddFloat64(model.SysConfigSalaryRecruitment, settings.Recruitment, "招新工资").
		AddFloat64(model.SysConfigSalaryWelfare, settings.WelfareDelivery, "福利发放工资").
		AddFloat64(model.SysConfigSalaryShop, settings.ShopDelivery, "商城发货工资").
		AddFloat64(model.SysConfigSalarySRP, settings.SRPPayout, "补损发放工资").
		AddFloat64(model.SysConfigPAPFCSalary, settings.FCSalary, "FC 工资").
		AddInt(model.SysConfigPAPFCSalaryLimit, settings.FCSalaryMonthlyLimit, "FC 工资每月上限次数").Items()
	if err := s.repo.SetMany(items); err != nil {
		return nil, err
	}
	_ = NewAuditService().RecordEvent(context.Background(), AuditRecordInput{
		Category: "config", Action: "salary_settings_update", ActorUserID: operatorID,
		ResourceType: "system_config", ResourceID: "salary", Result: model.AuditResultSuccess,
		Details: map[string]any{"settings": settings},
	})
	return &settings, nil
}

func validSalaryAmount(amount float64) bool {
	return !math.IsNaN(amount) && !math.IsInf(amount, 0) && amount >= 0 && amount <= 1e12 && math.Abs(amount*100-math.Round(amount*100)) < 0.001
}

// Read through the source completion transaction. A database failure must not
// silently pay a default or leave completion without its configured salary.
func salaryAmountTx(tx *gorm.DB, key string, fallback float64) (float64, error) {
	raw, err := repository.NewSysConfigRepository().GetTx(tx, key, strconv.FormatFloat(fallback, 'f', -1, 64))
	if err != nil {
		return 0, err
	}
	amount, err := strconv.ParseFloat(raw, 64)
	if err != nil || !validSalaryAmount(amount) {
		return 0, errors.New("工资配置无效，请联系管理员调整配置")
	}
	return amount, nil
}

// The caller holds the source row lock and commits its terminal transition in
// this transaction. Automatic paths must not call this human-operation helper.
func payOperationSalaryTx(tx *gorm.DB, operatorID uint, key, refType, refID, reason string) error {
	if operatorID == 0 {
		return nil
	}
	amount, err := salaryAmountTx(tx, key, model.DefaultOperationSalary)
	if err != nil {
		return err
	}
	if err := NewSysWalletService().ApplyWalletDeltaByOperatorTx(tx, operatorID, operatorID, amount, reason, refType, refID); err != nil {
		return fmt.Errorf("发放操作工资失败: %w", err)
	}
	return nil
}
