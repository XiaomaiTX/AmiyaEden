package service

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/utils"
	"amiya-eden/pkg/eve/esi"

	"go.uber.org/zap"
)

var ErrInvalidAllowCorporations = errors.New("军团 ID 必须为正整数")

type SysConfigService struct {
	repo               *repository.SysConfigRepository
	entityNameResolver *EntityNameResolver
}

type SDEConfig struct {
	APIKey      string
	Proxy       string
	DownloadURL string
}

type AlliancePAPConfig struct {
	BaseURL string
	APIKey  string
}

// OneBotRuntimeConfig 是 QQ 群治理反向 WebSocket 的动态运行时配置。
type OneBotRuntimeConfig struct {
	Enabled      bool
	AccessToken  string
	BotQQ        int64
	AllowedCIDRs []string
}

// QQGovernanceSettings 是所有受治理 QQ 群共用的巡检与清退确认参数。
type QQGovernanceSettings struct {
	ScanIntervalMinutes      int `json:"scan_interval_minutes"`
	MismatchConfirmations    int `json:"mismatch_confirmations"`
	MismatchObservationHours int `json:"mismatch_observation_hours"`
}

type CorporationDisplay struct {
	CorporationID   int64  `json:"corporation_id"`
	CorporationName string `json:"corporation_name"`
}

func NewSysConfigService() *SysConfigService {
	return NewSysConfigServiceWithRepository(repository.NewSysConfigRepository())
}

func NewSysConfigServiceWithRepository(repo *repository.SysConfigRepository) *SysConfigService {
	return &SysConfigService{
		repo:               repo,
		entityNameResolver: NewEntityNameResolver(),
	}
}

func (s *SysConfigService) GetSDEConfig() SDEConfig {
	apiKey, _ := s.repo.Get(model.SysConfigSDEAPIKey, model.SysConfigDefaultSDEAPIKey)
	proxy, _ := s.repo.Get(model.SysConfigSDEProxy, model.SysConfigDefaultSDEProxy)
	downloadURL, _ := s.repo.Get(model.SysConfigSDEDownloadURL, model.SysConfigDefaultSDEDownloadURL)

	return SDEConfig{
		APIKey:      apiKey,
		Proxy:       proxy,
		DownloadURL: downloadURL,
	}
}

func (s *SysConfigService) UpdateSDEConfig(apiKey, proxy, downloadURL *string) error {
	items := newSysConfigBatch(3)
	if apiKey != nil {
		items.AddString(model.SysConfigSDEAPIKey, *apiKey, "SDE 查询 API Key")
	}
	if proxy != nil {
		items.AddString(model.SysConfigSDEProxy, *proxy, "SDE 下载代理")
	}
	if downloadURL != nil {
		items.AddString(model.SysConfigSDEDownloadURL, *downloadURL, "SDE 下载地址")
	}
	if err := s.repo.SetMany(items.Items()); err != nil {
		return errors.New("更新 SDE 配置失败")
	}
	return nil
}

func (s *SysConfigService) GetAlliancePAPConfig() AlliancePAPConfig {
	return AlliancePAPConfig{
		BaseURL: s.repo.GetString(model.SysConfigAlliancePAPBaseURL, model.SysConfigDefaultAlliancePAPBaseURL),
		APIKey:  s.repo.GetString(model.SysConfigAlliancePAPAPIKey, model.SysConfigDefaultAlliancePAPAPIKey),
	}
}

func (s *SysConfigService) UpdateAlliancePAPConfig(baseURL, apiKey *string) error {
	items := newSysConfigBatch(2)
	if baseURL != nil {
		items.AddString(model.SysConfigAlliancePAPBaseURL, *baseURL, "联盟 PAP API 地址")
	}
	if apiKey != nil {
		items.AddString(model.SysConfigAlliancePAPAPIKey, *apiKey, "联盟 PAP API Key")
	}
	if err := s.repo.SetMany(items.Items()); err != nil {
		return errors.New("更新联盟 PAP 配置失败")
	}
	return nil
}

func (s *SysConfigService) GetOneBotConfig() OneBotRuntimeConfig {
	defaultCIDRs := []string{"127.0.0.1/32"}
	raw := s.repo.GetString(model.SysConfigOneBotAllowedCIDRs, "")
	cidrs := defaultCIDRs
	if raw != "" && json.Unmarshal([]byte(raw), &cidrs) != nil {
		cidrs = defaultCIDRs
	}
	return OneBotRuntimeConfig{
		Enabled:      s.repo.GetBool(model.SysConfigOneBotEnabled, model.SysConfigDefaultOneBotEnabled),
		AccessToken:  s.repo.GetString(model.SysConfigOneBotAccessToken, model.SysConfigDefaultOneBotAccessToken),
		BotQQ:        s.repo.GetInt64(model.SysConfigOneBotBotQQ, model.SysConfigDefaultOneBotBotQQ),
		AllowedCIDRs: cidrs,
	}
}

func (s *SysConfigService) UpdateOneBotConfig(enabled *bool, accessToken *string, botQQ *int64, allowedCIDRs *[]string) error {
	items := newSysConfigBatch(4)
	if enabled != nil {
		items.AddBool(model.SysConfigOneBotEnabled, *enabled, "是否启用 QQ 群治理 OneBot")
	}
	if accessToken != nil {
		items.AddString(model.SysConfigOneBotAccessToken, *accessToken, "QQ 群治理 OneBot Access Token")
	}
	if botQQ != nil {
		items.AddInt64(model.SysConfigOneBotBotQQ, *botQQ, "QQ 群治理 OneBot 机器人 QQ")
	}
	if allowedCIDRs != nil {
		for _, cidr := range *allowedCIDRs {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				return errors.New("OneBot 受控网段格式无效")
			}
		}
		raw, err := json.Marshal(*allowedCIDRs)
		if err != nil {
			return errors.New("编码 OneBot 受控网段失败")
		}
		items.AddString(model.SysConfigOneBotAllowedCIDRs, string(raw), "QQ 群治理 OneBot 受控网段")
	}
	if err := s.repo.SetMany(items.Items()); err != nil {
		return errors.New("更新 OneBot 配置失败")
	}
	return nil
}

func (s *SysConfigService) GetQQGovernanceSettings() QQGovernanceSettings {
	return QQGovernanceSettings{
		ScanIntervalMinutes: s.repo.GetInt(model.SysConfigQQGovernanceScanIntervalMinutes, model.SysConfigDefaultQQGovernanceScanIntervalMinutes),
		MismatchConfirmations: s.repo.GetInt(model.SysConfigQQGovernanceMismatchConfirmations, model.SysConfigDefaultQQGovernanceMismatchConfirmations),
		MismatchObservationHours: s.repo.GetInt(model.SysConfigQQGovernanceMismatchObservationHours, model.SysConfigDefaultQQGovernanceMismatchObservationHours),
	}
}

func (s *SysConfigService) UpdateQQGovernanceSettings(input QQGovernanceSettings) error {
	if err := validateQQGovernanceSettings(input); err != nil {
		return err
	}
	items := newSysConfigBatch(3)
	items.AddInt(model.SysConfigQQGovernanceScanIntervalMinutes, input.ScanIntervalMinutes, "QQ 群治理全局扫描间隔（分钟）")
	items.AddInt(model.SysConfigQQGovernanceMismatchConfirmations, input.MismatchConfirmations, "QQ 群治理全局连续不匹配次数")
	items.AddInt(model.SysConfigQQGovernanceMismatchObservationHours, input.MismatchObservationHours, "QQ 群治理全局观察期（小时）")
	if err := s.repo.SetMany(items.Items()); err != nil {
		return errors.New("更新 QQ 群治理全局设置失败")
	}
	return nil
}

func validateQQGovernanceSettings(input QQGovernanceSettings) error {
	if input.ScanIntervalMinutes < 15 || input.ScanIntervalMinutes > 360 || input.ScanIntervalMinutes%15 != 0 {
		return errors.New("扫描间隔必须为 15 到 360 分钟且为 15 的倍数")
	}
	if input.MismatchConfirmations < 2 || input.MismatchConfirmations > 3 {
		return errors.New("连续不匹配次数必须为 2 到 3")
	}
	if input.MismatchObservationHours < 1 || input.MismatchObservationHours > 6 {
		return errors.New("观察期必须为 1 到 6 小时")
	}
	return nil
}

func (s *SysConfigService) GetAllowCorporations() []int64 {
	return utils.GetAllowCorporations()
}

func (s *SysConfigService) GetAllowCorporationDisplays(ctx context.Context) []CorporationDisplay {
	allowCorporations := s.GetAllowCorporations()
	if len(allowCorporations) == 0 {
		return []CorporationDisplay{}
	}

	nameMap, err := s.resolveCorporationNames(ctx, allowCorporations)
	if err != nil {
		if global.Logger != nil {
			global.Logger.Warn("[SysConfig] resolve allow corporations names failed", zap.Error(err))
		}
	}

	displays := make([]CorporationDisplay, 0, len(allowCorporations))
	for _, corporationID := range allowCorporations {
		displays = append(displays, CorporationDisplay{
			CorporationID:   corporationID,
			CorporationName: nameMap[corporationID],
		})
	}
	return displays
}

// SearchCorporations resolves corporation names through ESI so governance rules
// persist stable corporation IDs instead of a free-form name.
func (s *SysConfigService) SearchCorporations(ctx context.Context, query string) ([]CorporationDisplay, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []CorporationDisplay{}, nil
	}
	client := esi.NewClient()
	if global.Config != nil {
		client = esi.NewClientWithConfig(global.Config.EveSSO.ESIBaseURL, global.Config.EveSSO.ESIAPIPrefix)
	}
	var result struct {
		CorporationIDs []int64 `json:"corporation"`
	}
	path := "/search/?categories=corporation&strict=false&search=" + url.QueryEscape(query)
	if err := client.Get(ctx, path, "", &result); err != nil {
		return nil, err
	}
	resolved := s.entityNameResolver.Resolve(ctx, result.CorporationIDs)
	items := make([]CorporationDisplay, 0, len(result.CorporationIDs))
	for _, corporationID := range result.CorporationIDs {
		if name := resolved.Names[corporationID]; name != "" {
			items = append(items, CorporationDisplay{CorporationID: corporationID, CorporationName: name})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].CorporationName) < strings.ToLower(items[j].CorporationName)
	})
	return items, nil
}

func (s *SysConfigService) UpdateAllowCorporations(allowCorporations []int64) error {
	if err := utils.ValidateAllowCorporations(allowCorporations); err != nil {
		return ErrInvalidAllowCorporations
	}
	normalizedAllowCorporations := utils.NormalizeAllowCorporations(allowCorporations)
	if err := s.repo.SetInt64Slice(model.SysConfigAllowCorporations, normalizedAllowCorporations, "允许访问的公司 ID 列表"); err != nil {
		return errors.New("更新允许的军团列表失败")
	}
	utils.InvalidateAllowCorporationsCache()
	return nil
}

func (s *SysConfigService) resolveCorporationNames(ctx context.Context, corporationIDs []int64) (map[int64]string, error) {
	nameMap := make(map[int64]string, len(corporationIDs))
	if len(corporationIDs) == 0 {
		return nameMap, nil
	}

	if s.entityNameResolver == nil {
		s.entityNameResolver = NewEntityNameResolver()
	}
	resolved := s.entityNameResolver.Resolve(ctx, corporationIDs)
	for id, name := range resolved.Names {
		nameMap[id] = name
	}
	if len(resolved.Miss) > 0 {
		return nameMap, errors.New("some corporation names are unresolved")
	}
	return nameMap, nil
}

func (s *SysConfigService) GetCharacterESIRestrictionConfig() bool {
	return s.repo.GetBool(
		model.SysConfigEnforceCharacterESIRestriction,
		model.SysConfigDefaultEnforceCharacterESIRestriction,
	)
}

func (s *SysConfigService) UpdateCharacterESIRestrictionConfig(enforce bool) error {
	if err := s.repo.Set(
		model.SysConfigEnforceCharacterESIRestriction,
		strconv.FormatBool(enforce),
		"是否强制限制失效人物 ESI 停留在人物页面",
	); err != nil {
		return errors.New("更新人物 ESI 限制配置失败")
	}
	return nil
}
