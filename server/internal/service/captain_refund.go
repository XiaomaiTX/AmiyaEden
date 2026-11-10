package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"math"
	"slices"
	"strings"
	"time"
)

const refundSettingsKey = "newbro.refund_settings"

type CaptainRefundSettings struct {
	MaxTotalSP       int64   `json:"max_total_sp"`
	MaxAgeDays       int     `json:"max_age_days"`
	TaxRate          float64 `json:"tax_rate"`
	AllowedSystemIDs []int64 `json:"allowed_system_ids"`
}
type CaptainRefundRequest struct {
	CaptainCharacterID int64     `json:"captain_character_id"`
	MemberNames        []string  `json:"member_names"`
	SystemID           int64     `json:"system_id"`
	Start              time.Time `json:"start"`
	End                time.Time `json:"end"`
}
type RefundMember struct {
	CharacterID     int64                   `json:"character_id"`
	CharacterName   string                  `json:"character_name"`
	UserID          uint                    `json:"user_id"`
	IsCaptain       bool                    `json:"is_captain"`
	Eligibility     NewbroEligibilityResult `json:"eligibility"`
	Bounty          float64                 `json:"bounty"`
	WalletUpdatedAt *time.Time              `json:"wallet_updated_at"`
}
type RefundJournal struct {
	CharacterID    int64      `json:"character_id"`
	JournalID      int64      `json:"journal_id"`
	Date           time.Time  `json:"date"`
	Amount         float64    `json:"amount"`
	MatchJournalID int64      `json:"match_journal_id"`
	MatchDate      *time.Time `json:"match_date"`
}
type RefundConflict struct {
	CharacterID int64     `json:"character_id"`
	JournalID   int64     `json:"journal_id"`
	JournalAt   time.Time `json:"journal_at"`
	RefundID    uint      `json:"refund_id"`
}
type CaptainRefundCalculation struct {
	Request      CaptainRefundRequest  `json:"request"`
	Settings     CaptainRefundSettings `json:"settings"`
	Members      []RefundMember        `json:"members"`
	IgnoredNames []string              `json:"ignored_names"`
	Journals     []RefundJournal       `json:"journals"`
	Conflicts    []RefundConflict      `json:"conflicts"`
	Bounty       float64               `json:"bounty"`
	Reward       float64               `json:"reward"`
	HasNewbro    bool                  `json:"has_newbro"`
	CanSubmit    bool                  `json:"can_submit"`
}
type CaptainRefundView struct {
	model.CaptainRefund
	Calculation CaptainRefundCalculation `json:"calculation"`
}
type CaptainRefundService struct {
	repo *repository.CaptainRefundRepository
}

func NewCaptainRefundService() *CaptainRefundService {
	return &CaptainRefundService{repository.NewCaptainRefundRepository()}
}
func (s *CaptainRefundService) Settings() CaptainRefundSettings {
	cfg := CaptainRefundSettings{MaxTotalSP: 10_000_000, MaxAgeDays: 90, TaxRate: 30, AllowedSystemIDs: []int64{}}
	raw := repository.NewSysConfigRepository().GetString(refundSettingsKey, "")
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	return cfg
}
func (s *CaptainRefundService) requireRole(uid uint, admin bool) error {
	roles, err := repository.NewRoleRepository().GetUserRoleCodes(uid)
	if err != nil {
		return err
	}
	if admin {
		if slices.Contains(roles, model.RoleAdmin) || slices.Contains(roles, model.RoleSuperAdmin) {
			return nil
		}
	} else if slices.Contains(roles, model.RoleCaptain) {
		return nil
	}
	return errors.New("forbidden")
}
func (s *CaptainRefundService) UpdateSettings(uid uint, cfg CaptainRefundSettings) (CaptainRefundSettings, error) {
	if err := s.requireRole(uid, true); err != nil {
		return cfg, err
	}
	if cfg.MaxTotalSP < 0 || cfg.MaxAgeDays < 0 || cfg.MaxAgeDays > 36500 || math.IsNaN(cfg.TaxRate) || math.IsInf(cfg.TaxRate, 0) || cfg.TaxRate < 0 || cfg.TaxRate > 100 {
		return cfg, errors.New("invalid settings")
	}
	seen := map[int64]bool{}
	ids := []int64{}
	for _, id := range cfg.AllowedSystemIDs {
		if id <= 0 {
			return cfg, errors.New("invalid system ID")
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	cfg.AllowedSystemIDs = ids
	raw, err := json.Marshal(cfg)
	if err != nil {
		return cfg, err
	}
	err = repository.NewSysConfigRepository().SetMany([]repository.SysConfigUpsertItem{{Key: refundSettingsKey, Value: string(raw), Desc: "Unified newbro eligibility and captain refunds"}})
	if err == nil {
		err = NewAuditService().RecordEvent(context.Background(), AuditRecordInput{Category: "config", Action: "captain_refund_settings_update", ActorUserID: uid, ResourceType: "system_config", ResourceID: refundSettingsKey, Result: model.AuditResultSuccess, Details: map[string]interface{}{"settings": cfg}})
	}
	return cfg, err
}
func (s *CaptainRefundService) Options(uid uint) (interface{}, error) {
	if err := s.requireRole(uid, false); err != nil {
		return nil, err
	}
	chars, err := repository.NewEveCharacterRepository().ListByUserID(uid)
	type option struct {
		CharacterID   int64  `json:"character_id"`
		CharacterName string `json:"character_name"`
	}
	result := []option{}
	for _, c := range chars {
		result = append(result, option{c.CharacterID, c.CharacterName})
	}
	return struct {
		Characters []option              `json:"characters"`
		Settings   CaptainRefundSettings `json:"settings"`
	}{result, s.Settings()}, err
}
func validateRefundRequest(req CaptainRefundRequest, cfg CaptainRefundSettings, now time.Time) error {
	if req.CaptainCharacterID <= 0 || !slices.Contains(cfg.AllowedSystemIDs, req.SystemID) {
		return errors.New("invalid captain or system")
	}
	if !req.Start.Before(req.End) || req.Start.Before(now.Add(-30*24*time.Hour)) || req.End.After(now) {
		return errors.New("invalid UTC time range (last 30 days only)")
	}
	if len(req.MemberNames) > 250 {
		return errors.New("too many members")
	}
	return nil
}
func (s *CaptainRefundService) Calculate(uid uint, req CaptainRefundRequest, now time.Time) (*CaptainRefundCalculation, error) {
	if err := s.requireRole(uid, false); err != nil {
		return nil, err
	}
	cfg := s.Settings()
	if err := validateRefundRequest(req, cfg, now); err != nil {
		return nil, err
	}
	owned, err := repository.NewEveCharacterRepository().ListByUserID(uid)
	if err != nil {
		return nil, err
	}
	var captain *model.EveCharacter
	for i := range owned {
		if owned[i].CharacterID == req.CaptainCharacterID {
			captain = &owned[i]
			break
		}
	}
	if captain == nil {
		return nil, errors.New("captain character must belong to applicant")
	}
	names := []string{}
	seenNames := map[string]bool{}
	for _, name := range req.MemberNames {
		key := strings.ToLower(strings.TrimSpace(name))
		if key != "" && !seenNames[key] {
			names = append(names, key)
			seenNames[key] = true
		}
	}
	chars, err := s.repo.FindCharacters(names)
	if err != nil {
		return nil, err
	}
	found := map[string]bool{}
	for _, c := range chars {
		found[strings.ToLower(c.CharacterName)] = true
	}
	calc := &CaptainRefundCalculation{Request: req, Settings: cfg, Members: []RefundMember{}, IgnoredNames: []string{}, Journals: []RefundJournal{}, Conflicts: []RefundConflict{}}
	for _, name := range names {
		if !found[name] {
			calc.IgnoredNames = append(calc.IgnoredNames, name)
		}
	}
	team := []model.EveCharacter{*captain}
	ids := []int64{captain.CharacterID}
	for _, c := range chars {
		if c.CharacterID != captain.CharacterID {
			team = append(team, c)
			ids = append(ids, c.CharacterID)
		}
	}
	updates, err := s.repo.WalletUpdates(ids)
	if err != nil {
		return nil, err
	}
	updateMap := map[int64]time.Time{}
	for _, u := range updates {
		updateMap[u.CharacterID] = u.UpdateTime
	}
	userIDs := []uint{}
	for _, c := range team {
		userIDs = append(userIDs, c.UserID)
	}
	eligibility, err := NewUnifiedNewbroService().EvaluateUsers(uniqueNonZeroUserIDs(userIDs), cfg, now)
	if err != nil {
		return nil, err
	}
	for i, c := range team {
		r := eligibility[c.UserID]
		m := RefundMember{CharacterID: c.CharacterID, CharacterName: c.CharacterName, UserID: c.UserID, IsCaptain: i == 0, Eligibility: *r}
		if t, ok := updateMap[c.CharacterID]; ok {
			m.WalletUpdatedAt = &t
		}
		calc.Members = append(calc.Members, m)
	}
	journals, err := s.repo.Journals(ids, req.SystemID, req.Start, req.End)
	if err != nil {
		return nil, err
	}
	calculateRefundJournals(calc, journals)
	journalIDs := make([]int64, 0, len(calc.Journals))
	for _, journal := range calc.Journals {
		journalIDs = append(journalIDs, journal.JournalID)
	}
	claims, err := s.repo.Claims(ids, journalIDs)
	if err != nil {
		return nil, err
	}
	occupied := map[[2]int64]model.CaptainRefundClaim{}
	for _, c := range claims {
		occupied[[2]int64{c.CharacterID, c.JournalID}] = c
	}
	for _, j := range calc.Journals {
		if c, ok := occupied[[2]int64{j.CharacterID, j.JournalID}]; ok {
			calc.Conflicts = append(calc.Conflicts, RefundConflict{j.CharacterID, j.JournalID, j.Date, c.RefundID})
		}
	}
	calc.CanSubmit = calc.HasNewbro && calc.Reward > 0 && len(calc.Conflicts) == 0
	return calc, nil
}
func calculateRefundJournals(calc *CaptainRefundCalculation, journals []model.EVECharacterWalletJournal) {
	captains := []model.EVECharacterWalletJournal{}
	for _, j := range journals {
		if j.CharacterID == calc.Request.CaptainCharacterID {
			captains = append(captains, j)
		}
	}
	indices := map[int64]int{}
	for i, m := range calc.Members {
		indices[m.CharacterID] = i
	}
	seen := map[[2]int64]bool{}
	for _, j := range journals {
		index, ok := indices[j.CharacterID]
		if !ok || j.RefType != "bounty_prizes" || j.Amount <= 0 || j.ContextID != calc.Request.SystemID || j.Date.Before(calc.Request.Start) || !j.Date.Before(calc.Request.End) {
			continue
		}
		key := [2]int64{j.CharacterID, j.ID}
		if seen[key] {
			continue
		}
		seen[key] = true
		row := RefundJournal{CharacterID: j.CharacterID, JournalID: j.ID, Date: j.Date, Amount: j.Amount}
		if j.CharacterID != calc.Request.CaptainCharacterID {
			var best *model.EVECharacterWalletJournal
			for k := range captains {
				c := &captains[k]
				delta := c.Date.Sub(j.Date).Abs()
				if delta > 5*time.Minute {
					continue
				}
				if best == nil || delta < best.Date.Sub(j.Date).Abs() || (delta == best.Date.Sub(j.Date).Abs() && (c.Date.Before(best.Date) || (c.Date.Equal(best.Date) && c.ID < best.ID))) {
					best = c
				}
			}
			if best == nil {
				continue
			}
			row.MatchJournalID = best.ID
			row.MatchDate = &best.Date
			if calc.Members[index].UserID != calc.Members[0].UserID && calc.Members[index].Eligibility.IsCurrentlyNewbro {
				calc.HasNewbro = true
			}
		}
		calc.Journals = append(calc.Journals, row)
		calc.Members[index].Bounty += j.Amount
		calc.Bounty += j.Amount
	}
	calc.Reward = math.Round(calc.Bounty*(calc.Settings.TaxRate/100)/model.ISKPerFuxiCoin*100) / 100
}
func (s *CaptainRefundService) Create(uid uint, req CaptainRefundRequest) (*CaptainRefundView, error) {
	calc, err := s.Calculate(uid, req, time.Now())
	if err != nil {
		return nil, err
	}
	if !calc.CanSubmit {
		return nil, errors.New("application cannot be submitted; preview eligibility, reward and conflicting claims")
	}
	snapshot, err := json.Marshal(calc)
	if err != nil {
		return nil, err
	}
	row := model.CaptainRefund{UserID: uid, Status: "pending", Snapshot: string(snapshot), Reward: calc.Reward}
	claims := make([]model.CaptainRefundClaim, 0, len(calc.Journals))
	for _, j := range calc.Journals {
		claims = append(claims, model.CaptainRefundClaim{CharacterID: j.CharacterID, JournalID: j.JournalID, JournalAt: j.Date})
	}
	err = s.repo.Transaction(func(tx *gorm.DB) error {
		if e := s.repo.Create(tx, &row, claims); e != nil {
			return e
		}
		return recordRefundAudit(tx, uid, row.ID, "submit")
	})
	if err != nil {
		return nil, errors.New("application could not be saved; refresh preview to check concurrent claims")
	}
	return refundView(row)
}
func refundView(row model.CaptainRefund) (*CaptainRefundView, error) {
	v := &CaptainRefundView{CaptainRefund: row}
	err := json.Unmarshal([]byte(row.Snapshot), &v.Calculation)
	return v, err
}
func (s *CaptainRefundService) List(uid uint, admin bool, page, size int) ([]CaptainRefundView, int64, error) {
	if err := s.requireRole(uid, admin); err != nil {
		return nil, 0, err
	}
	owner := uid
	if admin {
		owner = 0
	}
	rows, total, err := s.repo.List(owner, page, size)
	if err != nil {
		return nil, 0, err
	}
	items := []CaptainRefundView{}
	for _, r := range rows {
		v, e := refundView(r)
		if e != nil {
			return nil, 0, e
		}
		items = append(items, *v)
	}
	return items, total, nil
}
func (s *CaptainRefundService) Detail(uid, id uint, admin bool) (*CaptainRefundView, error) {
	if err := s.requireRole(uid, admin); err != nil {
		return nil, err
	}
	row, err := s.repo.Get(id)
	if err != nil {
		return nil, errors.New("application not found")
	}
	if !admin && row.UserID != uid {
		return nil, errors.New("application not found")
	}
	return refundView(*row)
}
func recordRefundAudit(tx *gorm.DB, uid, id uint, action string) error {
	return NewAuditService().RecordEventTx(tx, AuditRecordInput{Category: "captain_refund", Action: action, ActorUserID: uid, ResourceType: "captain_refund", ResourceID: fmt.Sprint(id), Result: model.AuditResultSuccess})
}
func (s *CaptainRefundService) Review(uid, id uint, approve bool, reason string) error {
	if err := s.requireRole(uid, true); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if !approve && reason == "" {
		return errors.New("rejection reason required")
	}
	return s.repo.Transaction(func(tx *gorm.DB) error {
		row, err := s.repo.Lock(tx, id)
		if err != nil {
			return errors.New("application not found")
		}
		if row.Status != "pending" {
			return errors.New("application already reviewed")
		}
		now := time.Now()
		row.ReviewedAt = &now
		row.ReviewedBy = uid
		row.Status = "rejected"
		row.RejectReason = reason
		if approve {
			row.Status = "approved"
			row.RejectReason = ""
			row.WalletRefID = fmt.Sprintf("captain_refund:%d", id)
		}
		if err = s.repo.Review(tx, row); err != nil {
			return err
		}
		if approve {
			if err = NewSysWalletService().ApplyWalletDeltaByOperatorTx(tx, row.UserID, uid, row.Reward, "Captain tax refund", model.WalletRefCaptainRefund, row.WalletRefID); err != nil {
				return err
			}
		}
		return recordRefundAudit(tx, uid, id, row.Status)
	})
}
