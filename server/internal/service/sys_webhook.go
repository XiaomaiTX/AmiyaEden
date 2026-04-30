package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultFleetTemplate = "@all 舰队行动通知\n行动: {title}\n指挥官: {fc_name}\n重要程度: {importance}\nPAP: {pap_count}\n时间: {start_at} ~ {end_at}\n{description}"

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	URL           string `json:"url"`
	Enabled       bool   `json:"enabled"`
	Type          string `json:"type"`           // discord | feishu | dingtalk | onebot
	FleetTemplate string `json:"fleet_template"` // 舰队行动通知模板
	OBTargetType  string `json:"ob_target_type"` // group | private
	OBTargetID    int64  `json:"ob_target_id"`   // 目标群号或用户 QQ
	OBToken       string `json:"ob_token"`       // access token（可空）
}

// WebhookService Webhook 业务逻辑层
type WebhookService struct {
	repo            webhookConfigStore
	fleetConfigRepo *repository.FleetConfigRepository
	http            *http.Client
	auditSvc        *AuditService
}

type webhookConfigStore interface {
	Get(key, defaultVal string) (string, error)
	GetBool(key string, defaultVal bool) bool
	SetMany(items []repository.SysConfigUpsertItem) error
}

func NewWebhookService() *WebhookService {
	return &WebhookService{
		repo:            repository.NewSysConfigRepository(),
		fleetConfigRepo: repository.NewFleetConfigRepository(),
		http:            &http.Client{Timeout: 10 * time.Second},
		auditSvc:        NewAuditService(),
	}
}

// GetConfig 获取 Webhook 配置
func (s *WebhookService) GetConfig() (*WebhookConfig, error) {
	url, err := s.repo.Get(model.SysConfigWebhookURL, "")
	if err != nil {
		return nil, fmt.Errorf("read webhook URL config: %w", err)
	}
	enabled := s.repo.GetBool(model.SysConfigWebhookEnabled, false)
	wtype, _ := s.repo.Get(model.SysConfigWebhookType, "discord")
	tmpl, _ := s.repo.Get(model.SysConfigWebhookFleetTemplate, defaultFleetTemplate)
	obTargetType, _ := s.repo.Get(model.SysConfigWebhookOBTargetType, "group")
	obTargetIDStr, _ := s.repo.Get(model.SysConfigWebhookOBTargetID, "0")
	obTargetID, _ := strconv.ParseInt(obTargetIDStr, 10, 64)
	obToken, _ := s.repo.Get(model.SysConfigWebhookOBToken, "")
	return &WebhookConfig{
		URL:           url,
		Enabled:       enabled,
		Type:          wtype,
		FleetTemplate: tmpl,
		OBTargetType:  obTargetType,
		OBTargetID:    obTargetID,
		OBToken:       obToken,
	}, nil
}

// SetConfig 保存 Webhook 配置
func (s *WebhookService) SetConfig(cfg *WebhookConfig) error {
	if err := validateWebhookRequestTarget(cfg); err != nil {
		return err
	}

	items := newSysConfigBatch(7).
		AddString(model.SysConfigWebhookURL, cfg.URL, "Webhook URL").
		AddBool(model.SysConfigWebhookEnabled, cfg.Enabled, "Webhook 是否启用").
		AddString(model.SysConfigWebhookType, cfg.Type, "Webhook 类型 (discord/feishu/dingtalk/onebot)").
		AddString(model.SysConfigWebhookFleetTemplate, cfg.FleetTemplate, "舰队行动通知模板").
		AddString(model.SysConfigWebhookOBTargetType, cfg.OBTargetType, "OneBot 目标类型 (group/private)").
		AddInt64(model.SysConfigWebhookOBTargetID, cfg.OBTargetID, "OneBot 目标 ID").
		AddString(model.SysConfigWebhookOBToken, cfg.OBToken, "OneBot Access Token").
		Items()
	return s.repo.SetMany(items)
}

func (s *WebhookService) SetConfigByOperator(cfg *WebhookConfig, operatorID uint) error {
	if err := s.SetConfig(cfg); err != nil {
		return err
	}
	if s.auditSvc != nil {
		_ = s.auditSvc.RecordEvent(context.Background(), AuditRecordInput{
			Category:     "config",
			Action:       "webhook_config_update",
			ActorUserID:  operatorID,
			ResourceType: "system_config",
			ResourceID:   model.SysConfigWebhookURL,
			Result:       model.AuditResultSuccess,
			Details: map[string]any{
				"enabled":        cfg.Enabled,
				"type":           cfg.Type,
				"url":            cfg.URL,
				"ob_target_type": cfg.OBTargetType,
				"ob_target_id":   cfg.OBTargetID,
			},
		})
	}
	return nil
}

// SendFleetPing 发送舰队行动 Ping（若未启用则静默忽略）
func (s *WebhookService) SendFleetPing(fleet *model.Fleet) error {
	cfg, err := s.GetConfig()
	if err != nil || !cfg.Enabled || cfg.URL == "" {
		return nil
	}

	importanceLabel := map[string]string{
		model.FleetImportanceStratOp: "战略行动",
		model.FleetImportanceCTA:     "全面集结",
		model.FleetImportanceOther:   "其他行动",
	}[fleet.Importance]
	if importanceLabel == "" {
		importanceLabel = fleet.Importance
	}

	desc := fleet.Description
	if desc == "" {
		desc = "-"
	}

	content := cfg.FleetTemplate
	content = strings.ReplaceAll(content, "{title}", fleet.Title)
	content = strings.ReplaceAll(content, "{fc_name}", fleet.FCCharacterName)
	content = strings.ReplaceAll(content, "{importance}", importanceLabel)
	content = strings.ReplaceAll(content, "{pap_count}", fmt.Sprintf("%.0f", fleet.PapCount))
	content = strings.ReplaceAll(content, "{start_at}", fleet.StartAt.Local().Format("01/02 15:04"))
	content = strings.ReplaceAll(content, "{end_at}", fleet.EndAt.Local().Format("01/02 15:04"))
	content = strings.ReplaceAll(content, "{description}", desc)

	// 舰队配置信息
	fleetConfigInfo := ""
	if fleet.FleetConfigID != nil && *fleet.FleetConfigID > 0 {
		if fc, fcErr := s.fleetConfigRepo.GetByID(*fleet.FleetConfigID); fcErr == nil {
			fittings, _ := s.fleetConfigRepo.ListFittingsByConfigID(fc.ID)
			fleetConfigInfo = fc.Name
			if len(fittings) > 0 {
				var names []string
				for _, f := range fittings {
					names = append(names, f.FittingName)
				}
				fleetConfigInfo += "\n  " + strings.Join(names, "\n  ")
			}
		}
	}
	content = strings.ReplaceAll(content, "{fleet_config}", fleetConfigInfo)

	return s.sendMessage(cfg, content)
}

// SendTest 发送测试消息
func (s *WebhookService) SendTest(cfg *WebhookConfig, content string) error {
	if cfg.Type == "" {
		cfg.Type = "discord"
	}
	if err := validateWebhookRequestTarget(cfg); err != nil {
		return err
	}
	if content == "" {
		content = "✅ Webhook 测试消息（来自 AmiyaEden）"
	}
	return s.sendMessage(cfg, content)
}

func (s *WebhookService) sendMessage(cfg *WebhookConfig, content string) error {
	reqURL, err := buildValidatedWebhookRequestURL(cfg)
	if err != nil {
		return err
	}

	var body []byte

	switch cfg.Type {
	case "feishu":
		body, err = json.Marshal(map[string]any{
			"msg_type": "text",
			"content":  map[string]string{"text": content},
		})
	case "dingtalk":
		body, err = json.Marshal(map[string]any{
			"msgtype": "text",
			"text":    map[string]string{"content": content},
		})
	case "onebot":
		endpoint := "/send_group_msg"
		var postBody map[string]any
		if cfg.OBTargetType == "private" {
			endpoint = "/send_private_msg"
			postBody = map[string]any{
				"user_id": cfg.OBTargetID,
				"message": content,
			}
		} else {
			postBody = map[string]any{
				"group_id": cfg.OBTargetID,
				"message":  content,
			}
		}
		reqURL += endpoint
		body, err = json.Marshal(postBody)
	default: // discord
		body, err = json.Marshal(map[string]string{"content": content})
	}
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "AmiyaEden/1.0")
	if cfg.Type == "onebot" && cfg.OBToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.OBToken)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook 返回错误状态码: %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func validateWebhookRequestTarget(cfg *WebhookConfig) error {
	_, err := buildValidatedWebhookRequestURL(cfg)
	return err
}

func buildValidatedWebhookRequestURL(cfg *WebhookConfig) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("webhook 配置错误: 配置不能为空")
	}

	webhookType := strings.ToLower(strings.TrimSpace(cfg.Type))
	if webhookType == "" {
		webhookType = "discord"
	}

	rawURL := strings.TrimSpace(cfg.URL)
	if rawURL == "" {
		return "", fmt.Errorf("webhook 配置错误: URL 不能为空")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("webhook 配置错误: URL 格式无效")
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("webhook 配置错误: URL 必须包含协议和主机")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("webhook 配置错误: URL 不支持包含用户信息")
	}
	if parsed.Fragment != "" {
		return "", fmt.Errorf("webhook 配置错误: URL 不支持包含片段")
	}

	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	switch webhookType {
	case "discord":
		if scheme != "https" {
			return "", fmt.Errorf("webhook 配置错误: discord 仅支持 https")
		}
		if !hostMatchesAllowList(host, []string{"discord.com", "discordapp.com"}) {
			return "", fmt.Errorf("webhook 配置错误: discord URL 域名不在允许范围内")
		}
	case "feishu":
		if scheme != "https" {
			return "", fmt.Errorf("webhook 配置错误: feishu 仅支持 https")
		}
		if !hostMatchesAllowList(host, []string{"feishu.cn", "larksuite.com"}) {
			return "", fmt.Errorf("webhook 配置错误: feishu URL 域名不在允许范围内")
		}
	case "dingtalk":
		if scheme != "https" {
			return "", fmt.Errorf("webhook 配置错误: dingtalk 仅支持 https")
		}
		if !hostMatchesAllowList(host, []string{"dingtalk.com", "dingtalkapps.com"}) {
			return "", fmt.Errorf("webhook 配置错误: dingtalk URL 域名不在允许范围内")
		}
	case "onebot":
		if scheme != "http" && scheme != "https" {
			return "", fmt.Errorf("webhook 配置错误: onebot 仅支持 http 或 https")
		}
		if cfg.OBTargetType != "" && cfg.OBTargetType != "group" && cfg.OBTargetType != "private" {
			return "", fmt.Errorf("webhook 配置错误: onebot 目标类型仅支持 group 或 private")
		}
	default:
		return "", fmt.Errorf("webhook 配置错误: 不支持的类型 %q", webhookType)
	}

	return strings.TrimRight(parsed.String(), "/"), nil
}

func hostMatchesAllowList(host string, allowList []string) bool {
	for _, allowed := range allowList {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		if allowed == "" {
			continue
		}
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return false
}
