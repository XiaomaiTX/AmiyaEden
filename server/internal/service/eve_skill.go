package service

import (
	"amiya-eden/internal/repository"
	"fmt"
)

type EveSkillService struct {
	skillRepo *repository.EveSkillRepository
	sdeRepo   *repository.SdeRepository
}

func NewEveSkillService() *EveSkillService {
	return &EveSkillService{
		skillRepo: repository.NewEveSkillRepository(),
		sdeRepo:   repository.NewSdeRepository(),
	}
}

type EveCharacterSkill struct {
	SkillID            int   `json:"skill_id"`
	ActiveLevel        int   `json:"active_level"`
	TrainedLevel       int   `json:"trained_level"`
	SkillpointsInSkill int64 `json:"skillpoints_in_skill"`
}

type SkillGroupTotal struct {
	GroupID int64 `json:"group_id"`
	Num     int   `json:"num"`
}
type EveSkillResponse struct {
	TotalSP   int64               `json:"total_sp"`
	SkillList []EveCharacterSkill `json:"skill_list"`
	Totals    []SkillGroupTotal   `json:"totals"`
}

func (s *EveSkillService) GetEveCharacterSkills(characterID int) (*EveSkillResponse, error) {
	result := &EveSkillResponse{}

	skill, err := s.skillRepo.GetSkill(characterID)
	if err != nil {
		return nil, fmt.Errorf("get skill for character %d: %w", characterID, err)
	}
	result.TotalSP = skill.TotalSP

	list, err := s.skillRepo.GetSkillList(characterID)
	if err != nil {
		return nil, fmt.Errorf("get skill list for character %d: %w", characterID, err)
	}

	skillIDs := make([]int, 0, len(list))
	result.SkillList = make([]EveCharacterSkill, 0, len(list))
	for _, sk := range list {
		skillIDs = append(skillIDs, sk.SkillID)
		result.SkillList = append(result.SkillList, EveCharacterSkill{
			SkillID:            sk.SkillID,
			ActiveLevel:        sk.ActiveLevel,
			TrainedLevel:       sk.TrainedLevel,
			SkillpointsInSkill: sk.SkillpointsInSkill,
		})
	}

	published := true
	typeInfos, err := s.sdeRepo.GetTypes(skillIDs, &published, "en")
	if err != nil {
		return nil, fmt.Errorf("get skill type info for character %d: %w", characterID, err)
	}

	groups := make(map[int64]int)
	for _, t := range typeInfos {
		groups[int64(t.GroupID)]++
	}

	for gid, num := range groups {
		result.Totals = append(result.Totals, SkillGroupTotal{GroupID: gid, Num: num})
	}

	return result, nil
}
