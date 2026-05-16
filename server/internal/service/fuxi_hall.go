package service

import (
	"amiya-eden/config"
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/pkg/eve/esi"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

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
	fuxiHallAvatarShapes = map[string]struct{}{
		"circle":  {},
		"rounded": {},
		"square":  {},
	}
)

// FuxiHallService 伏羲大厅业务层
type FuxiHallService struct {
	repo               *repository.FuxiHallRepository
	resolveCharacterID func(ctx context.Context, characterName string) (int64, error)
	auditSvc           *AuditService
}

func NewFuxiHallService() *FuxiHallService {
	return &FuxiHallService{
		repo:               repository.NewFuxiHallRepository(),
		resolveCharacterID: resolveCharacterIDByNameFromESI,
		auditSvc:           NewAuditService(),
	}
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
	PageKey           string   `json:"page_key"`
	Nickname          string   `json:"nickname"`
	MainCharacterName string   `json:"main_character_name"`
	TitleTags         []string `json:"title_tags"`
	DescriptionHTML   string   `json:"description_html"`
	AccentColor       string   `json:"accent_color"`
	AvatarShape       string   `json:"avatar_shape"`
	FontScale         *int     `json:"font_scale"`
	Visible           *bool    `json:"visible"`
}

type FuxiHallUpdateCardRequest struct {
	Nickname          *string   `json:"nickname"`
	MainCharacterName *string   `json:"main_character_name"`
	TitleTags         *[]string `json:"title_tags"`
	DescriptionHTML   *string   `json:"description_html"`
	AccentColor       *string   `json:"accent_color"`
	AvatarShape       *string   `json:"avatar_shape"`
	FontScale         *int      `json:"font_scale"`
	Visible           *bool     `json:"visible"`
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

func (s *FuxiHallService) UpdatePageConfig(operatorID uint, pageKey string, req *FuxiHallUpdatePageRequest) (*model.FuxiHallPage, error) {
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

func (s *FuxiHallService) CreateCard(operatorID uint, req *FuxiHallCreateCardRequest) (*model.FuxiHallCard, error) {
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
	mainCharacterID, err := s.resolveCharacterID(context.Background(), mainCharacterName)
	if err != nil {
		return nil, err
	}
	titleTags, err := normalizeFuxiHallTitleTags(req.TitleTags)
	if err != nil {
		return nil, err
	}
	if len(titleTags) == 0 {
		return nil, NewUserVisibleError("头衔标签不能为空")
	}

	accentColor, err := normalizeFuxiHallColor(req.AccentColor, "#3b82f6", "强调色")
	if err != nil {
		return nil, err
	}
	avatarShape, err := normalizeFuxiHallEnum(req.AvatarShape, "circle", fuxiHallAvatarShapes, "头像形状")
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
		MainCharacterID:   mainCharacterID,
		MainCharacterName: mainCharacterName,
		TitleTags:         titleTags,
		DescriptionHTML:   sanitizeRichTextHTML(req.DescriptionHTML),
		AccentColor:       accentColor,
		AvatarShape:       avatarShape,
		FontScale:         fontScale,
		Visible:           visible,
		SortOrder:         maxSortOrder + 1,
	}
	if err := s.repo.CreateCard(card); err != nil {
		return nil, err
	}
	s.recordFuxiCardAudit("fuxi_card_create", operatorID, card.ID, model.AuditResultSuccess, map[string]any{"page_key": card.PageKey, "nickname": card.Nickname})
	return card, nil
}

func (s *FuxiHallService) UpdateCard(operatorID, id uint, req *FuxiHallUpdateCardRequest) (*model.FuxiHallCard, error) {
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
		resolvedID, err := s.resolveCharacterID(context.Background(), value)
		if err != nil {
			return nil, err
		}
		updates["main_character_name"] = value
		updates["main_character_id"] = resolvedID
	}
	if req.TitleTags != nil {
		value, err := normalizeFuxiHallTitleTags(*req.TitleTags)
		if err != nil {
			return nil, err
		}
		if len(value) == 0 {
			return nil, NewUserVisibleError("头衔标签不能为空")
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		updates["title_tags"] = string(raw)
	}
	if req.DescriptionHTML != nil {
		updates["description_html"] = sanitizeRichTextHTML(*req.DescriptionHTML)
	}
	if req.AccentColor != nil {
		value, err := normalizeFuxiHallColor(*req.AccentColor, "", "强调色")
		if err != nil {
			return nil, err
		}
		updates["accent_color"] = value
	}
	if req.AvatarShape != nil {
		value, err := normalizeFuxiHallEnum(*req.AvatarShape, "", fuxiHallAvatarShapes, "头像形状")
		if err != nil {
			return nil, err
		}
		updates["avatar_shape"] = value
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

	updated, err := s.repo.GetCardByID(id)
	if err != nil {
		return nil, err
	}
	s.recordFuxiCardAudit("fuxi_card_update", operatorID, id, model.AuditResultSuccess, map[string]any{"nickname": updated.Nickname})
	return updated, nil
}

func (s *FuxiHallService) DeleteCard(operatorID, id uint) error {
	if err := s.repo.DeleteCard(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewUserVisibleError("卡片不存在")
		}
		return err
	}
	s.recordFuxiCardAudit("fuxi_card_delete", operatorID, id, model.AuditResultSuccess, map[string]any{"deleted_card_id": id})
	return nil
}

func (s *FuxiHallService) ReorderCards(operatorID uint, req *FuxiHallReorderRequest) error {
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

	if err := s.repo.ReorderCards(pageKey, req.OrderedIDs); err != nil {
		return err
	}
	s.recordFuxiAudit("fuxi_card_sort", operatorID, pageKey, model.AuditResultSuccess, map[string]any{"ordered_ids": req.OrderedIDs})
	return nil
}

func (s *FuxiHallService) recordFuxiAudit(action string, actorID uint, resourceID string, result string, details map[string]any) {
	if s.auditSvc == nil {
		return
	}
	_ = s.auditSvc.RecordEvent(context.Background(), AuditRecordInput{
		Category:     "content_admin",
		Action:       action,
		ActorUserID:  actorID,
		ResourceType: "fuxi_hall_page",
		ResourceID:   resourceID,
		Result:       result,
		Details:      details,
	})
}

func (s *FuxiHallService) recordFuxiCardAudit(action string, actorID, cardID uint, result string, details map[string]any) {
	if s.auditSvc == nil {
		return
	}
	_ = s.auditSvc.RecordEvent(context.Background(), AuditRecordInput{
		Category:     "content_admin",
		Action:       action,
		ActorUserID:  actorID,
		ResourceType: "fuxi_hall_card",
		ResourceID:   fmt.Sprintf("%d", cardID),
		Result:       result,
		Details:      details,
	})
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

func normalizeFuxiHallTitleTags(input []string) ([]string, error) {
	if len(input) == 0 {
		return []string{}, nil
	}

	normalized := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, raw := range input {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if utf8.RuneCountInString(tag) > 32 {
			return nil, NewUserVisibleError("头衔标签长度不能超过 32 个字符")
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
		if len(normalized) > 12 {
			return nil, NewUserVisibleError("头衔标签数量不能超过 12 个")
		}
	}

	return normalized, nil
}

func resolveCharacterIDByNameFromESI(ctx context.Context, characterName string) (int64, error) {
	baseURL := config.DefaultESIBaseURL
	apiPrefix := config.DefaultESIAPIPrefix
	if global.Config != nil {
		if value := strings.TrimSpace(global.Config.EveSSO.ESIBaseURL); value != "" {
			baseURL = strings.TrimRight(value, "/")
		}
		if value := strings.TrimSpace(global.Config.EveSSO.ESIAPIPrefix); value != "" {
			apiPrefix = strings.TrimRight(value, "/")
		}
	}
	client := esi.NewClientWithConfig(baseURL, apiPrefix)

	path := "/universe/ids/"
	var payload struct {
		Characters []struct {
			ID int64 `json:"id"`
		} `json:"characters"`
	}
	if err := client.PostJSON(ctx, path, "", []string{characterName}, &payload); err != nil {
		return 0, NewUserVisibleError(fmt.Sprintf("ESI 查询失败: %v", err))
	}
	if len(payload.Characters) != 1 || payload.Characters[0].ID <= 0 {
		return 0, NewUserVisibleError("未找到精确匹配角色")
	}
	return payload.Characters[0].ID, nil
}
