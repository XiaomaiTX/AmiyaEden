package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

var fuxiAdminHexColorPattern = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

type UserVisibleError struct {
	message string
}

func (e *UserVisibleError) Error() string {
	return e.message
}

func NewUserVisibleError(message string) error {
	return &UserVisibleError{message: message}
}

func IsUserVisibleError(err error) bool {
	var target *UserVisibleError
	return errors.As(err, &target)
}

func wrapFuxiAdminLookupError(err error, notFoundMessage string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return NewUserVisibleError(notFoundMessage)
	}
	return err
}

// FuxiAdminService 伏羲管理人员名录业务层
type FuxiAdminService struct {
	repo             *repository.FuxiAdminRepository
	userRepo         *repository.UserRepository
	eveCharacterRepo *repository.EveCharacterRepository
	fleetRepo        *repository.FleetRepository
	welfareRepo      *repository.WelfareRepository
	shopRepo         *repository.ShopRepository
}

func NewFuxiAdminService() *FuxiAdminService {
	return &FuxiAdminService{
		repo:             repository.NewFuxiAdminRepository(),
		userRepo:         repository.NewUserRepository(),
		eveCharacterRepo: repository.NewEveCharacterRepository(),
		fleetRepo:        repository.NewFleetRepository(),
		welfareRepo:      repository.NewWelfareRepository(),
		shopRepo:         repository.NewShopRepository(),
	}
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

type FuxiAdminManageAdmin struct {
	model.FuxiAdmin
	WelfareDeliveryOffset int   `json:"welfare_delivery_offset"`
	FleetLedCount         int64 `json:"fleet_led_count"`
	WelfareDeliveryCount  int64 `json:"welfare_delivery_count"`
}

type FuxiAdminManageTierWithAdmins struct {
	model.FuxiAdminTier
	Admins []FuxiAdminManageAdmin `json:"admins"`
}

type FuxiAdminManageDirectoryResponse struct {
	Config model.FuxiAdminConfig           `json:"config"`
	Tiers  []FuxiAdminManageTierWithAdmins `json:"tiers"`
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
	Nickname       string `json:"nickname"`
	CharacterName  string `json:"character_name"`
	Description    string `json:"description"`
	ContactQQ      string `json:"contact_qq"`
	ContactDiscord string `json:"contact_discord"`
	CharacterID    int64  `json:"character_id"`
}

type FuxiAdminUpdateAdminRequest struct {
	TierID                *uint   `json:"tier_id"`
	Nickname              *string `json:"nickname"`
	CharacterName         *string `json:"character_name"`
	Description           *string `json:"description"`
	ContactQQ             *string `json:"contact_qq"`
	ContactDiscord        *string `json:"contact_discord"`
	CharacterID           *int64  `json:"character_id"`
	WelfareDeliveryOffset *int    `json:"welfare_delivery_offset"`
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
			return nil, NewUserVisibleError("字体大小必须在 8–32 之间")
		}
		cfg.BaseFontSize = *req.BaseFontSize
	}
	if req.CardWidth != nil {
		if *req.CardWidth < 160 || *req.CardWidth > 420 {
			return nil, NewUserVisibleError("卡片宽度必须在 160–420 之间")
		}
		cfg.CardWidth = *req.CardWidth
	}
	if err := applyOptionalFuxiAdminColor(&cfg.PageBackgroundColor, req.PageBackgroundColor, "页面背景色"); err != nil {
		return nil, err
	}
	if err := applyOptionalFuxiAdminColor(&cfg.CardBackgroundColor, req.CardBackgroundColor, "卡片背景色"); err != nil {
		return nil, err
	}
	if err := applyOptionalFuxiAdminColor(&cfg.CardBorderColor, req.CardBorderColor, "卡片边框色"); err != nil {
		return nil, err
	}
	if err := applyOptionalFuxiAdminColor(&cfg.TierTitleColor, req.TierTitleColor, "层级标题颜色"); err != nil {
		return nil, err
	}
	if err := applyOptionalFuxiAdminColor(&cfg.NameTextColor, req.NameTextColor, "姓名文字颜色"); err != nil {
		return nil, err
	}
	if err := applyOptionalFuxiAdminColor(&cfg.BodyTextColor, req.BodyTextColor, "其他文字颜色"); err != nil {
		return nil, err
	}
	if err := s.repo.UpsertConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyOptionalFuxiAdminColor(target *string, input *string, label string) error {
	if input == nil {
		return nil
	}
	color, err := normalizeFuxiAdminHexColor(*input, label)
	if err != nil {
		return err
	}
	*target = color
	return nil
}

func normalizeFuxiAdminHexColor(input string, label string) (string, error) {
	color := strings.TrimSpace(input)
	if !fuxiAdminHexColorPattern.MatchString(color) {
		return "", NewUserVisibleError(label + "必须是十六进制颜色值")
	}
	return color, nil
}

func (s *FuxiAdminService) loadDirectoryData() (*model.FuxiAdminConfig, []model.FuxiAdminTier, map[uint][]model.FuxiAdmin, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return nil, nil, nil, err
	}

	tiers, err := s.repo.ListTiers()
	if err != nil {
		return nil, nil, nil, err
	}

	tierIDs := make([]uint, len(tiers))
	for i, tier := range tiers {
		tierIDs[i] = tier.ID
	}

	admins, err := s.repo.ListAdminsByTierIDs(tierIDs)
	if err != nil {
		return nil, nil, nil, err
	}

	adminsByTier := make(map[uint][]model.FuxiAdmin)
	for _, admin := range admins {
		adminsByTier[admin.TierID] = append(adminsByTier[admin.TierID], admin)
	}

	return cfg, tiers, adminsByTier, nil
}

// ─── Directory (public) ───

func (s *FuxiAdminService) GetDirectory() (*FuxiAdminDirectoryResponse, error) {
	cfg, tiers, adminsByTier, err := s.loadDirectoryData()
	if err != nil {
		return nil, err
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

func collectFuxiAdminCharacterIDs(admins []model.FuxiAdmin) []int64 {
	characterIDs := make([]int64, 0, len(admins))
	seenCharacterIDs := make(map[int64]struct{}, len(admins))
	for _, admin := range admins {
		if admin.CharacterID == 0 {
			continue
		}
		if _, exists := seenCharacterIDs[admin.CharacterID]; exists {
			continue
		}
		seenCharacterIDs[admin.CharacterID] = struct{}{}
		characterIDs = append(characterIDs, admin.CharacterID)
	}
	return characterIDs
}

func collectFuxiAdminUserIDs(characterToUser map[int64]uint) []uint {
	userIDs := make([]uint, 0, len(characterToUser))
	seenUserIDs := make(map[uint]struct{}, len(characterToUser))
	for _, userID := range characterToUser {
		if userID == 0 {
			continue
		}
		if _, exists := seenUserIDs[userID]; exists {
			continue
		}
		seenUserIDs[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}
	return userIDs
}

func buildManageAdmin(
	admin model.FuxiAdmin,
	characterToUser map[int64]uint,
	fleetCounts map[uint]int64,
	welfareCounts map[uint]int64,
	shopCounts map[uint]int64,
) FuxiAdminManageAdmin {
	userID := characterToUser[admin.CharacterID]
	return FuxiAdminManageAdmin{
		FuxiAdmin:             admin,
		WelfareDeliveryOffset: admin.WelfareDeliveryOffset,
		FleetLedCount:         fleetCounts[userID],
		WelfareDeliveryCount:  welfareCounts[userID] + shopCounts[userID] + int64(admin.WelfareDeliveryOffset),
	}
}

func (s *FuxiAdminService) resolveFuxiAdminUsers(characterIDs []int64) (map[int64]uint, error) {
	characterToUser, err := s.eveCharacterRepo.ListUserIDsByCharacterIDs(characterIDs)
	if err != nil {
		return nil, err
	}

	missingCharacterIDs := make([]int64, 0, len(characterIDs))
	for _, characterID := range characterIDs {
		if _, exists := characterToUser[characterID]; exists {
			continue
		}
		missingCharacterIDs = append(missingCharacterIDs, characterID)
	}
	if len(missingCharacterIDs) == 0 {
		return characterToUser, nil
	}

	primaryCharacterToUser, err := s.userRepo.ListByPrimaryCharacterIDs(missingCharacterIDs)
	if err != nil {
		return nil, err
	}
	for characterID, userID := range primaryCharacterToUser {
		characterToUser[characterID] = userID
	}

	return characterToUser, nil
}

func (s *FuxiAdminService) buildManageAdminLookup(admins []model.FuxiAdmin) (map[uint]FuxiAdminManageAdmin, error) {
	result := make(map[uint]FuxiAdminManageAdmin, len(admins))
	if len(admins) == 0 {
		return result, nil
	}

	characterIDs := collectFuxiAdminCharacterIDs(admins)
	characterToUser := make(map[int64]uint)
	if len(characterIDs) > 0 {
		var err error
		characterToUser, err = s.resolveFuxiAdminUsers(characterIDs)
		if err != nil {
			return nil, err
		}
	}

	userIDs := collectFuxiAdminUserIDs(characterToUser)
	fleetCounts := make(map[uint]int64)
	welfareCounts := make(map[uint]int64)
	shopCounts := make(map[uint]int64)
	if len(userIDs) > 0 {
		var err error
		fleetCounts, err = s.fleetRepo.CountByCreatorUserIDs(userIDs)
		if err != nil {
			return nil, err
		}
		welfareCounts, err = s.welfareRepo.CountDeliveredByReviewers(userIDs)
		if err != nil {
			return nil, err
		}
		shopCounts, err = s.shopRepo.CountDeliveredByReviewers(userIDs)
		if err != nil {
			return nil, err
		}
	}

	for _, admin := range admins {
		result[admin.ID] = buildManageAdmin(admin, characterToUser, fleetCounts, welfareCounts, shopCounts)
	}

	return result, nil
}

func (s *FuxiAdminService) GetManageAdmin(id uint) (*FuxiAdminManageAdmin, error) {
	admin, err := s.repo.GetAdminByID(id)
	if err != nil {
		return nil, wrapFuxiAdminLookupError(err, "管理员不存在")
	}

	manageAdminsByID, err := s.buildManageAdminLookup([]model.FuxiAdmin{*admin})
	if err != nil {
		return nil, err
	}
	manageAdmin := manageAdminsByID[admin.ID]
	return &manageAdmin, nil
}

func (s *FuxiAdminService) GetManageDirectory() (*FuxiAdminManageDirectoryResponse, error) {
	cfg, tiers, adminsByTier, err := s.loadDirectoryData()
	if err != nil {
		return nil, err
	}

	allAdmins := make([]model.FuxiAdmin, 0)
	for _, admins := range adminsByTier {
		allAdmins = append(allAdmins, admins...)
	}

	manageAdminsByID, err := s.buildManageAdminLookup(allAdmins)
	if err != nil {
		return nil, err
	}

	result := make([]FuxiAdminManageTierWithAdmins, len(tiers))
	for i, tier := range tiers {
		admins := adminsByTier[tier.ID]
		manageAdmins := make([]FuxiAdminManageAdmin, len(admins))
		for j, admin := range admins {
			manageAdmins[j] = manageAdminsByID[admin.ID]
		}
		result[i] = FuxiAdminManageTierWithAdmins{
			FuxiAdminTier: tier,
			Admins:        manageAdmins,
		}
	}

	return &FuxiAdminManageDirectoryResponse{Config: *cfg, Tiers: result}, nil
}

// ─── Tiers ───

func (s *FuxiAdminService) ListTiers() ([]model.FuxiAdminTier, error) {
	return s.repo.ListTiers()
}

func (s *FuxiAdminService) CreateTier(req *FuxiAdminCreateTierRequest) (*model.FuxiAdminTier, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, NewUserVisibleError("层级名称不能为空")
	}
	maxSort, err := s.repo.MaxTierSortOrder()
	if err != nil {
		return nil, err
	}
	tier := &model.FuxiAdminTier{
		Name:      name,
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
		return nil, wrapFuxiAdminLookupError(err, "层级不存在")
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, NewUserVisibleError("层级名称不能为空")
		}
		tier.Name = name
	}
	if err := s.repo.UpdateTier(tier); err != nil {
		return nil, err
	}
	return tier, nil
}

func (s *FuxiAdminService) DeleteTier(id uint) error {
	err := s.repo.Transaction(func(txRepo *repository.FuxiAdminRepository) error {
		if err := txRepo.DeleteAdminsByTierID(id); err != nil {
			return err
		}
		return txRepo.DeleteTier(id)
	})
	if err != nil {
		return wrapFuxiAdminLookupError(err, "层级不存在")
	}
	return nil
}

// ─── Admins ───

func (s *FuxiAdminService) CreateAdmin(req *FuxiAdminCreateAdminRequest) (*model.FuxiAdmin, error) {
	nickname := strings.TrimSpace(req.Nickname)
	if nickname == "" {
		return nil, NewUserVisibleError("昵称不能为空")
	}
	if _, err := s.repo.GetTierByID(req.TierID); err != nil {
		return nil, wrapFuxiAdminLookupError(err, "层级不存在")
	}
	admin := &model.FuxiAdmin{
		TierID:         req.TierID,
		Nickname:       nickname,
		CharacterName:  strings.TrimSpace(req.CharacterName),
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
		return nil, wrapFuxiAdminLookupError(err, "管理员不存在")
	}
	if req.TierID != nil {
		if _, err := s.repo.GetTierByID(*req.TierID); err != nil {
			return nil, wrapFuxiAdminLookupError(err, "层级不存在")
		}
		admin.TierID = *req.TierID
	}
	if req.Nickname != nil {
		nickname := strings.TrimSpace(*req.Nickname)
		if nickname == "" {
			return nil, NewUserVisibleError("昵称不能为空")
		}
		admin.Nickname = nickname
	}
	if req.CharacterName != nil {
		admin.CharacterName = strings.TrimSpace(*req.CharacterName)
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
	if req.WelfareDeliveryOffset != nil {
		if *req.WelfareDeliveryOffset < 0 {
			return nil, NewUserVisibleError("福利发放次数偏移不能为负数")
		}
		admin.WelfareDeliveryOffset = *req.WelfareDeliveryOffset
	}
	if err := s.repo.UpdateAdmin(admin); err != nil {
		return nil, err
	}
	return admin, nil
}

func (s *FuxiAdminService) DeleteAdmin(id uint) error {
	if _, err := s.repo.GetAdminByID(id); err != nil {
		return wrapFuxiAdminLookupError(err, "管理员不存在")
	}
	return s.repo.DeleteAdmin(id)
}
