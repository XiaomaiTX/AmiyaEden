package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

var (
	fuxiHallHexColorPattern = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
)

var (
	fuxiHallAllowedPageKeys = map[string]struct{}{
		string(model.FuxiHallPageLeadership):   {},
		string(model.FuxiHallPageContributors): {},
	}
	fuxiHallDefaultPageTitles = map[string]string{
		string(model.FuxiHallPageLeadership):   "管理层",
		string(model.FuxiHallPageContributors): "重大贡献成员",
	}
	fuxiHallStylePresets = map[string]struct{}{
		"classic": {},
		"aurora":  {},
		"slate":   {},
	}
	fuxiHallBadgeTones = map[string]struct{}{
		"neutral": {},
		"dawn":    {},
		"steel":   {},
	}
	fuxiHallAvatarShapes = map[string]struct{}{
		"circle":  {},
		"rounded": {},
		"square":  {},
	}
)

// FuxiHallService 伏羲大厅业务层
type FuxiHallService struct {
	repo *repository.FuxiHallRepository
}

func NewFuxiHallService() *FuxiHallService {
	return &FuxiHallService{repo: repository.NewFuxiHallRepository()}
}

type FuxiHallPublicPageResponse struct {
	Page  model.FuxiHallPage   `json:"page"`
	Cards []model.FuxiHallCard `json:"cards"`
}

type FuxiHallUpdatePageRequest struct {
	Title           *string `json:"title"`
	Subtitle        *string `json:"subtitle"`
	DescriptionHTML *string `json:"description_html"`
}

type FuxiHallCreateCardRequest struct {
	PageKey           string `json:"page_key"`
	Nickname          string `json:"nickname"`
	MainCharacterID   int64  `json:"main_character_id"`
	MainCharacterName string `json:"main_character_name"`
	Title             string `json:"title"`
	DescriptionHTML   string `json:"description_html"`
	CoverImage        string `json:"cover_image"`
	StylePreset       string `json:"style_preset"`
	AccentColor       string `json:"accent_color"`
	BadgeTone         string `json:"badge_tone"`
	AvatarShape       string `json:"avatar_shape"`
	CoverHeight       *int   `json:"cover_height"`
	FontScale         *int   `json:"font_scale"`
	Visible           *bool  `json:"visible"`
}

type FuxiHallUpdateCardRequest struct {
	Nickname          *string `json:"nickname"`
	MainCharacterID   *int64  `json:"main_character_id"`
	MainCharacterName *string `json:"main_character_name"`
	Title             *string `json:"title"`
	DescriptionHTML   *string `json:"description_html"`
	CoverImage        *string `json:"cover_image"`
	StylePreset       *string `json:"style_preset"`
	AccentColor       *string `json:"accent_color"`
	BadgeTone         *string `json:"badge_tone"`
	AvatarShape       *string `json:"avatar_shape"`
	CoverHeight       *int    `json:"cover_height"`
	FontScale         *int    `json:"font_scale"`
	Visible           *bool   `json:"visible"`
}

type FuxiHallReorderRequest struct {
	PageKey    string `json:"page_key"`
	OrderedIDs []uint `json:"ordered_ids"`
}

func (s *FuxiHallService) GetPublicPage(pageKey string) (*FuxiHallPublicPageResponse, error) {
	normalizedPageKey, err := normalizeFuxiHallPageKey(pageKey)
	if err != nil {
		return nil, err
	}

	page, err := s.ensurePage(normalizedPageKey)
	if err != nil {
		return nil, err
	}

	cards, err := s.repo.ListCardsByPage(normalizedPageKey, true)
	if err != nil {
		return nil, err
	}
	if cards == nil {
		cards = []model.FuxiHallCard{}
	}

	return &FuxiHallPublicPageResponse{Page: *page, Cards: cards}, nil
}

func (s *FuxiHallService) GetPageConfig(pageKey string) (*model.FuxiHallPage, error) {
	normalizedPageKey, err := normalizeFuxiHallPageKey(pageKey)
	if err != nil {
		return nil, err
	}
	return s.ensurePage(normalizedPageKey)
}

func (s *FuxiHallService) UpdatePageConfig(pageKey string, req *FuxiHallUpdatePageRequest) (*model.FuxiHallPage, error) {
	normalizedPageKey, err := normalizeFuxiHallPageKey(pageKey)
	if err != nil {
		return nil, err
	}

	page, err := s.ensurePage(normalizedPageKey)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, NewUserVisibleError("页面标题不能为空")
		}
		page.Title = title
	}
	if req.Subtitle != nil {
		page.Subtitle = strings.TrimSpace(*req.Subtitle)
	}
	if req.DescriptionHTML != nil {
		page.DescriptionHTML = sanitizeRichTextHTML(*req.DescriptionHTML)
	}

	if err := s.repo.UpsertPage(page); err != nil {
		return nil, err
	}
	return page, nil
}

func (s *FuxiHallService) ListCards(pageKey string, visibleOnly bool) ([]model.FuxiHallCard, error) {
	normalizedPageKey, err := normalizeFuxiHallPageKey(pageKey)
	if err != nil {
		return nil, err
	}

	if _, err := s.ensurePage(normalizedPageKey); err != nil {
		return nil, err
	}

	cards, err := s.repo.ListCardsByPage(normalizedPageKey, visibleOnly)
	if err != nil {
		return nil, err
	}
	if cards == nil {
		return []model.FuxiHallCard{}, nil
	}
	return cards, nil
}

func (s *FuxiHallService) CreateCard(req *FuxiHallCreateCardRequest) (*model.FuxiHallCard, error) {
	pageKey, err := normalizeFuxiHallPageKey(req.PageKey)
	if err != nil {
		return nil, err
	}
	if _, err := s.ensurePage(pageKey); err != nil {
		return nil, err
	}

	nickname := strings.TrimSpace(req.Nickname)
	if nickname == "" {
		return nil, NewUserVisibleError("昵称不能为空")
	}
	mainCharacterName := strings.TrimSpace(req.MainCharacterName)
	if mainCharacterName == "" {
		return nil, NewUserVisibleError("主角色名称不能为空")
	}
	if req.MainCharacterID <= 0 {
		return nil, NewUserVisibleError("主角色 ID 必须大于 0")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, NewUserVisibleError("头衔不能为空")
	}

	stylePreset, err := normalizeFuxiHallStylePreset(req.StylePreset, true)
	if err != nil {
		return nil, err
	}
	accentColor, err := normalizeFuxiHallColor(req.AccentColor, "#3b82f6", "强调色")
	if err != nil {
		return nil, err
	}
	badgeTone, err := normalizeFuxiHallEnum(req.BadgeTone, "neutral", fuxiHallBadgeTones, "徽章风格")
	if err != nil {
		return nil, err
	}
	avatarShape, err := normalizeFuxiHallEnum(req.AvatarShape, "circle", fuxiHallAvatarShapes, "头像形状")
	if err != nil {
		return nil, err
	}
	coverHeight, err := normalizeFuxiHallNumber(req.CoverHeight, 180, 96, 320, "封面高度")
	if err != nil {
		return nil, err
	}
	fontScale, err := normalizeFuxiHallNumber(req.FontScale, 14, 12, 20, "字体大小")
	if err != nil {
		return nil, err
	}

	maxSortOrder, err := s.repo.MaxCardSortOrder(pageKey)
	if err != nil {
		return nil, err
	}

	visible := true
	if req.Visible != nil {
		visible = *req.Visible
	}

	card := &model.FuxiHallCard{
		PageKey:           pageKey,
		Nickname:          nickname,
		MainCharacterID:   req.MainCharacterID,
		MainCharacterName: mainCharacterName,
		Title:             title,
		DescriptionHTML:   sanitizeRichTextHTML(req.DescriptionHTML),
		CoverImage:        strings.TrimSpace(req.CoverImage),
		StylePreset:       stylePreset,
		AccentColor:       accentColor,
		BadgeTone:         badgeTone,
		AvatarShape:       avatarShape,
		CoverHeight:       coverHeight,
		FontScale:         fontScale,
		Visible:           visible,
		SortOrder:         maxSortOrder + 1,
	}
	if err := s.repo.CreateCard(card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *FuxiHallService) UpdateCard(id uint, req *FuxiHallUpdateCardRequest) (*model.FuxiHallCard, error) {
	card, err := s.repo.GetCardByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUserVisibleError("卡片不存在")
		}
		return nil, err
	}

	updates := map[string]interface{}{}

	if req.Nickname != nil {
		value := strings.TrimSpace(*req.Nickname)
		if value == "" {
			return nil, NewUserVisibleError("昵称不能为空")
		}
		updates["nickname"] = value
	}
	if req.MainCharacterName != nil {
		value := strings.TrimSpace(*req.MainCharacterName)
		if value == "" {
			return nil, NewUserVisibleError("主角色名称不能为空")
		}
		updates["main_character_name"] = value
	}
	if req.MainCharacterID != nil {
		if *req.MainCharacterID <= 0 {
			return nil, NewUserVisibleError("主角色 ID 必须大于 0")
		}
		updates["main_character_id"] = *req.MainCharacterID
	}
	if req.Title != nil {
		value := strings.TrimSpace(*req.Title)
		if value == "" {
			return nil, NewUserVisibleError("头衔不能为空")
		}
		updates["title"] = value
	}
	if req.DescriptionHTML != nil {
		updates["description_html"] = sanitizeRichTextHTML(*req.DescriptionHTML)
	}
	if req.CoverImage != nil {
		updates["cover_image"] = strings.TrimSpace(*req.CoverImage)
	}
	if req.StylePreset != nil {
		value, err := normalizeFuxiHallStylePreset(*req.StylePreset, false)
		if err != nil {
			return nil, err
		}
		updates["style_preset"] = value
	}
	if req.AccentColor != nil {
		value, err := normalizeFuxiHallColor(*req.AccentColor, "", "强调色")
		if err != nil {
			return nil, err
		}
		updates["accent_color"] = value
	}
	if req.BadgeTone != nil {
		value, err := normalizeFuxiHallEnum(*req.BadgeTone, "", fuxiHallBadgeTones, "徽章风格")
		if err != nil {
			return nil, err
		}
		updates["badge_tone"] = value
	}
	if req.AvatarShape != nil {
		value, err := normalizeFuxiHallEnum(*req.AvatarShape, "", fuxiHallAvatarShapes, "头像形状")
		if err != nil {
			return nil, err
		}
		updates["avatar_shape"] = value
	}
	if req.CoverHeight != nil {
		value, err := normalizeFuxiHallNumber(req.CoverHeight, 0, 96, 320, "封面高度")
		if err != nil {
			return nil, err
		}
		updates["cover_height"] = value
	}
	if req.FontScale != nil {
		value, err := normalizeFuxiHallNumber(req.FontScale, 0, 12, 20, "字体大小")
		if err != nil {
			return nil, err
		}
		updates["font_scale"] = value
	}
	if req.Visible != nil {
		updates["visible"] = *req.Visible
	}

	if len(updates) == 0 {
		return card, nil
	}

	if err := s.repo.UpdateCardFields(id, updates); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUserVisibleError("卡片不存在")
		}
		return nil, err
	}

	return s.repo.GetCardByID(id)
}

func (s *FuxiHallService) DeleteCard(id uint) error {
	if err := s.repo.DeleteCard(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewUserVisibleError("卡片不存在")
		}
		return err
	}
	return nil
}

func (s *FuxiHallService) ReorderCards(req *FuxiHallReorderRequest) error {
	pageKey, err := normalizeFuxiHallPageKey(req.PageKey)
	if err != nil {
		return err
	}
	if _, err := s.ensurePage(pageKey); err != nil {
		return err
	}
	if len(req.OrderedIDs) == 0 {
		return NewUserVisibleError("排序列表不能为空")
	}

	seen := make(map[uint]struct{}, len(req.OrderedIDs))
	for _, id := range req.OrderedIDs {
		if id == 0 {
			return NewUserVisibleError("排序列表包含无效卡片 ID")
		}
		if _, exists := seen[id]; exists {
			return NewUserVisibleError("排序列表存在重复卡片 ID")
		}
		seen[id] = struct{}{}
	}

	count, err := s.repo.CountCardsByPageAndIDs(pageKey, req.OrderedIDs)
	if err != nil {
		return err
	}
	if count != int64(len(req.OrderedIDs)) {
		return NewUserVisibleError("排序列表包含不存在或不属于当前页面的卡片")
	}

	return s.repo.ReorderCards(pageKey, req.OrderedIDs)
}

func normalizeFuxiHallPageKey(pageKey string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(pageKey))
	if _, exists := fuxiHallAllowedPageKeys[normalized]; !exists {
		return "", NewUserVisibleError("page_key 仅支持 leadership 或 contributors")
	}
	return normalized, nil
}

func (s *FuxiHallService) ensurePage(pageKey string) (*model.FuxiHallPage, error) {
	page, err := s.repo.GetPageByKey(pageKey)
	if err != nil {
		return nil, err
	}
	if page != nil {
		return page, nil
	}

	def := &model.FuxiHallPage{
		PageKey:         pageKey,
		Title:           fuxiHallDefaultPageTitles[pageKey],
		Subtitle:        "",
		DescriptionHTML: "",
	}
	if err := s.repo.UpsertPage(def); err != nil {
		return nil, err
	}
	return def, nil
}

func normalizeFuxiHallStylePreset(input string, useDefault bool) (string, error) {
	value := strings.TrimSpace(strings.ToLower(input))
	if value == "" && useDefault {
		return "classic", nil
	}
	if _, exists := fuxiHallStylePresets[value]; !exists {
		return "", NewUserVisibleError("卡片风格不在允许范围内")
	}
	return value, nil
}

func normalizeFuxiHallColor(input string, defaultColor string, label string) (string, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		if defaultColor != "" {
			return defaultColor, nil
		}
		return "", NewUserVisibleError(label + "不能为空")
	}
	if !fuxiHallHexColorPattern.MatchString(value) {
		return "", NewUserVisibleError(label + "必须是十六进制颜色值")
	}
	return value, nil
}

func normalizeFuxiHallEnum(input string, defaultValue string, allowed map[string]struct{}, label string) (string, error) {
	value := strings.TrimSpace(strings.ToLower(input))
	if value == "" {
		if defaultValue != "" {
			return defaultValue, nil
		}
		return "", NewUserVisibleError(label + "不能为空")
	}
	if _, exists := allowed[value]; !exists {
		return "", NewUserVisibleError(label + "不在允许范围内")
	}
	return value, nil
}

func normalizeFuxiHallNumber(input *int, defaultValue int, minValue int, maxValue int, label string) (int, error) {
	if input == nil {
		return defaultValue, nil
	}
	if *input < minValue || *input > maxValue {
		return 0, NewUserVisibleError(label + "超出允许范围")
	}
	return *input, nil
}
