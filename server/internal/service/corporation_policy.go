package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
)

const (
	corpPolicyDefaultModeDeny           = "deny"
	CorpRuleSRPRecommendationMultiplier = "srp.recommendation_multiplier"
)

var (
	ErrInvalidCorporationAccessPolicy = errors.New("军团能力策略配置无效")
)

type CorporationPolicyConfig struct {
	Version     int                 `json:"version"`
	DefaultMode string              `json:"default_mode"`
	Policies    []CorporationPolicy `json:"policies"`
}

type CorporationPolicy struct {
	CorporationID int64          `json:"corporation_id"`
	FullAccess    bool           `json:"full_access"`
	Capabilities  []string       `json:"capabilities"`
	Rules         map[string]any `json:"rules"`
}

type UserCorpPolicyContext struct {
	PrimaryCorporationID int64
	Capabilities         []string
	Rules                map[string]any
	FullAccess           bool
}

type CorporationPolicyService struct {
	cfgRepo  *repository.SysConfigRepository
	userRepo *repository.UserRepository
	charRepo *repository.EveCharacterRepository
}

var (
	corpPolicyCacheMu sync.RWMutex
	corpPolicyCache   *CorporationPolicyConfig
)

func NewCorporationPolicyService() *CorporationPolicyService {
	return &CorporationPolicyService{
		cfgRepo:  repository.NewSysConfigRepository(),
		userRepo: repository.NewUserRepository(),
		charRepo: repository.NewEveCharacterRepository(),
	}
}

func (s *CorporationPolicyService) GetPolicies() CorporationPolicyConfig {
	if cfg := getCorpPolicyCache(); cfg != nil {
		return *cfg
	}
	cfg := s.loadPolicies()
	setCorpPolicyCache(cfg)
	return cfg
}

func (s *CorporationPolicyService) UpdatePolicies(raw CorporationPolicyConfig) error {
	normalized, err := normalizeCorpPolicyConfig(raw)
	if err != nil {
		return err
	}
	serialized, err := json.Marshal(normalized)
	if err != nil {
		return errors.New("序列化军团能力策略失败")
	}
	if err := s.cfgRepo.Set(
		model.SysConfigCorporationAccessPolicies,
		string(serialized),
		"军团能力策略配置",
	); err != nil {
		return errors.New("更新军团能力策略失败")
	}
	s.InvalidateCache()
	return nil
}

func (s *CorporationPolicyService) InvalidateCache() {
	clearCorpPolicyCache()
}

func (s *CorporationPolicyService) BuildUserPolicyContext(userID uint) UserCorpPolicyContext {
	ctx := UserCorpPolicyContext{
		PrimaryCorporationID: 0,
		Capabilities:         []string{},
		Rules:                map[string]any{},
		FullAccess:           false,
	}
	if userID == 0 {
		return ctx
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil || user.PrimaryCharacterID == 0 {
		return ctx
	}
	primaryCharacter, err := s.charRepo.GetByCharacterID(user.PrimaryCharacterID)
	if err != nil || primaryCharacter == nil {
		return ctx
	}
	ctx.PrimaryCorporationID = primaryCharacter.CorporationID
	if ctx.PrimaryCorporationID <= 0 {
		return ctx
	}

	config := s.GetPolicies()
	policy, matched := matchCorporationPolicy(config, ctx.PrimaryCorporationID)
	if !matched {
		return ctx
	}
	ctx.FullAccess = policy.FullAccess
	ctx.Capabilities = slices.Clone(policy.Capabilities)
	if policy.Rules != nil {
		ctx.Rules = cloneAnyMap(policy.Rules)
	}
	return ctx
}

func (s *CorporationPolicyService) UserHasCapability(userID uint, roles []string, capability string) bool {
	if model.IsSuperAdmin(roles) {
		return true
	}
	ctx := s.BuildUserPolicyContext(userID)
	return EvaluateCapability(ctx, capability)
}

func (s *CorporationPolicyService) GetRuleFloat(corporationID int64, key string, defaultValue float64) float64 {
	if corporationID <= 0 || key == "" {
		return defaultValue
	}
	config := s.GetPolicies()
	policy, matched := matchCorporationPolicy(config, corporationID)
	if !matched || policy.Rules == nil {
		return defaultValue
	}
	raw, ok := policy.Rules[key]
	if !ok {
		return defaultValue
	}
	switch value := raw.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	case uint:
		return float64(value)
	case uint32:
		return float64(value)
	case uint64:
		return float64(value)
	default:
		return defaultValue
	}
}

func EvaluateCapability(ctx UserCorpPolicyContext, capability string) bool {
	if ctx.FullAccess {
		return true
	}
	for _, granted := range ctx.Capabilities {
		if granted == capability {
			return true
		}
	}
	return false
}

func normalizeCorpPolicyConfig(raw CorporationPolicyConfig) (CorporationPolicyConfig, error) {
	cfg := CorporationPolicyConfig{
		Version:     raw.Version,
		DefaultMode: raw.DefaultMode,
		Policies:    raw.Policies,
	}
	if cfg.Version <= 0 {
		cfg.Version = 1
	}
	if cfg.DefaultMode == "" {
		cfg.DefaultMode = corpPolicyDefaultModeDeny
	}
	// 当前版本固定最小权限，不开放 allow。
	if cfg.DefaultMode != corpPolicyDefaultModeDeny {
		return CorporationPolicyConfig{}, fmt.Errorf("%w: default_mode 仅允许 deny", ErrInvalidCorporationAccessPolicy)
	}
	normalizedPolicies := make([]CorporationPolicy, 0, len(cfg.Policies))
	seenCorpID := make(map[int64]struct{}, len(cfg.Policies))
	for _, policy := range cfg.Policies {
		if policy.CorporationID <= 0 {
			return CorporationPolicyConfig{}, fmt.Errorf("%w: corporation_id 必须为正整数", ErrInvalidCorporationAccessPolicy)
		}
		if _, exists := seenCorpID[policy.CorporationID]; exists {
			return CorporationPolicyConfig{}, fmt.Errorf("%w: corporation_id 不能重复", ErrInvalidCorporationAccessPolicy)
		}
		seenCorpID[policy.CorporationID] = struct{}{}

		capabilities := make([]string, 0, len(policy.Capabilities))
		seenCapabilities := make(map[string]struct{}, len(policy.Capabilities))
		for _, capability := range policy.Capabilities {
			if !model.IsValidCorpCapability(capability) {
				return CorporationPolicyConfig{}, fmt.Errorf("%w: 非法 capability: %s", ErrInvalidCorporationAccessPolicy, capability)
			}
			if _, exists := seenCapabilities[capability]; exists {
				continue
			}
			seenCapabilities[capability] = struct{}{}
			capabilities = append(capabilities, capability)
		}
		slices.Sort(capabilities)

		rules := map[string]any{}
		for key, value := range policy.Rules {
			switch value.(type) {
			case string, bool, float64, int, int32, int64, uint, uint32, uint64:
				rules[key] = value
			default:
				return CorporationPolicyConfig{}, fmt.Errorf("%w: 规则 %s 的值类型不支持", ErrInvalidCorporationAccessPolicy, key)
			}
		}

		normalizedPolicies = append(normalizedPolicies, CorporationPolicy{
			CorporationID: policy.CorporationID,
			FullAccess:    policy.FullAccess,
			Capabilities:  capabilities,
			Rules:         rules,
		})
	}
	slices.SortFunc(normalizedPolicies, func(a, b CorporationPolicy) int {
		switch {
		case a.CorporationID < b.CorporationID:
			return -1
		case a.CorporationID > b.CorporationID:
			return 1
		default:
			return 0
		}
	})
	cfg.Policies = normalizedPolicies
	return cfg, nil
}

func (s *CorporationPolicyService) loadPolicies() CorporationPolicyConfig {
	raw, err := s.cfgRepo.Get(model.SysConfigCorporationAccessPolicies, "")
	if err != nil || raw == "" {
		return defaultCorpPolicyConfig()
	}
	var parsed CorporationPolicyConfig
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return defaultCorpPolicyConfig()
	}
	normalized, err := normalizeCorpPolicyConfig(parsed)
	if err != nil {
		return defaultCorpPolicyConfig()
	}
	return normalized
}

func defaultCorpPolicyConfig() CorporationPolicyConfig {
	return CorporationPolicyConfig{
		Version:     1,
		DefaultMode: corpPolicyDefaultModeDeny,
		Policies:    []CorporationPolicy{},
	}
}

func matchCorporationPolicy(cfg CorporationPolicyConfig, corporationID int64) (CorporationPolicy, bool) {
	for _, policy := range cfg.Policies {
		if policy.CorporationID == corporationID {
			return policy, true
		}
	}
	return CorporationPolicy{}, false
}

func getCorpPolicyCache() *CorporationPolicyConfig {
	corpPolicyCacheMu.RLock()
	defer corpPolicyCacheMu.RUnlock()
	if corpPolicyCache == nil {
		return nil
	}
	cloned := *corpPolicyCache
	cloned.Policies = slices.Clone(corpPolicyCache.Policies)
	return &cloned
}

func setCorpPolicyCache(cfg CorporationPolicyConfig) {
	corpPolicyCacheMu.Lock()
	defer corpPolicyCacheMu.Unlock()
	cloned := cfg
	cloned.Policies = slices.Clone(cfg.Policies)
	corpPolicyCache = &cloned
}

func clearCorpPolicyCache() {
	corpPolicyCacheMu.Lock()
	defer corpPolicyCacheMu.Unlock()
	corpPolicyCache = nil
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for k, v := range input {
		cloned[k] = v
	}
	return cloned
}
