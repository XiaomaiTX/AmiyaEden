package service

import (
	"amiya-eden/internal/model"
	"time"
)

type MenteeEligibilityResult struct {
	IsEligible         bool   `json:"is_eligible"`
	DisqualifiedReason string `json:"disqualified_reason"`
}
type MentorEligibilityService struct{}

func NewMentorEligibilityService() *MentorEligibilityService { return &MentorEligibilityService{} }
func (s *MentorEligibilityService) EvaluateEligibility(userID uint) (*MenteeEligibilityResult, error) {
	r, err := NewUnifiedNewbroService().Evaluate(userID, time.Now())
	if err != nil {
		return nil, err
	}
	return &MenteeEligibilityResult{r.IsCurrentlyNewbro, r.DisqualifiedReason}, nil
}

func (s *MentorEligibilityService) EvaluateEligibilityWithCharacters(characters []model.EveCharacter) (*MenteeEligibilityResult, error) {
	r, err := NewUnifiedNewbroService().EvaluateWithCharacters(characters, time.Now())
	if err != nil {
		return nil, err
	}
	return &MenteeEligibilityResult{r.IsCurrentlyNewbro, r.DisqualifiedReason}, nil
}
