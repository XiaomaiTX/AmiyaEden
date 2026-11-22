package service

import (
	"fmt"
	"sort"
	"strings"
)

type skillPlanEFTName struct {
	name string
	line int
}

func stripEFTOfflineSuffix(s string) string {
	s = strings.TrimSpace(s)
	const suffix = "/offline"
	if len(s) >= len(suffix) && strings.EqualFold(s[len(s)-len(suffix):], suffix) {
		return strings.TrimSpace(s[:len(s)-len(suffix)])
	}
	return s
}

// EFT is transient input. Slot positions and quantities do not affect requirements.
func parseSkillPlanEFTNames(text string) ([]skillPlanEFTName, error) {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	header := parseEFTHeader(lines[0])
	if header == nil {
		return nil, fmt.Errorf("EFT 格式错误：缺少 [舰船, 装配名] 头部")
	}
	names := []skillPlanEFTName{{header.ShipType, 1}}
	for i, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "[Empty") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			return nil, fmt.Errorf("EFT 第 %d 行格式错误", i+2)
		}
		line = stripEFTOfflineSuffix(line)
		if match := countRegex.FindStringSubmatch(line); match != nil {
			line = strings.TrimSpace(match[1])
		}
		for _, name := range strings.SplitN(line, ",", 2) {
			name = stripEFTOfflineSuffix(name)
			if name == "" {
				return nil, fmt.Errorf("EFT 第 %d 行缺少物品名称", i+2)
			}
			names = append(names, skillPlanEFTName{name, i + 2})
		}
	}
	return names, nil
}

func (s *SkillPlanService) parseSkillPlanEFT(text string) ([]SkillPlanSkillReq, error) {
	entries, err := parseSkillPlanEFTNames(text)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.name)
	}
	resolved, err := s.sdeRepo.GetTypeIDsByNames(names)
	if err != nil {
		return nil, err
	}
	typeIDs := make([]int, 0, len(entries))
	seen := make(map[int64]bool)
	for _, entry := range entries {
		id, ok := resolved[entry.name]
		if !ok {
			return nil, fmt.Errorf("EFT 第 %d 行未找到物品「%s」，请使用英文名称", entry.line, entry.name)
		}
		if !seen[id] {
			typeIDs = append(typeIDs, int(id))
			seen[id] = true
		}
	}
	requirements, err := s.sdeRepo.GetShipSkillRequirements(typeIDs)
	if err != nil {
		return nil, err
	}
	skills := make([]SkillPlanSkillReq, 0, len(requirements))
	for _, requirement := range requirements {
		skills = append(skills, SkillPlanSkillReq{requirement.SkillTypeID, requirement.RequiredLevel})
	}
	skills = normalizeSkillPlanRequirements(skills)
	sort.Slice(skills, func(i, j int) bool { return skills[i].SkillTypeID < skills[j].SkillTypeID })
	return skills, nil
}
