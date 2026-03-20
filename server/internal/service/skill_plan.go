package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// SkillPlanService 军团技能计划业务逻辑层
type SkillPlanService struct {
	repo    *repository.SkillPlanRepository
	sdeRepo *repository.SdeRepository
}

func NewSkillPlanService() *SkillPlanService {
	return &SkillPlanService{
		repo:    repository.NewSkillPlanRepository(),
		sdeRepo: repository.NewSdeRepository(),
	}
}

// SkillPlanSkillReq 单条技能要求请求
type SkillPlanSkillReq struct {
	SkillTypeID   int `json:"skill_type_id" binding:"required"`
	RequiredLevel int `json:"required_level" binding:"required"`
}

// CreateSkillPlanRequest 创建技能计划请求
type CreateSkillPlanRequest struct {
	Title       string              `json:"title" binding:"required"`
	Description string              `json:"description"`
	Skills      []SkillPlanSkillReq `json:"skills"`
	SkillsText  string              `json:"skills_text"`
}

// UpdateSkillPlanRequest 更新技能计划请求
type UpdateSkillPlanRequest struct {
	Title       string              `json:"title" binding:"required"`
	Description string              `json:"description"`
	Skills      []SkillPlanSkillReq `json:"skills"`
	SkillsText  string              `json:"skills_text"`
}

// SkillPlanListItemResp 技能计划列表项
type SkillPlanListItemResp struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedBy   uint   `json:"created_by"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	SkillCount  int    `json:"skill_count"`
}

// SkillPlanSkillResp 单条技能要求响应
type SkillPlanSkillResp struct {
	ID            uint   `json:"id"`
	SkillPlanID   uint   `json:"skill_plan_id"`
	SkillTypeID   int    `json:"skill_type_id"`
	SkillName     string `json:"skill_name"`
	GroupName     string `json:"group_name"`
	RequiredLevel int    `json:"required_level"`
	Sort          int    `json:"sort"`
}

// SkillPlanDetailResp 技能计划详情响应
type SkillPlanDetailResp struct {
	ID          uint                 `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	CreatedBy   uint                 `json:"created_by"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
	SkillCount  int                  `json:"skill_count"`
	Skills      []SkillPlanSkillResp `json:"skills"`
}

// CreateSkillPlan 创建技能计划
func (s *SkillPlanService) CreateSkillPlan(userID uint, req *CreateSkillPlanRequest, lang string) (*SkillPlanDetailResp, error) {
	title := strings.TrimSpace(req.Title)
	description := strings.TrimSpace(req.Description)

	normalizedSkills, err := s.normalizeSkillPlanInputs(req.Skills, req.SkillsText, lang)
	if err != nil {
		return nil, err
	}
	if err := validateSkillPlanPayload(title, normalizedSkills); err != nil {
		return nil, err
	}

	plan := &model.SkillPlan{
		Title:       title,
		Description: description,
		CreatedBy:   userID,
	}
	skills := buildSkillPlanModels(normalizedSkills)

	if err := s.repo.Create(plan, skills); err != nil {
		return nil, err
	}

	createdSkills, err := s.repo.ListSkillsByPlanID(plan.ID)
	if err != nil {
		return nil, err
	}

	return s.buildDetailResp(plan, createdSkills, lang)
}

// ListSkillPlans 获取技能计划列表
func (s *SkillPlanService) ListSkillPlans(page, pageSize int, keyword string) ([]SkillPlanListItemResp, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	plans, total, err := s.repo.List(page, pageSize, strings.TrimSpace(keyword))
	if err != nil {
		return nil, 0, err
	}

	planIDs := make([]uint, 0, len(plans))
	for _, plan := range plans {
		planIDs = append(planIDs, plan.ID)
	}

	allSkills, err := s.repo.ListSkillsByPlanIDs(planIDs)
	if err != nil {
		return nil, 0, err
	}

	skillCountMap := make(map[uint]int, len(planIDs))
	for _, skill := range allSkills {
		skillCountMap[skill.SkillPlanID]++
	}

	result := make([]SkillPlanListItemResp, 0, len(plans))
	for _, plan := range plans {
		result = append(result, SkillPlanListItemResp{
			ID:          plan.ID,
			Title:       plan.Title,
			Description: plan.Description,
			CreatedBy:   plan.CreatedBy,
			CreatedAt:   plan.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   plan.UpdatedAt.Format(time.RFC3339),
			SkillCount:  skillCountMap[plan.ID],
		})
	}

	return result, total, nil
}

// GetSkillPlan 获取技能计划详情
func (s *SkillPlanService) GetSkillPlan(id uint, lang string) (*SkillPlanDetailResp, error) {
	plan, err := s.repo.GetByID(id)
	if err != nil {
		return nil, errors.New("技能计划不存在")
	}

	skills, err := s.repo.ListSkillsByPlanID(id)
	if err != nil {
		return nil, err
	}

	return s.buildDetailResp(plan, skills, lang)
}

// UpdateSkillPlan 更新技能计划
func (s *SkillPlanService) UpdateSkillPlan(id uint, userID uint, userRoles []string, req *UpdateSkillPlanRequest, lang string) (*SkillPlanDetailResp, error) {
	plan, err := s.repo.GetByID(id)
	if err != nil {
		return nil, errors.New("技能计划不存在")
	}
	if !canManageSkillPlan(plan.CreatedBy, userID, userRoles) {
		return nil, errors.New("权限不足")
	}

	title := strings.TrimSpace(req.Title)
	description := strings.TrimSpace(req.Description)
	normalizedSkills, err := s.normalizeSkillPlanInputs(req.Skills, req.SkillsText, lang)
	if err != nil {
		return nil, err
	}
	if err := validateSkillPlanPayload(title, normalizedSkills); err != nil {
		return nil, err
	}

	plan.Title = title
	plan.Description = description

	skills := buildSkillPlanModels(normalizedSkills)
	if err := s.repo.Update(plan, skills); err != nil {
		return nil, err
	}

	updatedSkills, err := s.repo.ListSkillsByPlanID(plan.ID)
	if err != nil {
		return nil, err
	}

	return s.buildDetailResp(plan, updatedSkills, lang)
}

// DeleteSkillPlan 删除技能计划
func (s *SkillPlanService) DeleteSkillPlan(id uint, userID uint, userRoles []string) error {
	plan, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("技能计划不存在")
	}
	if !canManageSkillPlan(plan.CreatedBy, userID, userRoles) {
		return errors.New("权限不足")
	}
	return s.repo.Delete(id)
}

func validateSkillPlanPayload(title string, skills []SkillPlanSkillReq) error {
	if title == "" {
		return errors.New("计划标题不能为空")
	}
	if len([]rune(title)) > 256 {
		return errors.New("计划标题不能超过256个字符")
	}
	if len(skills) == 0 {
		return errors.New("请至少添加一个技能要求")
	}

	for idx, skill := range skills {
		if skill.SkillTypeID <= 0 {
			return fmt.Errorf("第 %d 条技能要求缺少有效技能", idx+1)
		}
		if skill.RequiredLevel < 1 || skill.RequiredLevel > 5 {
			return fmt.Errorf("第 %d 条技能要求等级必须在 1 到 5 之间", idx+1)
		}
	}

	return nil
}

type normalizedSkillEntry struct {
	SkillTypeID   int
	RequiredLevel int
	Order         int
}

var skillPlanTextLinePattern = regexp.MustCompile(`^(.*?)[[:space:]]+([1-5]|I|II|III|IV|V)$`)

func (s *SkillPlanService) normalizeSkillPlanInputs(skills []SkillPlanSkillReq, skillsText string, lang string) ([]SkillPlanSkillReq, error) {
	trimmedText := strings.TrimSpace(skillsText)
	if trimmedText != "" {
		return s.parseSkillPlanText(trimmedText, lang)
	}
	return normalizeSkillPlanRequirements(skills), nil
}

func normalizeSkillPlanRequirements(skills []SkillPlanSkillReq) []SkillPlanSkillReq {
	if len(skills) == 0 {
		return nil
	}

	normalizedMap := make(map[int]*normalizedSkillEntry, len(skills))
	order := make([]int, 0, len(skills))

	for idx, skill := range skills {
		if skill.SkillTypeID <= 0 {
			continue
		}
		if existing, ok := normalizedMap[skill.SkillTypeID]; ok {
			if skill.RequiredLevel > existing.RequiredLevel {
				existing.RequiredLevel = skill.RequiredLevel
			}
			continue
		}

		normalizedMap[skill.SkillTypeID] = &normalizedSkillEntry{
			SkillTypeID:   skill.SkillTypeID,
			RequiredLevel: skill.RequiredLevel,
			Order:         idx,
		}
		order = append(order, skill.SkillTypeID)
	}

	result := make([]SkillPlanSkillReq, 0, len(order))
	for _, skillTypeID := range order {
		entry := normalizedMap[skillTypeID]
		result = append(result, SkillPlanSkillReq{
			SkillTypeID:   entry.SkillTypeID,
			RequiredLevel: entry.RequiredLevel,
		})
	}

	return result
}

func (s *SkillPlanService) parseSkillPlanText(skillsText string, lang string) ([]SkillPlanSkillReq, error) {
	lines := strings.Split(skillsText, "\n")
	entries := make([]parsedSkillPlanLine, 0, len(lines))
	names := make([]string, 0, len(lines))

	for idx, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		match := skillPlanTextLinePattern.FindStringSubmatch(line)
		if len(match) != 3 {
			return nil, fmt.Errorf("第 %d 行格式不正确，请使用“技能名 等级”格式", idx+1)
		}

		level, err := parseSkillPlanLevelToken(match[2])
		if err != nil {
			return nil, fmt.Errorf("第 %d 行等级无效: %w", idx+1, err)
		}

		name := strings.TrimSpace(match[1])
		if name == "" {
			return nil, fmt.Errorf("第 %d 行缺少技能名称", idx+1)
		}

		entries = append(entries, parsedSkillPlanLine{
			Name:  name,
			Level: level,
			Line:  idx + 1,
		})
		names = append(names, name)
	}

	if len(entries) == 0 {
		return nil, nil
	}

	nameMap, err := s.loadSkillNameMap(lang)
	if err != nil {
		return nil, err
	}

	missingNames := make([]string, 0)
	normalizedMap := make(map[int]*normalizedSkillEntry, len(entries))
	order := make([]int, 0, len(entries))

	for idx, entry := range entries {
		typeInfo, ok := nameMap[normalizeSkillPlanName(entry.Name)]
		if !ok {
			missingNames = append(missingNames, entry.Name)
			continue
		}

		if existing, exists := normalizedMap[typeInfo.TypeID]; exists {
			if entry.Level > existing.RequiredLevel {
				existing.RequiredLevel = entry.Level
			}
			continue
		}

		normalizedMap[typeInfo.TypeID] = &normalizedSkillEntry{
			SkillTypeID:   typeInfo.TypeID,
			RequiredLevel: entry.Level,
			Order:         idx,
		}
		order = append(order, typeInfo.TypeID)
	}

	if len(missingNames) > 0 {
		slices.Sort(missingNames)
		missingNames = slices.Compact(missingNames)
		return nil, fmt.Errorf("无法识别以下技能名称: %s", strings.Join(missingNames, ", "))
	}

	result := make([]SkillPlanSkillReq, 0, len(order))
	for _, skillTypeID := range order {
		entry := normalizedMap[skillTypeID]
		result = append(result, SkillPlanSkillReq{
			SkillTypeID:   entry.SkillTypeID,
			RequiredLevel: entry.RequiredLevel,
		})
	}

	return result, nil
}

type parsedSkillPlanLine struct {
	Name  string
	Level int
	Line  int
}

func parseSkillPlanLevelToken(token string) (int, error) {
	switch strings.ToUpper(strings.TrimSpace(token)) {
	case "1", "I":
		return 1, nil
	case "2", "II":
		return 2, nil
	case "3", "III":
		return 3, nil
	case "4", "IV":
		return 4, nil
	case "5", "V":
		return 5, nil
	default:
		return 0, errors.New("等级必须为 1 到 5")
	}
}

func normalizeSkillPlanName(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(name))), " ")
}

func (s *SkillPlanService) loadSkillNameMap(lang string) (map[string]repository.TypeInfo, error) {
	if lang == "" {
		lang = "zh"
	}

	result := make(map[string]repository.TypeInfo)
	addEntries := func(typeInfos []repository.TypeInfo) {
		for _, info := range typeInfos {
			normalizedName := normalizeSkillPlanName(info.TypeName)
			if normalizedName == "" {
				continue
			}
			result[normalizedName] = info
		}
	}

	langSkills, err := s.sdeRepo.GetTypesByCategoryID(16, lang)
	if err != nil {
		return nil, err
	}
	addEntries(langSkills)

	if lang != "en" {
		enSkills, err := s.sdeRepo.GetTypesByCategoryID(16, "en")
		if err != nil {
			return nil, err
		}
		addEntries(enSkills)
	}

	return result, nil
}

func canManageSkillPlan(createdBy uint, userID uint, userRoles []string) bool {
	if model.IsSuperAdmin(userRoles) {
		return true
	}
	if model.ContainsAnyRole(userRoles, model.RoleAdmin, model.RoleFC) {
		return true
	}
	return createdBy == userID
}

func buildSkillPlanModels(skills []SkillPlanSkillReq) []model.SkillPlanSkill {
	result := make([]model.SkillPlanSkill, 0, len(skills))
	for idx, skill := range skills {
		result = append(result, model.SkillPlanSkill{
			SkillTypeID:   skill.SkillTypeID,
			RequiredLevel: skill.RequiredLevel,
			Sort:          idx + 1,
		})
	}
	return result
}

func (s *SkillPlanService) buildDetailResp(plan *model.SkillPlan, skills []model.SkillPlanSkill, lang string) (*SkillPlanDetailResp, error) {
	typeInfoMap, err := s.loadSkillTypeInfoMap(skills, lang)
	if err != nil {
		return nil, err
	}

	detailSkills := make([]SkillPlanSkillResp, 0, len(skills))
	for _, skill := range skills {
		typeInfo := typeInfoMap[skill.SkillTypeID]
		detailSkills = append(detailSkills, SkillPlanSkillResp{
			ID:            skill.ID,
			SkillPlanID:   skill.SkillPlanID,
			SkillTypeID:   skill.SkillTypeID,
			SkillName:     typeInfo.TypeName,
			GroupName:     typeInfo.GroupName,
			RequiredLevel: skill.RequiredLevel,
			Sort:          skill.Sort,
		})
	}

	return &SkillPlanDetailResp{
		ID:          plan.ID,
		Title:       plan.Title,
		Description: plan.Description,
		CreatedBy:   plan.CreatedBy,
		CreatedAt:   plan.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   plan.UpdatedAt.Format(time.RFC3339),
		SkillCount:  len(detailSkills),
		Skills:      detailSkills,
	}, nil
}

func (s *SkillPlanService) loadSkillTypeInfoMap(skills []model.SkillPlanSkill, lang string) (map[int]repository.TypeInfo, error) {
	typeInfoMap := make(map[int]repository.TypeInfo, len(skills))
	if len(skills) == 0 {
		return typeInfoMap, nil
	}

	if lang == "" {
		lang = "zh"
	}

	skillIDs := make([]int, 0, len(skills))
	seen := make(map[int]struct{}, len(skills))
	for _, skill := range skills {
		if _, ok := seen[skill.SkillTypeID]; ok {
			continue
		}
		seen[skill.SkillTypeID] = struct{}{}
		skillIDs = append(skillIDs, skill.SkillTypeID)
	}

	published := true
	typeInfos, err := s.sdeRepo.GetTypes(skillIDs, &published, lang)
	if err != nil {
		return nil, err
	}
	for _, info := range typeInfos {
		typeInfoMap[info.TypeID] = info
	}

	missingIDs := make([]int, 0)
	for _, skillID := range skillIDs {
		info, ok := typeInfoMap[skillID]
		if !ok || info.TypeName == "" || info.GroupName == "" {
			missingIDs = append(missingIDs, skillID)
		}
	}

	if len(missingIDs) > 0 && lang != "en" {
		fallbackInfos, err := s.sdeRepo.GetTypes(missingIDs, &published, "en")
		if err != nil {
			return nil, err
		}
		for _, info := range fallbackInfos {
			current := typeInfoMap[info.TypeID]
			if current.TypeID == 0 || current.TypeName == "" {
				current.TypeID = info.TypeID
				current.TypeName = info.TypeName
			}
			if current.GroupName == "" {
				current.GroupName = info.GroupName
			}
			typeInfoMap[info.TypeID] = current
		}
	}

	for _, skillID := range skillIDs {
		info := typeInfoMap[skillID]
		if info.TypeID == 0 {
			info.TypeID = skillID
		}
		if info.TypeName == "" {
			info.TypeName = fmt.Sprintf("Type %d", skillID)
		}
		if info.GroupName == "" {
			info.GroupName = "-"
		}
		typeInfoMap[skillID] = info
	}

	return typeInfoMap, nil
}
