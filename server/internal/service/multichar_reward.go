package service

import "amiya-eden/internal/model"

// MulticharRewardConfig holds the three configurable tier thresholds.
type MulticharRewardConfig struct {
	FullRewardCount    int // characters that get 100%
	ReducedRewardCount int // characters that get reduced %
	ReducedRewardPct   int // percentage for the reduced tier (0-100)
}

// CharacterTierMultiplier returns the reward multiplier (0.0–1.0)
// for a character at the given 1-based position among a user's
// characters in an event.
func CharacterTierMultiplier(position, fullCount, reducedCount, reducedPercent int) float64 {
	if position <= 0 {
		return 0
	}
	if position <= fullCount {
		return 1.0
	}
	if position <= fullCount+reducedCount {
		return float64(reducedPercent) / 100.0
	}
	return 0
}

// getMulticharRewardConfig reads tier thresholds from the config store
// with fallback to defaults.
type multicharConfigReader interface {
	GetInt(key string, defaultVal int) int
}

func getMulticharRewardConfig(cfg multicharConfigReader) MulticharRewardConfig {
	return MulticharRewardConfig{
		FullRewardCount:    cfg.GetInt(model.SysConfigMulticharFullRewardCount, model.SysConfigDefaultMulticharFullRewardCount),
		ReducedRewardCount: cfg.GetInt(model.SysConfigMulticharReducedRewardCount, model.SysConfigDefaultMulticharReducedRewardCount),
		ReducedRewardPct:   cfg.GetInt(model.SysConfigMulticharReducedRewardPct, model.SysConfigDefaultMulticharReducedRewardPct),
	}
}
