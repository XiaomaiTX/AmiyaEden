package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
	"regexp"
	"strings"
)

var fuxiAdminHexColorPattern = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

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
	BaseFontSize        *int    `json:"base_font_size"`
	CardWidth           *int    `json:"card_width"`
	PageBackgroundColor *string `json:"page_background_color"`
	CardBackgroundColor *string `json:"card_background_color"`
	CardBorderColor     *string `json:"card_border_color"`
	TierTitleColor      *string `json:"tier_title_color"`
	NameTextColor       *string `json:"name_text_color"`
	BodyTextColor       *string `json:"body_text_color"`
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
	Description    string `json:"description"`
	ContactQQ      string `json:"contact_qq"`
	ContactDiscord string `json:"contact_discord"`
	CharacterID    int64  `json:"character_id"`
}

type FuxiAdminUpdateAdminRequest struct {
	TierID         *uint   `json:"tier_id"`
	Name           *string `json:"name"`
	Title          *string `json:"title"`
	Description    *string `json:"description"`
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
	if req.CardWidth != nil {
		if *req.CardWidth < 160 || *req.CardWidth > 420 {
			return nil, errors.New("卡片宽度必须在 160–420 之间")
		}
		cfg.CardWidth = *req.CardWidth
	}
	if req.PageBackgroundColor != nil {
		color, err := normalizeFuxiAdminHexColor(*req.PageBackgroundColor, "页面背景色")
		if err != nil {
			return nil, err
		}
		cfg.PageBackgroundColor = color
	}
	if req.CardBackgroundColor != nil {
		color, err := normalizeFuxiAdminHexColor(*req.CardBackgroundColor, "卡片背景色")
		if err != nil {
			return nil, err
		}
		cfg.CardBackgroundColor = color
	}
	if req.CardBorderColor != nil {
		color, err := normalizeFuxiAdminHexColor(*req.CardBorderColor, "卡片边框色")
		if err != nil {
			return nil, err
		}
		cfg.CardBorderColor = color
	}
	if req.TierTitleColor != nil {
		color, err := normalizeFuxiAdminHexColor(*req.TierTitleColor, "层级标题颜色")
		if err != nil {
			return nil, err
		}
		cfg.TierTitleColor = color
	}
	if req.NameTextColor != nil {
		color, err := normalizeFuxiAdminHexColor(*req.NameTextColor, "姓名文字颜色")
		if err != nil {
			return nil, err
		}
		cfg.NameTextColor = color
	}
	if req.BodyTextColor != nil {
		color, err := normalizeFuxiAdminHexColor(*req.BodyTextColor, "其他文字颜色")
		if err != nil {
			return nil, err
		}
		cfg.BodyTextColor = color
	}
	if err := s.repo.UpsertConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func normalizeFuxiAdminHexColor(input string, label string) (string, error) {
	color := strings.TrimSpace(input)
	if !fuxiAdminHexColorPattern.MatchString(color) {
		return "", errors.New(label + "必须是十六进制颜色值")
	}
	return color, nil
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
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("姓名不能为空")
	}
	if _, err := s.repo.GetTierByID(req.TierID); err != nil {
		return nil, errors.New("层级不存在")
	}
	admin := &model.FuxiAdmin{
		TierID:         req.TierID,
		Name:           name,
		Title:          strings.TrimSpace(req.Title),
		Description:    strings.TrimSpace(req.Description),
		ContactQQ:      strings.TrimSpace(req.ContactQQ),
		ContactDiscord: strings.TrimSpace(req.ContactDiscord),
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
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.New("姓名不能为空")
		}
		admin.Name = name
	}
	if req.Title != nil {
		admin.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		admin.Description = strings.TrimSpace(*req.Description)
	}
	if req.ContactQQ != nil {
		admin.ContactQQ = strings.TrimSpace(*req.ContactQQ)
	}
	if req.ContactDiscord != nil {
		admin.ContactDiscord = strings.TrimSpace(*req.ContactDiscord)
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
