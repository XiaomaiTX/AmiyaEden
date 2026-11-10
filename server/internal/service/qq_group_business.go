package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"
)

var errQQBusinessRecheck = errors.New("business dependency needs recheck")

type QQGroupBusinessService struct {
	repo     *repository.QQGroupBusinessRepository
	dispatch *QQBotService
}

func NewQQGroupBusinessService(db *gorm.DB, d *QQBotService) *QQGroupBusinessService {
	return &QQGroupBusinessService{repository.NewQQGroupBusinessRepository(db), d}
}
func (s *QQGroupBusinessService) Configure(ctx context.Context, v model.QQGroupRule) error {
	if v.MismatchConfirmations < 1 || v.MismatchConfirmations > 100 || v.ObservationHours < 0 || v.ObservationHours > 8760 {
		return errors.New("invalid confirmation policy")
	}
	if e := validateQQCardTemplate(v.CardTemplate); e != nil {
		return e
	}
	if v.CardEnabled && v.CardTemplate == "" {
		return errors.New("card template required")
	}
	if v.GroupID <= 0 || len(v.CorporationIDs) > 100 || len(v.Roles) > 50 {
		return errors.New("invalid rule")
	}
	for _, id := range v.CorporationIDs {
		if id <= 0 {
			return errors.New("invalid corporation")
		}
	}
	for _, role := range v.Roles {
		if !model.IsValidRoleCode(role) {
			return errors.New("invalid role")
		}
	}
	slices.Sort(v.CorporationIDs)
	slices.Sort(v.Roles)
	if len(v.CorporationIDs) == 0 {
		v.CorporationIDs = nil
	}
	if len(v.Roles) == 0 {
		v.Roles = nil
	}
	s.dispatch.stepMu.Lock()
	defer s.dispatch.stepMu.Unlock()
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		old, e := s.repo.Rule(tx, v.GroupID)
		if e != nil {
			return e
		}
		requested, previous := v, old
		requested.Version = 0
		previous.Version = 0
		requested.LastInspectionAt = time.Time{}
		previous.LastInspectionAt = time.Time{}
		requested.UpdatedAt = time.Time{}
		previous.UpdatedAt = time.Time{}
		if reflect.DeepEqual(requested, previous) {
			return nil
		}
		if e = s.repo.CancelSnapshots(tx, v.GroupID); e != nil {
			return e
		}
		v.Version = old.Version + 1
		v.LastInspectionAt = old.LastInspectionAt
		if e = s.repo.CancelKicks(tx, v.GroupID); e != nil {
			return e
		}
		if e = s.repo.ResetConfirmations(tx, v.GroupID); e != nil {
			return e
		}
		if e = s.repo.SaveRule(tx, &v); e != nil {
			return e
		}
		if !v.CardEnabled || old.CardTemplate != v.CardTemplate {
			if e = s.repo.CancelCards(tx, v.GroupID); e != nil {
				return e
			}
		}
		return s.repo.CancelGroup(tx, v.GroupID)
	})
}
func (s *QQGroupBusinessService) qualify(tx *gorm.DB, rule model.QQGroupRule, qq int64) (string, string, error) {
	d, e := s.repo.Qualification(tx, qq)
	if e != nil {
		return "unknown", "dependency_error", e
	}
	if len(d.Users) != 1 || d.Character.CharacterID <= 0 || d.Character.CorporationID <= 0 {
		return "unknown", "profile_missing_or_ambiguous", nil
	}
	if len(rule.CorporationIDs) == 0 && len(rule.Roles) == 0 {
		return "unknown", "empty_rule", nil
	}
	corp := len(rule.CorporationIDs) == 0
	for _, id := range rule.CorporationIDs {
		if id == d.Character.CorporationID {
			corp = true
		}
	}
	role := len(rule.Roles) == 0
	for _, required := range rule.Roles {
		for _, actual := range d.Roles {
			if actual.RoleCode == required {
				role = true
			}
		}
	}
	if d.Users[0].Status != 1 || !corp || !role {
		return "mismatch", fmt.Sprintf("user:%d character:%d corporation:%d corp_match:%t role_match:%t", d.Users[0].ID, d.Character.CharacterID, d.Character.CorporationID, corp, role), nil
	}
	return "eligible", fmt.Sprintf("user:%d character:%d corporation:%d", d.Users[0].ID, d.Character.CharacterID, d.Character.CorporationID), nil
}
func (s *QQGroupBusinessService) ConsumeEvent(ctx context.Context, tx *gorm.DB, payload string) error {
	var event struct {
		NoticeType  string `json:"notice_type"`
		PostType    string `json:"post_type"`
		RequestType string `json:"request_type"`
		SubType     string `json:"sub_type"`
		GroupID     int64  `json:"group_id"`
		UserID      int64  `json:"user_id"`
		Flag        string `json:"flag"`
		Time        int64  `json:"time"`
	}
	if e := json.Unmarshal([]byte(payload), &event); e != nil {
		return e
	}
	if event.PostType == "notice" && (event.NoticeType == "group_increase" || event.NoticeType == "group_decrease") {
		if event.GroupID <= 0 || event.UserID <= 0 || event.Time <= 0 {
			return errors.New("invalid membership event")
		}
		eventAt := time.Unix(event.Time, 0)
		fresh, e := s.repo.AppliedEvent(tx, qqbotHash([]byte(fmt.Sprintf("%d:%d:%s:%d", event.GroupID, event.UserID, event.NoticeType, event.Time))))
		if e != nil || !fresh {
			return e
		}
		rule, e := s.repo.Rule(tx, event.GroupID)
		if e != nil {
			return e
		}
		member, e := s.repo.Member(tx, event.GroupID, event.UserID)
		if e != nil {
			return e
		}
		if eventAt.Before(member.EventAt) {
			return nil
		}
		wasPresent := member.Present
		member.Present = event.NoticeType == "group_increase"
		if member.Present && !wasPresent {
			member.CardTarget = ""
		}
		member.EventAt = eventAt
		member.Epoch++
		member.MembershipReceivedAt = time.Now()
		if e = s.resetChain(tx, &member); e != nil {
			return e
		}
		if !member.Present {
			if e = s.repo.CancelOperation(tx, member.CardOperationID); e != nil {
				return e
			}
		} else {
			if e = s.planCard(ctx, tx, rule, &member); e != nil {
				return e
			}
		}
		return s.repo.SaveMember(tx, &member)
	}
	if event.PostType != "request" || event.RequestType != "group" {
		return nil
	}
	if event.GroupID <= 0 || event.UserID <= 0 || event.Flag == "" || (event.SubType != "add" && event.SubType != "invite") {
		return errors.New("invalid group application")
	}
	now := time.Now()
	expires := now.Add(24 * time.Hour)
	if event.Time > 0 && time.Unix(event.Time, 0).Add(24*time.Hour).Before(expires) {
		expires = time.Unix(event.Time, 0).Add(24 * time.Hour)
	}
	app := model.QQGroupApplication{ID: qqbotHash([]byte(fmt.Sprintf("%d:%s:%s", event.GroupID, event.SubType, event.Flag))), GroupID: event.GroupID, QQ: event.UserID, Flag: event.Flag, SubType: event.SubType, Status: "waiting", ExpiresAt: expires, RecheckAt: now}
	if e := s.repo.Application(tx, &app); e != nil {
		return e
	}
	return s.evaluateApplication(ctx, tx, &app, now)
}
func (s *QQGroupBusinessService) evaluateApplication(ctx context.Context, tx *gorm.DB, app *model.QQGroupApplication, now time.Time) error {
	if app.Status != "waiting" {
		return nil
	}
	if !app.ExpiresAt.After(now) {
		app.Status = "expired"
		return s.repo.SaveApplication(tx, app)
	}
	rule, e := s.repo.Rule(tx, app.GroupID)
	if e != nil {
		return e
	}
	if !rule.AdmissionEnabled {
		app.Evidence = "admission_disabled"
		app.RecheckAt = now.Add(5 * time.Minute)
		return s.repo.SaveApplication(tx, app)
	}
	qualification, evidence, e := s.qualify(tx, rule, app.QQ)
	if e != nil {
		return e
	}
	app.Evidence = evidence
	app.RecheckAt = now.Add(5 * time.Minute)
	if !rule.AdmissionEnabled || qualification == "unknown" {
		return s.repo.SaveApplication(tx, app)
	}
	decision := "approve"
	if qualification == "mismatch" {
		if !rule.AutoReject {
			return s.repo.SaveApplication(tx, app)
		}
		decision = "reject"
	}
	if app.Decision != "" && app.Decision != decision {
		app.Status = "invalidated"
		return s.repo.SaveApplication(tx, app)
	}
	op, e := s.dispatch.Submit(ctx, tx, "application:"+app.ID, 0, app.ExpiresAt, model.QQBotRequest{Kind: "group_request", GroupID: app.GroupID, UserID: app.QQ, Flag: app.Flag, SubType: app.SubType, Approve: decision == "approve"})
	if e != nil {
		return e
	}
	app.Decision = decision
	app.RuleVersion = rule.Version
	app.OperationID = op.ID
	app.Status = "action_pending"
	return s.repo.SaveApplication(tx, app)
}
func (s *QQGroupBusinessService) valid(tx *gorm.DB, op model.QQBotOperation) (bool, error) {
	if op.Kind == "group_kick" {
		return s.validKick(tx, op)
	}
	if op.Kind == "group_card" {
		return s.validCard(tx, op)
	}
	if op.Kind != "group_request" {
		return true, nil
	}
	app, e := s.repo.ApplicationForOperation(tx, op.ID)
	if e != nil {
		return false, e
	}
	rule, e := s.repo.Rule(tx, app.GroupID)
	if e != nil {
		return false, e
	}
	if app.Status != "action_pending" || !app.ExpiresAt.After(time.Now()) || !rule.AdmissionEnabled || rule.Version != app.RuleVersion {
		return false, nil
	}
	q, _, e := s.qualify(tx, rule, app.QQ)
	if e != nil {
		return false, e
	}
	if q == "unknown" {
		return false, errQQBusinessRecheck
	}
	return (app.Decision == "approve" && q == "eligible") || (app.Decision == "reject" && rule.AutoReject && q == "mismatch"), nil
}
func (s *QQGroupBusinessService) ConsumeResult(_ context.Context, tx *gorm.DB, payload string) error {
	var op model.QQBotOperation
	if e := json.Unmarshal([]byte(payload), &op); e != nil {
		return e
	}
	if op.Kind == "member_list" {
		return s.consumeSnapshot(tx, op)
	}
	if op.Kind == "group_kick" {
		m, e := s.repo.KickMember(tx, op.ID)
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return nil
		}
		if e != nil {
			return e
		}
		m.KickStatus = op.Status
		if op.Status == "succeeded" {
			m.Present = false
		}
		return s.repo.SaveMember(tx, &m)
	}
	if op.Kind == "group_card" {
		member, e := s.repo.CardMember(tx, op.ID)
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return nil
		}
		if e != nil {
			return e
		}
		member.CardStatus = op.Status
		return s.repo.SaveMember(tx, &member)
	}
	if op.Kind != "group_request" {
		return nil
	}
	app, e := s.repo.ApplicationForOperation(tx, op.ID)
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil
	}
	if e != nil {
		return e
	}
	if app.Status != "action_pending" {
		return nil
	}
	app.Status = op.Status
	if op.Status == "succeeded" {
		app.Status = "approved"
		if app.Decision == "reject" {
			app.Status = "rejected"
		}
	}
	return s.repo.SaveApplication(tx, &app)
}
func (s *QQGroupBusinessService) Recheck(ctx context.Context, now time.Time) error {
	ids, e := s.repo.RecheckIDs(ctx, now)
	if e != nil {
		return e
	}
	for _, id := range ids {
		if e = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
			app, e := s.repo.GetApplication(tx, id)
			if e != nil {
				return e
			}
			return s.evaluateApplication(ctx, tx, &app, now)
		}); e != nil {
			return e
		}
	}
	return s.Inspect(ctx, now)
}
func (s *QQGroupBusinessService) Rules(ctx context.Context) ([]model.QQGroupRule, error) {
	return s.repo.Rules(ctx)
}
func (s *QQGroupBusinessService) Applications(ctx context.Context) ([]model.QQGroupApplication, error) {
	return s.repo.Applications(ctx)
}

func (s *QQBotService) Groups() *QQGroupBusinessService { return s.groups }

var qqCardVariable = regexp.MustCompile(`\{[^{}]+\}`)

func validateQQCardTemplate(template string) error {
	if len(template) > 180 {
		return errors.New("template too long")
	}
	for _, v := range qqCardVariable.FindAllString(template, -1) {
		switch v {
		case "{character_name}", "{corporation_name}", "{nickname}", "{qq}":
		default:
			return errors.New("unsupported card variable")
		}
	}
	return nil
}
func (s *QQGroupBusinessService) planCard(ctx context.Context, tx *gorm.DB, rule model.QQGroupRule, m *model.QQGroupMember) error {
	if !rule.CardEnabled || !m.Present {
		return nil
	}
	q, evidence, e := s.qualify(tx, rule, m.QQ)
	if e != nil {
		return e
	}
	m.Evidence = evidence
	if q != "eligible" {
		return nil
	}
	d, e := s.repo.Qualification(tx, m.QQ)
	if e != nil {
		return e
	}
	corp := ""
	if strings.Contains(rule.CardTemplate, "{corporation_name}") {
		corp, e = s.repo.CorporationName(tx, d.Character.CorporationID)
		if e != nil {
			return e
		}
		if corp == "" {
			m.Evidence = "corporation_name_missing"
			return nil
		}
	}
	target := strings.NewReplacer("{character_name}", d.Character.CharacterName, "{corporation_name}", corp, "{nickname}", d.Users[0].Nickname, "{qq}", fmt.Sprintf("%d", m.QQ)).Replace(rule.CardTemplate)
	if target == "" || len(target) > 180 {
		m.Evidence = "card_target_invalid"
		return nil
	}
	if m.CardTarget == target {
		return nil
	}
	if e = s.repo.CancelOperation(tx, m.CardOperationID); e != nil {
		return e
	}
	m.CardVersion++
	m.CardTarget = target
	op, e := s.dispatch.Submit(ctx, tx, fmt.Sprintf("card:%d:%d:%d", m.GroupID, m.QQ, m.CardVersion), 1, time.Now().Add(24*time.Hour), model.QQBotRequest{Kind: "group_card", GroupID: m.GroupID, UserID: m.QQ, Text: target})
	if e != nil {
		return e
	}
	m.CardOperationID = op.ID
	m.CardStatus = "pending"
	return nil
}
func (s *QQGroupBusinessService) validCard(tx *gorm.DB, op model.QQBotOperation) (bool, error) {
	var request model.QQBotRequest
	if e := json.Unmarshal([]byte(op.Request), &request); e != nil {
		return false, e
	}
	rule, e := s.repo.Rule(tx, request.GroupID)
	if e != nil {
		return false, e
	}
	m, e := s.repo.CardMember(tx, op.ID)
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if !rule.CardEnabled || !m.Present {
		return false, nil
	}
	if e = s.planCard(context.Background(), tx, rule, &m); e != nil {
		return false, e
	}
	if e = s.repo.SaveMember(tx, &m); e != nil {
		return false, e
	}
	if m.CardOperationID != op.ID {
		return false, nil
	}
	var req model.QQBotRequest
	if e = json.Unmarshal([]byte(op.Request), &req); e != nil {
		return false, e
	}
	if req.Text != m.CardTarget {
		return false, nil
	}
	q, _, e := s.qualify(tx, rule, m.QQ)
	if e != nil {
		return false, e
	}
	if q == "unknown" {
		return false, errQQBusinessRecheck
	}
	if q != "eligible" {
		return false, nil
	}
	if time.Since(m.CardLastAttemptAt) < time.Minute {
		return false, errQQBusinessRecheck
	}
	m.CardLastAttemptAt = time.Now()
	return true, s.repo.SaveMember(tx, &m)
}
func (s *QQGroupBusinessService) Members(ctx context.Context) ([]model.QQGroupMember, error) {
	return s.repo.Members(ctx)
}

type QQGroupBusinessAudit struct {
	Rules        []model.QQGroupRule           `json:"rules"`
	Applications []model.QQGroupApplication    `json:"applications"`
	Members      []model.QQGroupMember         `json:"members"`
	Inspections  []model.QQGroupInspection     `json:"inspections"`
	Judgements   []model.QQGroupInspectionItem `json:"judgements"`
}

func (s *QQGroupBusinessService) Audit(ctx context.Context) (QQGroupBusinessAudit, error) {
	var v QQGroupBusinessAudit
	var e error
	if v.Rules, e = s.Rules(ctx); e != nil {
		return v, e
	}
	if v.Applications, e = s.Applications(ctx); e != nil {
		return v, e
	}
	if v.Members, e = s.Members(ctx); e != nil {
		return v, e
	}
	if v.Inspections, e = s.repo.Inspections(ctx); e != nil {
		return v, e
	}
	v.Judgements, e = s.repo.Judgements(ctx)
	return v, e
}
