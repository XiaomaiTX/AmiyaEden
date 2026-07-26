package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	qqGovernanceLease                = 30 * time.Second
	qqGovernanceMaxRetries           = 5
	qqGovernanceMaxCardRunes         = 60
	qqGovernanceGlobalBucketCapacity = 3
	qqGovernanceGlobalRefillInterval = 3 * time.Second
	qqGovernanceGroupActionInterval  = 8 * time.Second
	qqGovernanceKickGroupInterval    = 30 * time.Second
	oneBotDisconnectedMessage        = "OneBot 机器人未连接"
)

type OneBotActionExecutor interface {
	CallOneBot(context.Context, string, map[string]any) (json.RawMessage, error)
	OneBotConnected() bool
}

type OneBotActionError struct {
	Message   string
	Retryable bool
}

func (e *OneBotActionError) Error() string { return e.Message }

type QQGovernanceInboundEvent struct {
	EventKey, EventType, RequestFlag string
	GroupID, QQ                      int64
}
type qqGovernanceActionPayload struct {
	RequestFlag string `json:"request_flag"`
	Approve     bool   `json:"approve"`
	Card        string `json:"card"`
	RunID       uint   `json:"run_id"`
	Batch       int    `json:"batch"`
	Message     string `json:"message"`
}
type qqEligibility struct {
	Decision, Reason, Nickname, CharacterName, CorporationName, TargetCard string
	UserID                                                                 uint
	Roles                                                                  []string
}

type QQGovernanceService struct {
	repo       *repository.QQGovernanceRepository
	mu         sync.RWMutex
	executor   OneBotActionExecutor
	workerOnce sync.Once
	now        func() time.Time
}

var defaultQQGovernance struct {
	sync.Once
	service *QQGovernanceService
}

func DefaultQQGovernanceService() *QQGovernanceService {
	defaultQQGovernance.Do(func() { defaultQQGovernance.service = NewQQGovernanceService() })
	return defaultQQGovernance.service
}
func NewQQGovernanceService() *QQGovernanceService {
	return NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepository())
}
func NewQQGovernanceServiceWithRepository(repo *repository.QQGovernanceRepository) *QQGovernanceService {
	return &QQGovernanceService{repo: repo, now: time.Now}
}
func (s *QQGovernanceService) SetOneBotActionExecutor(executor OneBotActionExecutor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executor = executor
}
func (s *QQGovernanceService) actionExecutor() OneBotActionExecutor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.executor
}

// HandleOneBotEvent 将事件去重、审查、成员状态和后续动作任务一起提交。
func (s *QQGovernanceService) HandleOneBotEvent(ctx context.Context, event QQGovernanceInboundEvent) error {
	if strings.TrimSpace(event.EventKey) == "" || event.GroupID <= 0 || event.QQ <= 0 {
		return errors.New("OneBot 群治理事件无效")
	}
	return s.repo.Transaction(func(tx *gorm.DB) error {
		stored := &model.QQGovernanceEvent{EventKey: event.EventKey, EventType: event.EventType, GroupID: event.GroupID, QQ: event.QQ, RequestFlag: event.RequestFlag}
		created, err := s.repo.CreateEventIfNewTx(tx, stored)
		if err != nil {
			return err
		}
		if !created {
			return nil
		}
		policy, err := s.repo.GetEnabledPolicyTx(tx, event.GroupID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if event.EventType == "notice/group_decrease" {
			return s.recordDeparture(tx, stored, policy, event)
		}
		eligibility, err := s.evaluate(tx, policy, event.QQ)
		if err != nil {
			return err
		}
		return s.recordDecision(tx, stored, policy, event, eligibility)
	})
}

func (s *QQGovernanceService) evaluate(tx *gorm.DB, policy *model.QQGroupGovernancePolicy, qq int64) (qqEligibility, error) {
	result := qqEligibility{Decision: model.QQGovernanceReviewUnmatched}
	var user model.User
	if err := tx.Where("qq = ?", fmt.Sprintf("%d", qq)).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result.Reason = "QQ 未绑定 Seat 用户"
			return result, nil
		}
		return result, err
	}
	result.UserID, result.Nickname = user.ID, user.Nickname
	if user.PrimaryCharacterID <= 0 {
		result.Reason = "Seat 用户未设置主人物"
		return result, nil
	}
	var character model.EveCharacter
	if err := tx.Where("character_id = ? AND user_id = ?", user.PrimaryCharacterID, user.ID).First(&character).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result.Reason = "Seat 主人物不存在"
			return result, nil
		}
		return result, err
	}
	if character.CorporationID <= 0 {
		result.Reason = "Seat 主人物缺少军团信息"
		return result, nil
	}
	result.CharacterName, result.CorporationName = character.CharacterName, fmt.Sprintf("Corporation-%d", character.CorporationID)
	if err := tx.Model(&model.UserRole{}).Where("user_id = ?", user.ID).Pluck("role_code", &result.Roles).Error; err != nil {
		return result, err
	}
	result.Roles = model.NormalizeRoleCodes(result.Roles, user.Role)
	corps, err := parseQQGovernanceCorps(policy.AllowedCorporationIDsJSON)
	if err != nil {
		result.Reason = "群治理军团规则无效"
		return result, nil
	}
	roles, err := parseQQGovernanceRoles(policy.AllowedRoleCodesJSON)
	if err != nil {
		result.Reason = "群治理职权规则无效"
		return result, nil
	}
	if len(corps) == 0 && len(roles) == 0 {
		result.Decision = model.QQGovernanceReviewUnmatched
		result.Reason = "群治理规则为空，禁止自动放行"
		return result, nil
	}
	if len(corps) > 0 && !containsQQGovernanceCorp(corps, character.CorporationID) {
		result.Decision = model.QQGovernanceReviewUnmatched
		result.Reason = "主人物军团不在群准入范围"
		return result, nil
	}
	if len(roles) > 0 && !model.ContainsAnyRole(result.Roles, roles...) {
		result.Decision = model.QQGovernanceReviewUnmatched
		result.Reason = "Seat 用户职权不满足群准入规则"
		return result, nil
	}
	result.Decision, result.Reason = model.QQGovernanceReviewMatched, "主人物军团和职权满足群准入规则"
	card, err := renderQQGovernanceCard(policy.CardTemplate, result)
	if err != nil {
		result.Reason += "；名片未创建：" + err.Error()
	} else {
		result.TargetCard = card
	}
	return result, nil
}

func (s *QQGovernanceService) recordDecision(tx *gorm.DB, event *model.QQGovernanceEvent, policy *model.QQGroupGovernancePolicy, inbound QQGovernanceInboundEvent, decision qqEligibility) error {
	now := s.now()
	state, err := s.repo.GetMemberStateTx(tx, inbound.GroupID, inbound.QQ)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		state = &model.QQGroupMemberState{GroupID: inbound.GroupID, QQ: inbound.QQ, Version: 1}
	} else if err != nil {
		return err
	} else {
		state.Version++
	}
	state.UserID, state.TargetCard, state.LastCheckedAt = decision.UserID, decision.TargetCard, now
	if inbound.EventType == "request/group_add" {
		state.Status = model.QQGovernanceMemberJoinPending
	} else if decision.Decision == model.QQGovernanceReviewMatched {
		state.Status = model.QQGovernanceMemberValid
	} else if decision.Decision == model.QQGovernanceReviewUnmatched {
		state.Status = model.QQGovernanceMemberInvalidCand
	} else {
		// 仅保留给依赖读取失败等暂态异常的自动重试；它不是人工审核队列。
		state.Status = model.QQGovernanceMemberReview
	}
	if err := s.repo.SaveMemberStateTx(tx, state); err != nil {
		return err
	}
	roles, _ := json.Marshal(decision.Roles)
	if err := s.repo.CreateReviewTx(tx, &model.QQGovernanceReview{EventID: event.ID, PolicyID: policy.ID, GroupID: inbound.GroupID, QQ: inbound.QQ, Source: inbound.EventType, Decision: decision.Decision, Reason: decision.Reason, UserID: decision.UserID, Nickname: decision.Nickname, PrimaryCharacterName: decision.CharacterName, PrimaryCorporationName: decision.CorporationName, RoleCodesJSON: string(roles)}); err != nil {
		return err
	}
	if inbound.EventType == "request/group_add" && decision.Decision == model.QQGovernanceReviewMatched {
		return s.enqueueDecision(tx, state, inbound, model.QQGovernanceActionApprove, true, now)
	}
	if inbound.EventType == "request/group_add" && decision.Decision == model.QQGovernanceReviewUnmatched && policy.AutoRejectUnmatched {
		return s.enqueueDecision(tx, state, inbound, model.QQGovernanceActionReject, false, now)
	}
	if inbound.EventType == "notice/group_increase" && decision.Decision == model.QQGovernanceReviewMatched && decision.TargetCard != "" && policy.CardSyncEnabled {
		return s.enqueueCard(tx, state, inbound, now)
	}
	return nil
}
func (s *QQGovernanceService) recordDeparture(tx *gorm.DB, event *model.QQGovernanceEvent, policy *model.QQGroupGovernancePolicy, inbound QQGovernanceInboundEvent) error {
	state, err := s.repo.GetMemberStateTx(tx, inbound.GroupID, inbound.QQ)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		state = &model.QQGroupMemberState{GroupID: inbound.GroupID, QQ: inbound.QQ, Version: 1}
	} else if err != nil {
		return err
	} else {
		state.Version++
	}
	state.Status, state.TargetCard, state.LastCheckedAt = model.QQGovernanceMemberLeft, "", s.now()
	if err = s.repo.SaveMemberStateTx(tx, state); err != nil {
		return err
	}
	return s.repo.CreateReviewTx(tx, &model.QQGovernanceReview{EventID: event.ID, PolicyID: policy.ID, GroupID: inbound.GroupID, QQ: inbound.QQ, Source: inbound.EventType, Decision: model.QQGovernanceReviewWait, Reason: "成员已离群", UserID: state.UserID})
}
func (s *QQGovernanceService) enqueueDecision(tx *gorm.DB, state *model.QQGroupMemberState, event QQGovernanceInboundEvent, kind string, approve bool, now time.Time) error {
	payload, _ := json.Marshal(qqGovernanceActionPayload{RequestFlag: event.RequestFlag, Approve: approve})
	_, err := s.repo.CreateActionTaskIfAbsentTx(tx, &model.QQGovernanceActionTask{ActionType: kind, IdempotencyKey: fmt.Sprintf("%s:%d:%d:%s", kind, event.GroupID, event.QQ, event.RequestFlag), GroupID: event.GroupID, QQ: event.QQ, TargetVersion: state.Version, PayloadJSON: string(payload), Status: model.QQGovernanceActionPending, Priority: 10, RunAfter: now.Add(2*time.Second + time.Duration(rand.IntN(5))*time.Second)})
	return err
}
func (s *QQGovernanceService) enqueueCard(tx *gorm.DB, state *model.QQGroupMemberState, event QQGovernanceInboundEvent, now time.Time) error {
	payload, _ := json.Marshal(qqGovernanceActionPayload{Card: state.TargetCard})
	_, err := s.repo.CreateActionTaskIfAbsentTx(tx, &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionSetCard, IdempotencyKey: fmt.Sprintf("card:%d:%d:%d", event.GroupID, event.QQ, state.Version), GroupID: event.GroupID, QQ: event.QQ, TargetVersion: state.Version, PayloadJSON: string(payload), Status: model.QQGovernanceActionPending, Priority: 30, RunAfter: now.Add(15*time.Second + time.Duration(rand.IntN(46))*time.Second)})
	return err
}

// EnqueueGroupNotifications 为 qq_governance_onebot webhook 入口创建多群通知任务。
// 多群入队使用同一事务，避免部分成功造成部分发送。每个群使用独立幂等键以允许重试。
func (s *QQGovernanceService) EnqueueGroupNotifications(groupIDs []int64, content string) error {
	return s.enqueueGroupNotifications(groupIDs, content, "webhook")
}

// EnqueueStructureAlertNotifications puts corporation-structure notices onto
// the existing OneBot worker queue. The worker remains responsible for WSS
// connectivity, rate limits, retries and dead-letter handling.
func (s *QQGovernanceService) EnqueueStructureAlertNotifications(groupIDs []int64, content string) error {
	return s.enqueueGroupNotifications(groupIDs, content, "structure_alert")
}

func (s *QQGovernanceService) enqueueGroupNotifications(groupIDs []int64, content, source string) error {
	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		return errors.New("QQ 群治理通知内容不能为空")
	}
	ids, err := normalizeQQGovernanceGroupIDs(groupIDs)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return errors.New("QQ 群治理通知目标群号不能为空")
	}
	payload, err := json.Marshal(qqGovernanceActionPayload{Message: trimmedContent})
	if err != nil {
		return err
	}
	now := s.now()
	return s.repo.Transaction(func(tx *gorm.DB) error {
		for _, groupID := range ids {
			token, err := newQQGovernanceLeaseToken()
			if err != nil {
				return err
			}
			task := &model.QQGovernanceActionTask{
				ActionType:     model.QQGovernanceActionNotify,
				IdempotencyKey: fmt.Sprintf("%s-notify:%d:%s", source, groupID, token),
				GroupID:        groupID,
				QQ:             0,
				PayloadJSON:    string(payload),
				Status:         model.QQGovernanceActionPending,
				Priority:       20,
				RunAfter:       now.Add(2*time.Second + time.Duration(rand.IntN(5))*time.Second),
				Source:         source,
			}
			if _, err := s.repo.CreateActionTaskIfAbsentTx(tx, task); err != nil {
				return err
			}
		}
		return nil
	})
}

func parseQQGovernanceCorps(raw string) ([]int64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var v []int64
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, err
	}
	for _, id := range v {
		if id <= 0 {
			return nil, errors.New("军团 ID 必须为正整数")
		}
	}
	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	return v, nil
}
func parseQQGovernanceRoles(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var v []string
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, err
	}
	for _, role := range v {
		if !model.IsValidRoleCode(role) {
			return nil, fmt.Errorf("未知职权编码: %s", role)
		}
	}
	return v, nil
}
func containsQQGovernanceCorp(values []int64, target int64) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

var qqCardPlaceholder = regexp.MustCompile(`\{([^{}]+)\}`)

func renderQQGovernanceCard(template string, value qqEligibility) (string, error) {
	template = strings.TrimSpace(template)
	if template == "" {
		return "", nil
	}
	values := map[string]string{"nickname": value.Nickname, "primary_character_name": value.CharacterName, "primary_corporation_name": value.CorporationName}
	for _, m := range qqCardPlaceholder.FindAllStringSubmatch(template, -1) {
		if _, ok := values[m[1]]; !ok {
			return "", fmt.Errorf("名片模板包含不支持的占位符 %q", m[1])
		}
	}
	for k, v := range values {
		template = strings.ReplaceAll(template, "{"+k+"}", v)
	}
	template = strings.TrimSpace(template)
	if template == "" {
		return "", errors.New("名片渲染结果为空")
	}
	if len([]rune(template)) > qqGovernanceMaxCardRunes {
		return "", fmt.Errorf("名片超过 QQ 平台 %d 字符限制", qqGovernanceMaxCardRunes)
	}
	return template, nil
}

func newQQGovernanceLeaseToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := crand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
func (s *QQGovernanceService) logger() *zap.Logger {
	if l := global.CurrentLogger(); l != nil {
		return l
	}
	return zap.NewNop()
}
