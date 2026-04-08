package service

import (
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
	multicharFullRewardCount    int
	multicharReducedRewardCount int
	multicharReducedRewardPct   int
	setManyCalls                int
	setManyItems                []repository.SysConfigUpsertItem
	setManyErr                  error
	hasSalary                   bool
	hasLimit                    bool
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
	multicharFull := 4
	multicharReduced := 2
	multicharPct := 75

	updated, err := svc.UpdateConfig(&UpdateConfigRequest{
		Rates:                       []SetRateRequest{{PapType: "cta", DisplayName: "CTA", Rate: 1.5}},
		FCSalary:                    &fcSalary,
		FCSalaryMonthlyLimit:        &fcSalaryMonthlyLimit,
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
	if len(configStore.setManyItems) != 5 {
		t.Fatalf("expected 5 config items, got %d", len(configStore.setManyItems))
	}
	if updated.FCSalary != fcSalary {
		t.Fatalf("expected fc salary %v, got %v", fcSalary, updated.FCSalary)
	}
	if updated.FCSalaryMonthlyLimit != fcSalaryMonthlyLimit {
		t.Fatalf("expected monthly limit %d, got %d", fcSalaryMonthlyLimit, updated.FCSalaryMonthlyLimit)
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
