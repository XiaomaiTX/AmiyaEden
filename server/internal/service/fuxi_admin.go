package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
)

// FuxiAdminService 伏羲管理人员名录业务层
type FuxiAdminService struct {
	repo *repository.FuxiAdminRepository
}

func NewFuxiAdminService() *FuxiAdminService {
	return &FuxiAdminService{repo: repository.NewFuxiAdminRepository()}
}

// ─── Response types ───

type FuxiAdminTierWithAdmins struct {
	model.FuxiAdminTier
	Admins []model.FuxiAdmin `json:"admins"`
}

type FuxiAdminDirectoryResponse struct {
	Config model.FuxiAdminConfig     `json:"config"`
	Tiers  []FuxiAdminTierWithAdmins `json:"tiers"`
}

// ─── Request types ───

type FuxiAdminUpdateConfigRequest struct {
	BaseFontSize *int `json:"base_font_size"`
}

type FuxiAdminCreateTierRequest struct {
	Name string `json:"name"`
}

type FuxiAdminUpdateTierRequest struct {
	Name *string `json:"name"`
}

type FuxiAdminCreateAdminRequest struct {
	TierID         uint   `json:"tier_id"`
	Name           string `json:"name"`
	Title          string `json:"title"`
	ContactQQ      string `json:"contact_qq"`
	ContactDiscord string `json:"contact_discord"`
	CharacterID    int64  `json:"character_id"`
}

type FuxiAdminUpdateAdminRequest struct {
	TierID         *uint   `json:"tier_id"`
	Name           *string `json:"name"`
	Title          *string `json:"title"`
	ContactQQ      *string `json:"contact_qq"`
	ContactDiscord *string `json:"contact_discord"`
	CharacterID    *int64  `json:"character_id"`
}

// ─── Config ───

func (s *FuxiAdminService) GetConfig() (*model.FuxiAdminConfig, error) {
	cfg, err := s.repo.GetConfig()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		def := model.DefaultFuxiAdminConfig()
		if err := s.repo.UpsertConfig(&def); err != nil {
			return nil, err
		}
		return &def, nil
	}
	return cfg, nil
}

func (s *FuxiAdminService) UpdateConfig(req *FuxiAdminUpdateConfigRequest) (*model.FuxiAdminConfig, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return nil, err
	}
	if req.BaseFontSize != nil {
		if *req.BaseFontSize < 8 || *req.BaseFontSize > 32 {
			return nil, errors.New("字体大小必须在 8–32 之间")
		}
		cfg.BaseFontSize = *req.BaseFontSize
	}
	if err := s.repo.UpsertConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ─── Directory (public) ───

func (s *FuxiAdminService) GetDirectory() (*FuxiAdminDirectoryResponse, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return nil, err
	}

	tiers, err := s.repo.ListTiers()
	if err != nil {
		return nil, err
	}

	tierIDs := make([]uint, len(tiers))
	for i, t := range tiers {
		tierIDs[i] = t.ID
	}

	admins, err := s.repo.ListAdminsByTierIDs(tierIDs)
	if err != nil {
		return nil, err
	}

	// Group admins by tier
	adminsByTier := make(map[uint][]model.FuxiAdmin)
	for _, a := range admins {
		adminsByTier[a.TierID] = append(adminsByTier[a.TierID], a)
	}

	result := make([]FuxiAdminTierWithAdmins, len(tiers))
	for i, tier := range tiers {
		result[i] = FuxiAdminTierWithAdmins{
			FuxiAdminTier: tier,
			Admins:        adminsByTier[tier.ID],
		}
		if result[i].Admins == nil {
			result[i].Admins = []model.FuxiAdmin{}
		}
	}

	return &FuxiAdminDirectoryResponse{Config: *cfg, Tiers: result}, nil
}

// ─── Tiers ───

func (s *FuxiAdminService) ListTiers() ([]model.FuxiAdminTier, error) {
	return s.repo.ListTiers()
}

func (s *FuxiAdminService) CreateTier(req *FuxiAdminCreateTierRequest) (*model.FuxiAdminTier, error) {
	if req.Name == "" {
		return nil, errors.New("层级名称不能为空")
	}
	maxSort, err := s.repo.MaxTierSortOrder()
	if err != nil {
		return nil, err
	}
	tier := &model.FuxiAdminTier{
		Name:      req.Name,
		SortOrder: maxSort + 1,
	}
	if err := s.repo.CreateTier(tier); err != nil {
		return nil, err
	}
	return tier, nil
}

func (s *FuxiAdminService) UpdateTier(id uint, req *FuxiAdminUpdateTierRequest) (*model.FuxiAdminTier, error) {
	tier, err := s.repo.GetTierByID(id)
	if err != nil {
		return nil, errors.New("层级不存在")
	}
	if req.Name != nil {
		if *req.Name == "" {
			return nil, errors.New("层级名称不能为空")
		}
		tier.Name = *req.Name
	}
	if err := s.repo.UpdateTier(tier); err != nil {
		return nil, err
	}
	return tier, nil
}

func (s *FuxiAdminService) DeleteTier(id uint) error {
	if err := s.repo.DeleteAdminsByTierID(id); err != nil {
		return err
	}
	return s.repo.DeleteTier(id)
}

// ─── Admins ───

func (s *FuxiAdminService) CreateAdmin(req *FuxiAdminCreateAdminRequest) (*model.FuxiAdmin, error) {
	if req.Name == "" {
		return nil, errors.New("姓名不能为空")
	}
	if _, err := s.repo.GetTierByID(req.TierID); err != nil {
		return nil, errors.New("层级不存在")
	}
	admin := &model.FuxiAdmin{
		TierID:         req.TierID,
		Name:           req.Name,
		Title:          req.Title,
		ContactQQ:      req.ContactQQ,
		ContactDiscord: req.ContactDiscord,
		CharacterID:    req.CharacterID,
	}
	if err := s.repo.CreateAdmin(admin); err != nil {
		return nil, err
	}
	return admin, nil
}

func (s *FuxiAdminService) UpdateAdmin(id uint, req *FuxiAdminUpdateAdminRequest) (*model.FuxiAdmin, error) {
	admin, err := s.repo.GetAdminByID(id)
	if err != nil {
		return nil, errors.New("管理员不存在")
	}
	if req.TierID != nil {
		if _, err := s.repo.GetTierByID(*req.TierID); err != nil {
			return nil, errors.New("层级不存在")
		}
		admin.TierID = *req.TierID
	}
	if req.Name != nil {
		if *req.Name == "" {
			return nil, errors.New("姓名不能为空")
		}
		admin.Name = *req.Name
	}
	if req.Title != nil {
		admin.Title = *req.Title
	}
	if req.ContactQQ != nil {
		admin.ContactQQ = *req.ContactQQ
	}
	if req.ContactDiscord != nil {
		admin.ContactDiscord = *req.ContactDiscord
	}
	if req.CharacterID != nil {
		admin.CharacterID = *req.CharacterID
	}
	if err := s.repo.UpdateAdmin(admin); err != nil {
		return nil, err
	}
	return admin, nil
}

func (s *FuxiAdminService) DeleteAdmin(id uint) error {
	if _, err := s.repo.GetAdminByID(id); err != nil {
		return errors.New("管理员不存在")
	}
	return s.repo.DeleteAdmin(id)
}
