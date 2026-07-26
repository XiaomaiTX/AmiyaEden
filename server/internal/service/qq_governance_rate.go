package service

import (
	"amiya-eden/global"
	"context"
	"math"
	"strconv"
	"time"
)

type QQGovernanceRateLimitBucket struct {
	Capacity float64 `json:"capacity"`
	Tokens   float64 `json:"tokens"`
	WaitMS   int64   `json:"wait_ms"`
}

type QQGovernanceGroupRateLimit struct {
	GroupID int64                       `json:"group_id"`
	Bucket  QQGovernanceRateLimitBucket `json:"bucket"`
}

type QQGovernanceRateLimitStatus struct {
	Available bool                         `json:"available"`
	Global    QQGovernanceRateLimitBucket  `json:"global"`
	Groups    []QQGovernanceGroupRateLimit `json:"groups"`
}

type QQGovernanceConnectionStatus struct {
	Connected bool                        `json:"connected"`
	RiskLevel int                         `json:"risk_level"`
	RateLimit QQGovernanceRateLimitStatus `json:"rate_limit"`
}

func (s *QQGovernanceService) ConnectionStatus() (QQGovernanceConnectionStatus, error) {
	result := QQGovernanceConnectionStatus{Connected: s.actionExecutor() != nil && s.actionExecutor().OneBotConnected()}
	if botQQ := NewSysConfigService().GetOneBotConfig().BotQQ; botQQ > 0 {
		if risk, err := s.repo.GetRiskState(botQQ); err == nil {
			result.RiskLevel = risk.Level
		}
	}
	result.RateLimit = s.rateLimitStatus()
	return result, nil
}

func (s *QQGovernanceService) rateLimitStatus() QQGovernanceRateLimitStatus {
	status := QQGovernanceRateLimitStatus{Global: QQGovernanceRateLimitBucket{Capacity: qqGovernanceGlobalBucketCapacity}}
	if global.Redis == nil {
		return status
	}
	policies, err := s.repo.ListPolicies(nil)
	if err != nil {
		return status
	}
	now := s.now().UnixMilli()
	globalValues, err := global.Redis.HGetAll(context.Background(), "qq_governance:rate:global").Result()
	if err != nil {
		return status
	}
	globalBucket, ok := qqGovernanceRateLimitBucket(globalValues, qqGovernanceGlobalBucketCapacity, qqGovernanceGlobalRefillInterval, now)
	if !ok {
		return status
	}
	status.Global = globalBucket
	groups := make([]QQGovernanceGroupRateLimit, 0, len(policies))
	for _, policy := range policies {
		values, getErr := global.Redis.HGetAll(context.Background(), "qq_governance:rate:group:"+strconv.FormatInt(policy.GroupID, 10)).Result()
		if getErr != nil {
			return QQGovernanceRateLimitStatus{Global: status.Global}
		}
		bucket, valid := qqGovernanceRateLimitBucket(values, 1, qqGovernanceGroupActionInterval, now)
		if !valid {
			return QQGovernanceRateLimitStatus{Global: status.Global}
		}
		groups = append(groups, QQGovernanceGroupRateLimit{GroupID: policy.GroupID, Bucket: bucket})
	}
	status.Available, status.Groups = true, groups
	return status
}

func qqGovernanceRateLimitBucket(values map[string]string, capacity float64, refillInterval time.Duration, nowMS int64) (QQGovernanceRateLimitBucket, bool) {
	tokens, last := capacity, float64(nowMS)
	if raw := values["tokens"]; raw != "" {
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return QQGovernanceRateLimitBucket{}, false
		}
		tokens = parsed
	}
	if raw := values["last"]; raw != "" {
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return QQGovernanceRateLimitBucket{}, false
		}
		last = parsed
	}
	refillPerMS := 1 / float64(refillInterval.Milliseconds())
	tokens = math.Min(capacity, tokens+math.Max(0, float64(nowMS)-last)*refillPerMS)
	waitMS := int64(0)
	if tokens < 1 {
		waitMS = int64(math.Ceil((1 - tokens) / refillPerMS))
	}
	return QQGovernanceRateLimitBucket{Capacity: capacity, Tokens: tokens, WaitMS: waitMS}, true
}
