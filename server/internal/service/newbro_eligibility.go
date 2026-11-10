package service

import (
	"amiya-eden/internal/repository"
	"math"
	"time"
)

type NewbroCharacterSnapshot struct {
	CharacterID  int64      `json:"character_id"`
	TotalSP      int64      `json:"total_sp"`
	HasSkillData bool       `json:"has_skill_data"`
	Birthday     *time.Time `json:"birthday"`
}
type NewbroEligibilityResult struct {
	IsCurrentlyNewbro  bool                      `json:"is_currently_newbro"`
	DisqualifiedReason string                    `json:"disqualified_reason"`
	TotalSP            int64                     `json:"total_sp"`
	OldestBirthday     *time.Time                `json:"oldest_birthday"`
	MaxTotalSP         int64                     `json:"max_total_sp"`
	MaxAgeDays         int                       `json:"max_age_days"`
	EvaluatedAt        time.Time                 `json:"evaluated_at"`
	Characters         []NewbroCharacterSnapshot `json:"characters"`
}

func EvaluateUnifiedNewbro(characters []NewbroCharacterSnapshot, maxSP int64, maxDays int, now time.Time) NewbroEligibilityResult {
	now = now.UTC()
	if characters == nil {
		characters = []NewbroCharacterSnapshot{}
	}
	r := NewbroEligibilityResult{MaxTotalSP: maxSP, MaxAgeDays: maxDays, EvaluatedAt: now, Characters: characters}
	if len(characters) == 0 {
		r.DisqualifiedReason = "no_characters"
		return r
	}
	missing := false
	for _, c := range characters {
		if !c.HasSkillData || c.TotalSP < 0 || c.Birthday == nil || c.Birthday.After(now) {
			missing = true
		}
		if c.TotalSP >= 0 {
			if c.TotalSP > math.MaxInt64-r.TotalSP {
				r.TotalSP = math.MaxInt64
			} else {
				r.TotalSP += c.TotalSP
			}
		}
		if c.Birthday != nil && (r.OldestBirthday == nil || c.Birthday.Before(*r.OldestBirthday)) {
			r.OldestBirthday = c.Birthday
		}
	}
	if missing {
		r.DisqualifiedReason = "missing_character_data"
		return r
	}
	if r.TotalSP > maxSP {
		r.DisqualifiedReason = "skill_points_too_high"
		return r
	}
	if r.OldestBirthday.Before(now.AddDate(0, 0, -maxDays)) {
		r.DisqualifiedReason = "character_too_old"
		return r
	}
	r.IsCurrentlyNewbro = true
	return r
}

type UnifiedNewbroService struct{}

func NewUnifiedNewbroService() *UnifiedNewbroService { return &UnifiedNewbroService{} }
func (s *UnifiedNewbroService) Evaluate(userID uint, now time.Time) (*NewbroEligibilityResult, error) {
	results, err := s.EvaluateUsers([]uint{userID}, NewCaptainRefundService().Settings(), now)
	if err != nil {
		return nil, err
	}
	return results[userID], nil
}

// EvaluateUsers batches character and skill reads and uses one consistent rule snapshot.
func (s *UnifiedNewbroService) EvaluateUsers(userIDs []uint, cfg CaptainRefundSettings, now time.Time) (map[uint]*NewbroEligibilityResult, error) {
	chars, err := repository.NewEveCharacterRepository().ListByUserIDs(userIDs)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(chars))
	for _, c := range chars {
		ids = append(ids, c.CharacterID)
	}
	totals, err := repository.NewEveSkillRepository().GetSkillTotalsByCharacterIDs(ids)
	if err != nil {
		return nil, err
	}
	groups := map[uint][]NewbroCharacterSnapshot{}
	for _, c := range chars {
		sp, ok := totals[c.CharacterID]
		groups[c.UserID] = append(groups[c.UserID], NewbroCharacterSnapshot{c.CharacterID, sp, ok, c.Birthday})
	}
	results := map[uint]*NewbroEligibilityResult{}
	for _, uid := range userIDs {
		r := EvaluateUnifiedNewbro(groups[uid], cfg.MaxTotalSP, cfg.MaxAgeDays, now)
		results[uid] = &r
	}
	return results, nil
}
