package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"errors"
	"sort"

	"gorm.io/gorm"
)

const (
	SkillReadinessSatisfied   = "satisfied"
	SkillReadinessUnsatisfied = "unsatisfied"
	SkillReadinessUnknown     = "unknown"
)

var ErrSkillReadinessRequest = errors.New("invalid skill readiness request")
var ErrSkillReadinessPlan = errors.New("invalid corporation skill plan")
var ErrSkillReadinessCharacter = errors.New("character is not in the requested scope")

type skillReadinessStore interface {
	Load(context.Context, repository.SkillReadinessFilter) (*repository.SkillReadinessSnapshot, error)
	Options(context.Context) ([]int64, []model.SkillPlan, error)
}

type SkillReadinessService struct {
	repo         skillReadinessStore
	resolveNames func(context.Context, []int64) map[int64]string
	loadTypes    func([]int, string) (map[int]repository.TypeInfo, error)
}

func NewSkillReadinessService() *SkillReadinessService {
	resolver := NewEntityNameResolver()
	planner := NewSkillPlanService()
	return &SkillReadinessService{
		repo:         repository.NewSkillReadinessRepository(),
		resolveNames: func(ctx context.Context, ids []int64) map[int64]string { return resolver.Resolve(ctx, ids).Names },
		loadTypes:    planner.loadTypeInfoMap,
	}
}

type SkillReadinessRequest struct {
	CorporationIDs []int64 `json:"corporation_ids"`
	PlanID         uint    `json:"plan_id"`
	Page           int     `json:"page"`
	PageSize       int     `json:"page_size"`
	Language       string  `json:"language"`
}

type SkillReadinessSummary struct {
	CharacterCount          int `json:"character_count"`
	UserCount               int `json:"user_count"`
	SatisfiedUserCount      int `json:"satisfied_user_count"`
	SatisfiedCharacterCount int `json:"satisfied_character_count"`
	UnknownCharacterCount   int `json:"unknown_character_count"`
}

type SkillReadinessUser struct {
	UserID                  uint   `json:"user_id"`
	Nickname                string `json:"nickname"`
	CharacterCount          int    `json:"character_count"`
	SatisfiedCharacterCount int    `json:"satisfied_character_count"`
	UnknownCharacterCount   int    `json:"unknown_character_count"`
}

type SkillReadinessResult struct {
	Summary  SkillReadinessSummary `json:"summary"`
	List     []SkillReadinessUser  `json:"list"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

type SkillReadinessCharacter struct {
	UserID          uint   `json:"user_id"`
	CharacterID     int64  `json:"character_id"`
	CharacterName   string `json:"character_name"`
	CorporationID   int64  `json:"corporation_id"`
	CorporationName string `json:"corporation_name"`
	Status          string `json:"status"`
	MatchedSkills   int    `json:"matched_skills"`
	TotalSkills     int    `json:"total_skills"`
	SkillUpdatedAt  *int64 `json:"skill_updated_at"`
	TokenInvalid    bool   `json:"token_invalid"`
}

type SkillReadinessSkill struct {
	SkillTypeID        int    `json:"skill_type_id"`
	SkillName          string `json:"skill_name"`
	GroupName          string `json:"group_name"`
	ActiveLevel        *int   `json:"active_level"`
	TrainedLevel       *int   `json:"trained_level"`
	SkillpointsInSkill *int64 `json:"skillpoints_in_skill"`
	RequiredLevel      *int   `json:"required_level,omitempty"`
	CurrentLevel       *int   `json:"current_level"`
	Satisfied          *bool  `json:"satisfied,omitempty"`
}

type SkillReadinessCharacterDetail struct {
	Character    SkillReadinessCharacter `json:"character"`
	Requirements []SkillReadinessSkill   `json:"requirements"`
	Skills       []SkillReadinessSkill   `json:"skills"`
}

type SkillReadinessCorporationOption struct {
	CorporationID   int64  `json:"corporation_id"`
	CorporationName string `json:"corporation_name"`
}

type SkillReadinessPlanOption struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ShipTypeID  *int   `json:"ship_type_id"`
}

type SkillReadinessOptions struct {
	Corporations []SkillReadinessCorporationOption `json:"corporations"`
	Plans        []SkillReadinessPlanOption        `json:"plans"`
}

func normalizeSkillReadinessRequest(req SkillReadinessRequest) (SkillReadinessRequest, error) {
	if req.PlanID == 0 || req.Page < 0 || req.PageSize < 0 || req.PageSize > 100 {
		return req, ErrSkillReadinessRequest
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}
	if req.Language == "" {
		req.Language = "zh"
	}
	if req.Language != "zh" && req.Language != "en" {
		return req, ErrSkillReadinessRequest
	}
	for _, id := range req.CorporationIDs {
		if id <= 0 {
			return req, ErrSkillReadinessRequest
		}
	}
	req.CorporationIDs = uniquePositiveInt64(req.CorporationIDs)
	return req, nil
}

func (s *SkillReadinessService) load(ctx context.Context, req SkillReadinessRequest, userID uint, characterID int64, allSkills bool) (*repository.SkillReadinessSnapshot, SkillReadinessRequest, error) {
	normalized, err := normalizeSkillReadinessRequest(req)
	if err != nil {
		return nil, normalized, err
	}
	snapshot, err := s.repo.Load(ctx, repository.SkillReadinessFilter{
		CorporationIDs: normalized.CorporationIDs, PlanID: normalized.PlanID, UserID: userID, CharacterID: characterID, AllSkills: allSkills,
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, normalized, ErrSkillReadinessPlan
	}
	if err != nil {
		return nil, normalized, err
	}
	if snapshot.Plan.PlanScope != model.SkillPlanScopeCorp || len(snapshot.Requirements) == 0 {
		return nil, normalized, ErrSkillReadinessPlan
	}
	return snapshot, normalized, nil
}

func evaluateSkillReadiness(snapshot *repository.SkillReadinessSnapshot) []SkillReadinessCharacter {
	updated := make(map[int64]int64, len(snapshot.Snapshots))
	for _, row := range snapshot.Snapshots {
		updated[row.CharacterID] = row.UpdatedTime
	}
	grouped := make(map[int64][]model.EveCharacterSkills)
	for _, row := range snapshot.Skills {
		grouped[row.CharacterID] = append(grouped[row.CharacterID], row)
	}
	result := make([]SkillReadinessCharacter, 0, len(snapshot.Characters))
	for _, character := range snapshot.Characters {
		row := SkillReadinessCharacter{
			UserID: character.UserID, CharacterID: character.CharacterID, CharacterName: character.CharacterName,
			CorporationID: character.CorporationID, TokenInvalid: character.TokenInvalid,
			Status: SkillReadinessUnknown, TotalSkills: len(snapshot.Requirements),
		}
		if timestamp, ok := updated[character.CharacterID]; ok {
			row.SkillUpdatedAt = &timestamp
			comparison := compareSkillPlanRequirements(snapshot.Plan, snapshot.Requirements, nil, buildCharacterSkillLevelMap(grouped[character.CharacterID]))
			row.MatchedSkills = comparison.MatchedSkills
			row.Status = SkillReadinessUnsatisfied
			if comparison.FullySatisfied {
				row.Status = SkillReadinessSatisfied
			}
		}
		result = append(result, row)
	}
	return result
}

func aggregateSkillReadiness(snapshot *repository.SkillReadinessSnapshot, rows []SkillReadinessCharacter, req SkillReadinessRequest) *SkillReadinessResult {
	result := &SkillReadinessResult{List: make([]SkillReadinessUser, 0), Page: req.Page, PageSize: req.PageSize}
	users := make(map[uint]*SkillReadinessUser)
	nicknames := make(map[uint]string)
	for _, row := range snapshot.Characters {
		nicknames[row.UserID] = row.Nickname
	}
	for _, row := range rows {
		user := users[row.UserID]
		if user == nil {
			user = &SkillReadinessUser{UserID: row.UserID, Nickname: nicknames[row.UserID]}
			users[row.UserID] = user
		}
		user.CharacterCount++
		if row.Status == SkillReadinessSatisfied {
			user.SatisfiedCharacterCount++
			result.Summary.SatisfiedCharacterCount++
		}
		if row.Status == SkillReadinessUnknown {
			user.UnknownCharacterCount++
			result.Summary.UnknownCharacterCount++
		}
	}
	all := make([]SkillReadinessUser, 0, len(users))
	for _, user := range users {
		all = append(all, *user)
		if user.SatisfiedCharacterCount > 0 {
			result.Summary.SatisfiedUserCount++
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].SatisfiedCharacterCount != all[j].SatisfiedCharacterCount {
			return all[i].SatisfiedCharacterCount > all[j].SatisfiedCharacterCount
		}
		return all[i].UserID < all[j].UserID
	})
	result.Summary.CharacterCount = len(rows)
	result.Summary.UserCount = len(users)
	result.Total = len(users)
	// Compare before multiplying to avoid overflowing a client-provided page number.
	if len(all) > 0 && req.Page <= (len(all)-1)/req.PageSize+1 {
		start := (req.Page - 1) * req.PageSize
		result.List = all[start:min(start+req.PageSize, len(all))]
	}
	return result
}

func (s *SkillReadinessService) Query(ctx context.Context, req SkillReadinessRequest) (*SkillReadinessResult, error) {
	snapshot, normalized, err := s.load(ctx, req, 0, 0, false)
	if err != nil {
		return nil, err
	}
	return aggregateSkillReadiness(snapshot, evaluateSkillReadiness(snapshot), normalized), nil
}

func (s *SkillReadinessService) UserCharacters(ctx context.Context, userID uint, req SkillReadinessRequest) ([]SkillReadinessCharacter, error) {
	if userID == 0 {
		return nil, ErrSkillReadinessRequest
	}
	snapshot, _, err := s.load(ctx, req, userID, 0, false)
	if err != nil {
		return nil, err
	}
	rows := evaluateSkillReadiness(snapshot)
	s.decorateCorporations(ctx, rows)
	return rows, nil
}

func (s *SkillReadinessService) decorateCorporations(ctx context.Context, rows []SkillReadinessCharacter) {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.CorporationID)
	}
	names := s.resolveNames(ctx, uniquePositiveInt64(ids))
	for i := range rows {
		rows[i].CorporationName = names[rows[i].CorporationID]
	}
}

func (s *SkillReadinessService) Options(ctx context.Context) (*SkillReadinessOptions, error) {
	ids, plans, err := s.repo.Options(ctx)
	if err != nil {
		return nil, err
	}
	names := s.resolveNames(ctx, ids)
	result := &SkillReadinessOptions{Corporations: make([]SkillReadinessCorporationOption, 0, len(ids)), Plans: make([]SkillReadinessPlanOption, 0, len(plans))}
	for _, id := range ids {
		result.Corporations = append(result.Corporations, SkillReadinessCorporationOption{CorporationID: id, CorporationName: names[id]})
	}
	for _, plan := range plans {
		result.Plans = append(result.Plans, SkillReadinessPlanOption{ID: plan.ID, Title: plan.Title, Description: plan.Description, ShipTypeID: plan.ShipTypeID})
	}
	return result, nil
}

func (s *SkillReadinessService) CharacterSkills(ctx context.Context, characterID int64, req SkillReadinessRequest) (*SkillReadinessCharacterDetail, error) {
	if characterID <= 0 {
		return nil, ErrSkillReadinessRequest
	}
	snapshot, normalized, err := s.load(ctx, req, 0, characterID, true)
	if err != nil {
		return nil, err
	}
	rows := evaluateSkillReadiness(snapshot)
	if len(rows) != 1 {
		return nil, ErrSkillReadinessCharacter
	}
	s.decorateCorporations(ctx, rows)
	result := &SkillReadinessCharacterDetail{Character: rows[0], Requirements: make([]SkillReadinessSkill, 0), Skills: make([]SkillReadinessSkill, 0)}
	known := rows[0].Status != SkillReadinessUnknown
	skills := make(map[int]model.EveCharacterSkills)
	ids := make([]int, 0, len(snapshot.Skills)+len(snapshot.Requirements))
	if known {
		for _, row := range snapshot.Skills {
			skills[row.SkillID] = row
			ids = append(ids, row.SkillID)
		}
	}
	for _, row := range snapshot.Requirements {
		ids = append(ids, row.SkillTypeID)
	}
	types, err := s.loadTypes(ids, normalized.Language)
	if err != nil {
		return nil, err
	}
	levels := buildCharacterSkillLevelMap(snapshot.Skills)
	makeSkill := func(id int) SkillReadinessSkill {
		info := types[id]
		row := SkillReadinessSkill{SkillTypeID: id, SkillName: info.TypeName, GroupName: info.GroupName}
		if known {
			skill := skills[id]
			level := levels[id]
			row.ActiveLevel = &skill.ActiveLevel
			row.TrainedLevel = &skill.TrainedLevel
			row.SkillpointsInSkill = &skill.SkillpointsInSkill
			row.CurrentLevel = &level
		}
		return row
	}
	for _, requirement := range snapshot.Requirements {
		row := makeSkill(requirement.SkillTypeID)
		required := requirement.RequiredLevel
		row.RequiredLevel = &required
		if known {
			satisfied := *row.CurrentLevel >= required
			row.Satisfied = &satisfied
		}
		result.Requirements = append(result.Requirements, row)
	}
	if known {
		for _, skill := range snapshot.Skills {
			result.Skills = append(result.Skills, makeSkill(skill.SkillID))
		}
	}
	return result, nil
}
