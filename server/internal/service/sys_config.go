package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// MumblePublicNode 是一个对用户展示的 Mumble 服务器节点。Name 与 Description
// 仅用于展示，Address+Port 才是用户实际连接的入口。
type MumblePublicNode struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
}

const maxMumblePublicNodes = 10

// MumbleRuntimeConfig is managed in Seat system settings. ServiceToken is used
// for go-mumble-server -> Seat calls; RevalidateToken is used in the reverse
// direction and must be a different credential. PublicNodes are display-only
// connection hints for end users and are unrelated to ServerURL.
type MumbleRuntimeConfig struct {
	ServiceToken        string             `json:"service_token"`
	ServerURL           string             `json:"server_url"`
	RevalidateToken     string             `json:"revalidate_token"`
	RevalidateTimeoutMS int                `json:"revalidate_timeout_ms"`
	PublicNodes         []MumblePublicNode `json:"public_nodes"`
	DisplayNameTemplate string             `json:"display_name_template"`
}

const defaultMumbleDisplayNameTemplate = "{character_name}"

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

// loadMumblePublicNodes 读取对用户展示的 Mumble 服务器节点列表；未配置或损坏时返回空列表。
func loadMumblePublicNodes(repo *repository.SysConfigRepository) []MumblePublicNode {
	nodes := []MumblePublicNode{}
	if repo == nil {
		return nodes
	}
	raw := strings.TrimSpace(repo.GetString(model.SysConfigMumblePublicNodes, ""))
	if raw == "" {
		return nodes
	}
	if err := json.Unmarshal([]byte(raw), &nodes); err != nil || nodes == nil {
		return []MumblePublicNode{}
	}
	return nodes
}

func (s *SysConfigService) GetMumbleConfig() MumbleRuntimeConfig {
	return MumbleRuntimeConfig{
		ServiceToken:        s.repo.GetString(model.SysConfigMumbleServiceToken, ""),
		ServerURL:           s.repo.GetString(model.SysConfigMumbleServerURL, ""),
		RevalidateToken:     s.repo.GetString(model.SysConfigMumbleRevalidateToken, ""),
		RevalidateTimeoutMS: s.repo.GetInt(model.SysConfigMumbleRevalidateTimeoutMS, model.SysConfigDefaultMumbleRevalidateTimeoutMS),
		PublicNodes:         loadMumblePublicNodes(s.repo),
		DisplayNameTemplate: s.repo.GetString(model.SysConfigMumbleDisplayNameTemplate, defaultMumbleDisplayNameTemplate),
	}
}

// normalizeMumblePublicNodes 校验并规范化对用户展示的 Mumble 服务器节点列表。
func normalizeMumblePublicNodes(nodes []MumblePublicNode) ([]MumblePublicNode, error) {
	if len(nodes) > maxMumblePublicNodes {
		return nil, fmt.Errorf("mumble 服务器节点最多 %d 个", maxMumblePublicNodes)
	}
	normalized := make([]MumblePublicNode, 0, len(nodes))
	seen := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		node.Name = strings.TrimSpace(node.Name)
		node.Description = strings.TrimSpace(node.Description)
		node.Address = strings.TrimSpace(node.Address)
		if len(node.Name) > 64 {
			return nil, errors.New("mumble 服务器节点名称最多 64 个字节")
		}
		if len(node.Description) > 256 {
			return nil, errors.New("mumble 服务器节点描述最多 256 个字节")
		}
		if node.Address == "" || len(node.Address) > 253 || strings.ContainsAny(node.Address, "/ \t\r\n") {
			return nil, errors.New("mumble 服务器连接地址格式无效")
		}
		if node.Port < 0 || node.Port > 65535 {
			return nil, errors.New("mumble 服务器连接端口必须在 0 到 65535 之间")
		}
		key := fmt.Sprintf("%s:%d", node.Address, node.Port)
		if _, dup := seen[key]; dup {
			return nil, errors.New("mumble 服务器节点地址与端口不能重复")
		}
		seen[key] = struct{}{}
		normalized = append(normalized, node)
	}
	return normalized, nil
}

func (s *SysConfigService) UpdateMumbleConfig(cfg MumbleRuntimeConfig) error {
	cfg.ServiceToken = strings.TrimSpace(cfg.ServiceToken)
	cfg.ServerURL = strings.TrimRight(strings.TrimSpace(cfg.ServerURL), "/")
	cfg.RevalidateToken = strings.TrimSpace(cfg.RevalidateToken)
	cfg.DisplayNameTemplate = strings.TrimSpace(cfg.DisplayNameTemplate)
	if cfg.DisplayNameTemplate == "" {
		cfg.DisplayNameTemplate = defaultMumbleDisplayNameTemplate
	}
	if cfg.ServerURL != "" {
		parsed, err := url.Parse(cfg.ServerURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return errors.New("mumble 服务地址无效")
		}
	}
	if cfg.ServiceToken != "" && len(cfg.ServiceToken) < 16 {
		return errors.New("mumble 调用 Seat 的服务令牌至少需要 16 个字符")
	}
	if cfg.RevalidateToken != "" && len(cfg.RevalidateToken) < 16 {
		return errors.New("seat 调用 Mumble 的重校验令牌至少需要 16 个字符")
	}
	if cfg.ServiceToken != "" && cfg.ServiceToken == cfg.RevalidateToken {
		return errors.New("两个方向必须使用不同的服务令牌")
	}
	if cfg.RevalidateTimeoutMS < 100 || cfg.RevalidateTimeoutMS > 10000 {
		return errors.New("mumble 重校验超时必须在 100 到 10000 毫秒之间")
	}
	nodes, err := normalizeMumblePublicNodes(cfg.PublicNodes)
	if err != nil {
		return err
	}
	rawNodes, err := json.Marshal(nodes)
	if err != nil {
		return errors.New("编码 Mumble 服务器节点列表失败")
	}
	if err := validateMumbleDisplayNameTemplate(cfg.DisplayNameTemplate); err != nil {
		return err
	}
	items := newSysConfigBatch(6).
		AddString(model.SysConfigMumbleServiceToken, cfg.ServiceToken, "Mumble 调用 Seat 的服务令牌").
		AddString(model.SysConfigMumbleServerURL, cfg.ServerURL, "Mumble 服务管理地址").
		AddString(model.SysConfigMumbleRevalidateToken, cfg.RevalidateToken, "Seat 调用 Mumble 的重校验令牌").
		AddInt(model.SysConfigMumbleRevalidateTimeoutMS, cfg.RevalidateTimeoutMS, "Mumble 重校验请求超时（毫秒）").
		AddString(model.SysConfigMumblePublicNodes, string(rawNodes), "对用户展示的 Mumble 服务器节点列表").
		AddString(model.SysConfigMumbleDisplayNameTemplate, cfg.DisplayNameTemplate, "Mumble 昵称显示模板").
		Items()
	if err := s.repo.SetMany(items); err != nil {
		return errors.New("更新 Mumble 连接设置失败")
	}
	return nil
}

// legacyMumblePublicKeys 是 v1 单节点展示配置的 key，仅在启动迁移中引用。
var legacyMumblePublicKeys = []string{"mumble.public_address", "mumble.public_port"}

// MigrateLegacyMumblePublicNodes 在启动时把废弃的单节点配置折叠进 mumble.public_nodes，
// 并删除旧 key；幂等，可安全重复执行。
func (s *SysConfigService) MigrateLegacyMumblePublicNodes() {
	if _, exists := s.repo.GetIfExists(model.SysConfigMumblePublicNodes); !exists {
		addrRaw, hasAddr := s.repo.GetIfExists(legacyMumblePublicKeys[0])
		addr := strings.TrimSpace(addrRaw)
		if hasAddr && addr != "" {
			port := 0
			if portRaw, hasPort := s.repo.GetIfExists(legacyMumblePublicKeys[1]); hasPort {
				if v, err := strconv.Atoi(strings.TrimSpace(portRaw)); err == nil {
					port = v
				}
			}
			if raw, err := json.Marshal([]MumblePublicNode{{Address: addr, Port: port}}); err == nil {
				_ = s.repo.Set(model.SysConfigMumblePublicNodes, string(raw), "对用户展示的 Mumble 服务器节点列表")
			}
		}
	}
	_ = s.repo.DeleteMany(legacyMumblePublicKeys)
}

func validateMumbleDisplayNameTemplate(template string) error {
	if len(template) > 128 {
		return errors.New("mumble 昵称显示模板最多 128 个字节")
	}
	remaining := strings.ReplaceAll(template, "{nickname}", "")
	remaining = strings.ReplaceAll(remaining, "{character_name}", "")
	remaining = strings.ReplaceAll(remaining, "{corporation_ticker}", "")
	remaining = strings.ReplaceAll(remaining, "{alliance_ticker}", "")
	remaining = strings.ReplaceAll(remaining, "{roles}", "")
	if strings.Contains(remaining, "{") || strings.Contains(remaining, "}") {
		return errors.New("mumble 昵称显示模板只支持 {alliance_ticker}、{corporation_ticker}、{nickname}、{character_name} 和 {roles}")
	}
	return nil
}

func (s *SysConfigService) GetQQGovernanceSettings() QQGovernanceSettings {
	return QQGovernanceSettings{
		ScanIntervalMinutes:      s.repo.GetInt(model.SysConfigQQGovernanceScanIntervalMinutes, model.SysConfigDefaultQQGovernanceScanIntervalMinutes),
		MismatchConfirmations:    s.repo.GetInt(model.SysConfigQQGovernanceMismatchConfirmations, model.SysConfigDefaultQQGovernanceMismatchConfirmations),
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

// SearchCorporations resolves a corporation through ESI's public
// POST /universe/ids/ endpoint using an exact name match, so governance rules
// persist stable corporation IDs instead of a free-form name. The endpoint is
// unauthenticated and only resolves exact names.
func (s *SysConfigService) SearchCorporations(ctx context.Context, query string) ([]CorporationDisplay, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []CorporationDisplay{}, nil
	}
	client := esi.NewClient()
	if global.Config != nil {
		client = esi.NewClientWithConfig(global.Config.EveSSO.ESIBaseURL, global.Config.EveSSO.ESIAPIPrefix)
	}
	var payload struct {
		Corporations []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"corporations"`
	}
	if err := client.PostJSON(ctx, "/universe/ids/?datasource=tranquility", "", []string{query}, &payload); err != nil {
		return nil, err
	}
	items := make([]CorporationDisplay, 0, len(payload.Corporations))
	for _, corp := range payload.Corporations {
		if corp.ID <= 0 || strings.TrimSpace(corp.Name) == "" {
			continue
		}
		items = append(items, CorporationDisplay{CorporationID: corp.ID, CorporationName: strings.TrimSpace(corp.Name)})
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].CorporationName) < strings.ToLower(items[j].CorporationName)
	})
	if s.entityNameResolver != nil {
		s.entityNameResolver.PrimeCorporationNames(items)
	}
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

// resolveCorporationNamesAllowMissing resolves as many corporation names as the
// cache + ESI can provide and silently treats unresolved IDs as empty names.
// Used on read paths where partial resolution must not break the listing.
func (s *SysConfigService) resolveCorporationNamesAllowMissing(ctx context.Context, corporationIDs []int64) map[int64]string {
	if len(corporationIDs) == 0 {
		return map[int64]string{}
	}
	if s.entityNameResolver == nil {
		s.entityNameResolver = NewEntityNameResolver()
	}
	resolved := s.entityNameResolver.Resolve(ctx, corporationIDs)
	return resolved.Names
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
