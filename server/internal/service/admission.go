package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/utils"

	"gorm.io/gorm"
)

type AdmissionService struct {
	repo     *repository.AdmissionRepository
	roleRepo *repository.RoleRepository
	userRepo *repository.UserRepository
	charRepo *repository.EveCharacterRepository
	// blacklistRepo screens applicants; see blacklistHitsTx.
	blacklistRepo *repository.BlacklistRepository
}

func NewAdmissionService() *AdmissionService {
	return &AdmissionService{repo: repository.NewAdmissionRepository(), roleRepo: repository.NewRoleRepository(), userRepo: repository.NewUserRepository(), charRepo: repository.NewEveCharacterRepository(), blacklistRepo: repository.NewBlacklistRepository()}
}

type AdmissionCharacterOption struct {
	CharacterID   int64  `json:"character_id"`
	CharacterName string `json:"character_name"`
	CorporationID int64  `json:"corporation_id"`
}

type AdmissionTarget struct {
	CorporationID int64  `json:"corporation_id"`
	Name          string `json:"name"`
}

type AdmissionContext struct {
	IsMember                 bool                       `json:"is_member"`
	CanCreateNewMember       bool                       `json:"can_create_new_member"`
	CanCreateAlt             bool                       `json:"can_create_alt"`
	QQ                       string                     `json:"qq"`
	DiscordID                string                     `json:"discord_id"`
	Targets                  []AdmissionTarget          `json:"targets"`
	Characters               []AdmissionCharacterOption `json:"characters"`
	ReviewableCorporationIDs []int64                    `json:"reviewable_corporation_ids"`
}

func admissionTargets() []AdmissionTarget {
	return []AdmissionTarget{{CorporationID: model.SystemCorporationID, Name: "Fuxi Legion"}, {CorporationID: model.AdmissionFMACorporationID, Name: "FMA"}}
}

func (s *AdmissionService) Context(userID uint) (*AdmissionContext, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	characters, err := s.charRepo.ListByUserID(userID)
	if err != nil {
		return nil, err
	}
	scope, err := s.reviewableCorporations(global.DB, userID)
	if err != nil {
		return nil, err
	}
	allowSet := make(map[int64]struct{})
	for _, id := range utils.GetAllowCorporations() {
		allowSet[id] = struct{}{}
	}
	isMember := hasAnyAllowedCharacter(characters, allowSet)
	options := make([]AdmissionCharacterOption, 0, len(characters))
	for _, character := range characters {
		options = append(options, AdmissionCharacterOption{CharacterID: character.CharacterID, CharacterName: character.CharacterName, CorporationID: character.CorporationID})
	}
	return &AdmissionContext{
		IsMember: isMember, CanCreateNewMember: !isMember && user.ProfileComplete(), CanCreateAlt: isMember && user.ProfileComplete(),
		QQ: user.QQ, DiscordID: user.DiscordID, Targets: admissionTargets(), Characters: options, ReviewableCorporationIDs: scope,
	}, nil
}

// Recruitment authority always comes from local snapshots with corporation
// provenance, never from a cached global recruiter claim in a JWT.
func (s *AdmissionService) qualifyingRecruiterCorporations(db *gorm.DB, userID uint) ([]int64, error) {
	targets := make([]int64, 0, 2)
	for _, target := range admissionTargets() {
		targets = append(targets, target.CorporationID)
	}
	return s.repo.UserCorporationsWithRoles(db, userID, targets, []string{"Personnel_Manager", "Director"})
}

func (s *AdmissionService) reviewableCorporations(db *gorm.DB, userID uint) ([]int64, error) {
	roles, err := s.roleRepo.GetUserRoleCodesTx(db, userID)
	if err != nil {
		return nil, err
	}
	if model.ContainsAnyRole(roles, model.RoleAdmin, model.RoleSuperAdmin) {
		return []int64{model.SystemCorporationID, model.AdmissionFMACorporationID}, nil
	}
	return s.qualifyingRecruiterCorporations(db, userID)
}

// Count only applications the user can actually review, using the same local
// authority snapshots as the review endpoints rather than cached JWT roles.
func (s *AdmissionService) PendingReviewCount(userID uint) (int64, error) {
	scope, err := s.reviewableCorporations(global.DB, userID)
	if err != nil || len(scope) == 0 {
		return 0, err
	}
	// The reviewer's own pending application is not actionable by them.
	return s.repo.CountPendingInCorporations(global.DB, scope, userID)
}
