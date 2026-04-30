package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/json"
)

// PAPExchangeService PAP 兑换汇率业务逻辑层
type PAPExchangeService struct {
	rateRepo   papExchangeRateStore
	configRepo papExchangeConfigStore
	auditSvc   *AuditService
}

type papExchangeRateStore interface {
	List() ([]model.PAPTypeRate, error)
	Save(rates []model.PAPTypeRate) error
}

type papExchangeConfigStore interface {
	GetFloat(key string, defaultVal float64) float64
	GetInt(key string, defaultVal int) int
	SetMany(items []repository.SysConfigUpsertItem) error
}

func NewPAPExchangeService() *PAPExchangeService {
	return &PAPExchangeService{
		rateRepo:   repository.NewPAPTypeRateRepository(),
		configRepo: repository.NewSysConfigRepository(),
		auditSvc:   NewAuditService(),
	}
}

// PAPExchangeConfigResponse PAP 兑换配置响应
type PAPExchangeConfigResponse struct {
	Rates                       []model.PAPTypeRate `json:"rates"`
	FCSalary                    float64             `json:"fc_salary"`
	FCSalaryMonthlyLimit        int                 `json:"fc_salary_monthly_limit"`
	AdminAward                  int                 `json:"admin_award"`
	MulticharFullRewardCount    int                 `json:"multichar_full_reward_count"`
	MulticharReducedRewardCount int                 `json:"multichar_reduced_reward_count"`
	MulticharReducedRewardPct   int                 `json:"multichar_reduced_reward_pct"`
}

// GetConfig 获取所有 PAP 类型兑换汇率与 FC 工资
func (s *PAPExchangeService) GetConfig() (*PAPExchangeConfigResponse, error) {
	rates, err := s.rateRepo.List()
	if err != nil {
		return nil, err
	}
	tierCfg := getMulticharRewardConfig(s.configRepo)
	return &PAPExchangeConfigResponse{
		Rates:                       rates,
		FCSalary:                    s.getFCSalary(),
		FCSalaryMonthlyLimit:        s.getFCSalaryMonthlyLimit(),
		AdminAward:                  s.getAdminAward(),
		MulticharFullRewardCount:    tierCfg.FullRewardCount,
		MulticharReducedRewardCount: tierCfg.ReducedRewardCount,
		MulticharReducedRewardPct:   tierCfg.ReducedRewardPct,
	}, nil
}

// SetRateRequest 更新 PAP 类型汇率的请求项
type SetRateRequest struct {
	PapType     string  `json:"pap_type"     binding:"required"`
	DisplayName string  `json:"display_name"`
	Rate        float64 `json:"rate"         binding:"required,gt=0"`
}

// UpdateConfigRequest 更新 PAP 兑换配置请求
type UpdateConfigRequest struct {
	Rates                       []SetRateRequest `json:"rates" binding:"required"`
	FCSalary                    *float64         `json:"fc_salary" binding:"required,gte=0"`
	FCSalaryMonthlyLimit        *int             `json:"fc_salary_monthly_limit" binding:"required,gte=0"`
	AdminAward                  *int             `json:"admin_award" binding:"required,gte=0"`
	MulticharFullRewardCount    *int             `json:"multichar_full_reward_count" binding:"required,gte=0"`
	MulticharReducedRewardCount *int             `json:"multichar_reduced_reward_count" binding:"required,gte=0"`
	MulticharReducedRewardPct   *int             `json:"multichar_reduced_reward_pct" binding:"required,gte=0,lte=100"`
}

// UpdateConfig 批量更新 PAP 类型兑换汇率与 FC 工资
func (s *PAPExchangeService) UpdateConfig(req *UpdateConfigRequest) (*PAPExchangeConfigResponse, error) {
	if err := s.SetRates(req.Rates); err != nil {
		return nil, err
	}
	items := newSysConfigBatch(6).
		AddFloat64(model.SysConfigPAPFCSalary, *req.FCSalary, "FC工资").
		AddInt(model.SysConfigPAPFCSalaryLimit, *req.FCSalaryMonthlyLimit, "FC工资每月上限次数").
		AddInt(model.SysConfigPAPAdminAward, *req.AdminAward, "管理发放奖励").
		AddInt(model.SysConfigMulticharFullRewardCount, *req.MulticharFullRewardCount, "多人物满额奖励人物数").
		AddInt(model.SysConfigMulticharReducedRewardCount, *req.MulticharReducedRewardCount, "多人物折扣奖励人物数").
		AddInt(model.SysConfigMulticharReducedRewardPct, *req.MulticharReducedRewardPct, "多人物折扣奖励百分比").
		Items()
	if err := s.configRepo.SetMany(items); err != nil {
		return nil, err
	}
	return s.GetConfig()
}

func (s *PAPExchangeService) UpdateConfigByOperator(req *UpdateConfigRequest, operatorID uint) (*PAPExchangeConfigResponse, error) {
	updated, err := s.UpdateConfig(req)
	if err != nil {
		return nil, err
	}
	if s.auditSvc != nil {
		details := map[string]any{
			"operator_id":                     operatorID,
			"rates":                           req.Rates,
			"fc_salary":                       req.FCSalary,
			"fc_salary_monthly_limit":         req.FCSalaryMonthlyLimit,
			"admin_award":                     req.AdminAward,
			"multichar_full_reward_count":     req.MulticharFullRewardCount,
			"multichar_reduced_reward_count":  req.MulticharReducedRewardCount,
			"multichar_reduced_reward_pct":    req.MulticharReducedRewardPct,
			"updated_config_snapshot_compact": compactPAPConfigForAudit(updated),
		}
		_ = s.auditSvc.RecordEvent(context.Background(), AuditRecordInput{
			Category:     "config",
			Action:       "pap_exchange_config_update",
			ActorUserID:  operatorID,
			ResourceType: "system_config",
			ResourceID:   model.SysConfigPAPFCSalary,
			Result:       model.AuditResultSuccess,
			Details:      details,
		})
	}
	return updated, nil
}

func compactPAPConfigForAudit(cfg *PAPExchangeConfigResponse) string {
	if cfg == nil {
		return ""
	}
	blob, err := json.Marshal(map[string]any{
		"fc_salary":                      cfg.FCSalary,
		"fc_salary_monthly_limit":        cfg.FCSalaryMonthlyLimit,
		"admin_award":                    cfg.AdminAward,
		"multichar_full_reward_count":    cfg.MulticharFullRewardCount,
		"multichar_reduced_reward_count": cfg.MulticharReducedRewardCount,
		"multichar_reduced_reward_pct":   cfg.MulticharReducedRewardPct,
		"rate_count":                     len(cfg.Rates),
	})
	if err != nil {
		return ""
	}
	return string(blob)
}

// SetRates 批量更新 PAP 类型兑换汇率
func (s *PAPExchangeService) SetRates(items []SetRateRequest) error {
	rates := make([]model.PAPTypeRate, 0, len(items))
	for _, item := range items {
		rates = append(rates, model.PAPTypeRate{
			PapType:     item.PapType,
			DisplayName: item.DisplayName,
			Rate:        item.Rate,
		})
	}
	return s.rateRepo.Save(rates)
}

func (s *PAPExchangeService) getFCSalary() float64 {
	return s.configRepo.GetFloat(model.SysConfigPAPFCSalary, model.SysConfigDefaultPAPFCSalary)
}

func (s *PAPExchangeService) getFCSalaryMonthlyLimit() int {
	return s.configRepo.GetInt(model.SysConfigPAPFCSalaryLimit, model.SysConfigDefaultPAPFCSalaryLimit)
}

func (s *PAPExchangeService) getAdminAward() int {
	return configuredAdminAward(s.configRepo)
}
