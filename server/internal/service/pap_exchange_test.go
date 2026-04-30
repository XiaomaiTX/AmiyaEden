package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"strconv"
	"testing"
)

type fakePAPExchangeRateStore struct {
	listedRates []model.PAPTypeRate
	savedRates  []model.PAPTypeRate
	saveErr     error
	listErr     error
}

func (f *fakePAPExchangeRateStore) List() ([]model.PAPTypeRate, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return append([]model.PAPTypeRate(nil), f.listedRates...), nil
}

func (f *fakePAPExchangeRateStore) Save(rates []model.PAPTypeRate) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.savedRates = append([]model.PAPTypeRate(nil), rates...)
	f.listedRates = append([]model.PAPTypeRate(nil), rates...)
	return nil
}

type fakePAPExchangeConfigStore struct {
	fcSalary                    float64
	fcSalaryMonthlyLimit        int
	adminAward                  int
	multicharFullRewardCount    int
	multicharReducedRewardCount int
	multicharReducedRewardPct   int
	setManyCalls                int
	setManyItems                []repository.SysConfigUpsertItem
	setManyErr                  error
	hasSalary                   bool
	hasLimit                    bool
	hasAdminAward               bool
	hasMulticharFull            bool
	hasMulticharReduced         bool
	hasMulticharPct             bool
}

func (f *fakePAPExchangeConfigStore) GetFloat(key string, defaultVal float64) float64 {
	if key == model.SysConfigPAPFCSalary && f.hasSalary {
		return f.fcSalary
	}
	return defaultVal
}

func (f *fakePAPExchangeConfigStore) GetInt(key string, defaultVal int) int {
	switch key {
	case model.SysConfigPAPFCSalaryLimit:
		if f.hasLimit {
			return f.fcSalaryMonthlyLimit
		}
	case model.SysConfigPAPAdminAward:
		if f.hasAdminAward {
			return f.adminAward
		}
	case model.SysConfigMulticharFullRewardCount:
		if f.hasMulticharFull {
			return f.multicharFullRewardCount
		}
	case model.SysConfigMulticharReducedRewardCount:
		if f.hasMulticharReduced {
			return f.multicharReducedRewardCount
		}
	case model.SysConfigMulticharReducedRewardPct:
		if f.hasMulticharPct {
			return f.multicharReducedRewardPct
		}
	}
	return defaultVal
}

func (f *fakePAPExchangeConfigStore) SetMany(items []repository.SysConfigUpsertItem) error {
	if f.setManyErr != nil {
		return f.setManyErr
	}
	f.setManyCalls++
	f.setManyItems = append([]repository.SysConfigUpsertItem(nil), items...)
	for _, item := range items {
		switch item.Key {
		case model.SysConfigPAPFCSalary:
			value, err := strconv.ParseFloat(item.Value, 64)
			if err != nil {
				return err
			}
			f.fcSalary = value
			f.hasSalary = true
		case model.SysConfigPAPFCSalaryLimit:
			value, err := strconv.Atoi(item.Value)
			if err != nil {
				return err
			}
			f.fcSalaryMonthlyLimit = value
			f.hasLimit = true
		case model.SysConfigPAPAdminAward:
			value, err := strconv.Atoi(item.Value)
			if err != nil {
				return err
			}
			f.adminAward = value
			f.hasAdminAward = true
		case model.SysConfigMulticharFullRewardCount:
			value, err := strconv.Atoi(item.Value)
			if err != nil {
				return err
			}
			f.multicharFullRewardCount = value
			f.hasMulticharFull = true
		case model.SysConfigMulticharReducedRewardCount:
			value, err := strconv.Atoi(item.Value)
			if err != nil {
				return err
			}
			f.multicharReducedRewardCount = value
			f.hasMulticharReduced = true
		case model.SysConfigMulticharReducedRewardPct:
			value, err := strconv.Atoi(item.Value)
			if err != nil {
				return err
			}
			f.multicharReducedRewardPct = value
			f.hasMulticharPct = true
		}
	}
	return nil
}

func TestPAPExchangeUpdateConfigPersistsSingleBatch(t *testing.T) {
	rateStore := &fakePAPExchangeRateStore{}
	configStore := &fakePAPExchangeConfigStore{}
	svc := &PAPExchangeService{rateRepo: rateStore, configRepo: configStore}
	fcSalary := 5.5
	fcSalaryMonthlyLimit := 3
	adminAward := 12
	multicharFull := 4
	multicharReduced := 2
	multicharPct := 75

	updated, err := svc.UpdateConfig(&UpdateConfigRequest{
		Rates:                       []SetRateRequest{{PapType: "cta", DisplayName: "CTA", Rate: 1.5}},
		FCSalary:                    &fcSalary,
		FCSalaryMonthlyLimit:        &fcSalaryMonthlyLimit,
		AdminAward:                  &adminAward,
		MulticharFullRewardCount:    &multicharFull,
		MulticharReducedRewardCount: &multicharReduced,
		MulticharReducedRewardPct:   &multicharPct,
	})
	if err != nil {
		t.Fatalf("expected update to succeed, got %v", err)
	}
	if configStore.setManyCalls != 1 {
		t.Fatalf("expected exactly one batch write, got %d", configStore.setManyCalls)
	}
	if len(configStore.setManyItems) != 6 {
		t.Fatalf("expected 6 config items, got %d", len(configStore.setManyItems))
	}
	if updated.FCSalary != fcSalary {
		t.Fatalf("expected fc salary %v, got %v", fcSalary, updated.FCSalary)
	}
	if updated.FCSalaryMonthlyLimit != fcSalaryMonthlyLimit {
		t.Fatalf("expected monthly limit %d, got %d", fcSalaryMonthlyLimit, updated.FCSalaryMonthlyLimit)
	}
	if updated.AdminAward != adminAward {
		t.Fatalf("expected admin award %d, got %d", adminAward, updated.AdminAward)
	}
	if updated.MulticharFullRewardCount != multicharFull {
		t.Fatalf("expected multichar full %d, got %d", multicharFull, updated.MulticharFullRewardCount)
	}
	if updated.MulticharReducedRewardCount != multicharReduced {
		t.Fatalf("expected multichar reduced %d, got %d", multicharReduced, updated.MulticharReducedRewardCount)
	}
	if updated.MulticharReducedRewardPct != multicharPct {
		t.Fatalf("expected multichar pct %d, got %d", multicharPct, updated.MulticharReducedRewardPct)
	}
	if len(updated.Rates) != 1 || updated.Rates[0].Rate != 1.5 {
		t.Fatalf("expected updated PAP rates to round-trip, got %+v", updated.Rates)
	}
}

func TestPAPExchangeGetConfigResolvesAdminAwardDefaultsAndZero(t *testing.T) {
	tests := []struct {
		name  string
		store *fakePAPExchangeConfigStore
		want  int
	}{
		{
			name:  "defaults to configured constant when unset",
			store: &fakePAPExchangeConfigStore{},
			want:  model.SysConfigDefaultPAPAdminAward,
		},
		{
			name: "preserves configured zero award",
			store: &fakePAPExchangeConfigStore{
				adminAward:    0,
				hasAdminAward: true,
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &PAPExchangeService{
				rateRepo:   &fakePAPExchangeRateStore{},
				configRepo: tt.store,
			}

			cfg, err := svc.GetConfig()
			if err != nil {
				t.Fatalf("GetConfig() error = %v", err)
			}
			if cfg.AdminAward != tt.want {
				t.Fatalf("admin award = %d, want %d", cfg.AdminAward, tt.want)
			}
		})
	}
}

func TestPAPExchangeUpdateConfigByOperatorWritesAuditEvent(t *testing.T) {
	db := newServiceTestDB(t, "pap_exchange_audit", &model.AuditEvent{})
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	rateStore := &fakePAPExchangeRateStore{}
	configStore := &fakePAPExchangeConfigStore{}
	svc := &PAPExchangeService{
		rateRepo:   rateStore,
		configRepo: configStore,
		auditSvc:   NewAuditService(),
	}

	fcSalary := 8.8
	fcSalaryMonthlyLimit := 2
	adminAward := 15
	multicharFull := 4
	multicharReduced := 1
	multicharPct := 60

	_, err := svc.UpdateConfigByOperator(&UpdateConfigRequest{
		Rates:                       []SetRateRequest{{PapType: "stratop", DisplayName: "StratOp", Rate: 2.0}},
		FCSalary:                    &fcSalary,
		FCSalaryMonthlyLimit:        &fcSalaryMonthlyLimit,
		AdminAward:                  &adminAward,
		MulticharFullRewardCount:    &multicharFull,
		MulticharReducedRewardCount: &multicharReduced,
		MulticharReducedRewardPct:   &multicharPct,
	}, 88)
	if err != nil {
		t.Fatalf("UpdateConfigByOperator() error = %v", err)
	}

	var events []model.AuditEvent
	if err := db.Where("resource_type = ? AND action = ?", "system_config", "pap_exchange_config_update").Find(&events).Error; err != nil {
		t.Fatalf("load audit events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected pap_exchange_config_update audit event")
	}
	if events[0].Category != "config" || events[0].ActorUserID != 88 || events[0].Result != model.AuditResultSuccess {
		t.Fatalf("unexpected audit event: %+v", events[0])
	}
}
