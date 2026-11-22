package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/pkg/eve/esi"
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type FuxiHallOwnCardRequest struct {
	Nickname        string `json:"nickname"`
	DescriptionHTML string `json:"description_html"`
}

var hallFloorKey = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

func validateHallFloors(floors []model.FuxiHallFloor, cards []model.FuxiHallCard) error {
	if len(floors) == 0 || len(floors) > 12 {
		return NewUserVisibleError("fuxiHall.errors.floors")
	}
	capacities := map[string]int{}
	for _, floor := range floors {
		if strings.TrimSpace(floor.Name) == "" && floor.Key != "top" && floor.Key != "second" && floor.Key != "first" {
			return NewUserVisibleError("fuxiHall.errors.floors")
		}
		if !hallFloorKey.MatchString(floor.Key) || utf8.RuneCountInString(floor.Name) > 64 || floor.Capacity < 1 || floor.Capacity > 10000 {
			return NewUserVisibleError("fuxiHall.errors.floors")
		}
		if _, found := capacities[floor.Key]; found {
			return NewUserVisibleError("fuxiHall.errors.floors")
		}
		capacities[floor.Key] = floor.Capacity
	}
	counts := map[string]int{}
	for _, card := range cards {
		if _, found := capacities[card.FloorKey]; !found {
			return NewUserVisibleError("fuxiHall.errors.occupiedFloor")
		}
		if card.Visible {
			counts[card.FloorKey]++
		}
	}
	for key, count := range counts {
		if count > capacities[key] {
			return NewUserVisibleError("fuxiHall.errors.capacity")
		}
	}
	return nil
}

// Capacity and floor membership mutations share this lock so simultaneous edits cannot
// overfill floors or move a card into a floor being removed.
func (s *FuxiHallService) withHallLock(fn func(*FuxiHallService, *model.FuxiHallPage) error) error {
	if _, err := s.ensurePage(string(model.FuxiHallPageFame)); err != nil {
		return err
	}
	return s.repo.Transaction(func(repo *repository.FuxiHallRepository) error {
		page, err := repo.LockPage(string(model.FuxiHallPageFame))
		if err != nil {
			return err
		}
		scoped := *s
		scoped.repo = repo
		return fn(&scoped, page)
	})
}

func (s *FuxiHallService) UpdatePageConfig(operatorID uint, pageKey string, req *FuxiHallUpdatePageRequest) (*model.FuxiHallPage, error) {
	if _, err := normalizeFuxiHallPageKey(pageKey); err != nil {
		return nil, err
	}
	var result *model.FuxiHallPage
	err := s.withHallLock(func(scoped *FuxiHallService, page *model.FuxiHallPage) error {
		if req.Floors != nil {
			cards, err := scoped.repo.ListCardsByPage(page.PageKey, false)
			if err != nil {
				return err
			}
			if err := validateHallFloors(*req.Floors, cards); err != nil {
				return err
			}
		}
		var err error
		result, err = scoped.updatePageConfig(operatorID, pageKey, req)
		return err
	})
	return result, err
}

func validateHallProfile(nickname, html string) error {
	if strings.TrimSpace(nickname) == "" || utf8.RuneCountInString(nickname) > 256 || len(html) > 64000 {
		return NewUserVisibleError("fuxiHall.errors.profile")
	}
	return nil
}

func validateHallCard(page *model.FuxiHallPage, cards []model.FuxiHallCard, candidate model.FuxiHallCard) error {
	if candidate.IsCurrentManagement && len(candidate.TitleTags) == 0 {
		return NewUserVisibleError("fuxiHall.errors.titleRequired")
	}
	next := make([]model.FuxiHallCard, 0, len(cards)+1)
	for _, card := range cards {
		if card.ID != candidate.ID {
			next = append(next, card)
		}
	}
	next = append(next, candidate)
	return validateHallFloors(page.Floors, next)
}

func (s *FuxiHallService) CreateCard(operatorID uint, req *FuxiHallCreateCardRequest) (*model.FuxiHallCard, error) {
	if _, err := normalizeFuxiHallPageKey(req.PageKey); err != nil {
		return nil, err
	}
	if err := validateHallProfile(req.Nickname, req.DescriptionHTML); err != nil {
		return nil, err
	}
	input := *req
	if input.FloorKey == "" {
		input.FloorKey = "first"
	}
	characterID, err := s.resolveHallCharacterID(req.MainCharacterName)
	if err != nil {
		return nil, err
	}
	var result *model.FuxiHallCard
	err = s.withHallLock(func(scoped *FuxiHallService, page *model.FuxiHallPage) error {
		cards, err := scoped.repo.ListCardsByPage(page.PageKey, false)
		if err != nil {
			return err
		}
		tags, err := normalizeFuxiHallTitleTags(input.TitleTags)
		if err != nil {
			return err
		}
		visible := input.Visible == nil || *input.Visible
		if err := validateHallCard(page, cards, model.FuxiHallCard{FloorKey: input.FloorKey, Visible: visible, IsCurrentManagement: input.IsCurrentManagement, TitleTags: tags}); err != nil {
			return err
		}
		result, err = scoped.createCard(operatorID, &input, characterID)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.recordFuxiCardAudit("fuxi_card_create", operatorID, result.ID, model.AuditResultSuccess, map[string]any{"page_key": result.PageKey, "nickname": result.Nickname})
	s.enrichSavedCard(operatorID, result)
	return result, nil
}

func (s *FuxiHallService) UpdateCard(operatorID, id uint, roles []string, req *FuxiHallUpdateCardRequest) (*model.FuxiHallCard, error) {
	var characterID int64
	if req.MainCharacterName != nil {
		var err error
		characterID, err = s.resolveHallCharacterID(*req.MainCharacterName)
		if err != nil {
			return nil, err
		}
	}
	var result *model.FuxiHallCard
	err := s.withHallLock(func(scoped *FuxiHallService, page *model.FuxiHallPage) error {
		candidate, err := scoped.repo.GetCardByID(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewUserVisibleError("fuxiHall.errors.cardNotFound")
		}
		if err != nil {
			return err
		}
		if candidate.PageKey != page.PageKey {
			return NewUserVisibleError("fuxiHall.errors.invalid")
		}
		if req.Nickname != nil {
			candidate.Nickname = *req.Nickname
		}
		if req.DescriptionHTML != nil {
			candidate.DescriptionHTML = *req.DescriptionHTML
		}
		if err := validateHallProfile(candidate.Nickname, candidate.DescriptionHTML); err != nil {
			return err
		}
		if req.FloorKey != nil {
			candidate.FloorKey = *req.FloorKey
		}
		if req.Visible != nil {
			candidate.Visible = *req.Visible
		}
		if req.IsCurrentManagement != nil {
			candidate.IsCurrentManagement = *req.IsCurrentManagement
		}
		if req.TitleTags != nil {
			candidate.TitleTags, err = normalizeFuxiHallTitleTags(*req.TitleTags)
			if err != nil {
				return err
			}
		}
		cards, err := scoped.repo.ListCardsByPage(page.PageKey, false)
		if err != nil {
			return err
		}
		if err := validateHallCard(page, cards, *candidate); err != nil {
			return err
		}
		result, err = scoped.updateCard(operatorID, id, roles, req, characterID)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.recordFuxiCardAudit("fuxi_card_update", operatorID, id, model.AuditResultSuccess, map[string]any{"nickname": result.Nickname})
	s.enrichSavedCard(operatorID, result)
	return result, nil
}

func (s *FuxiHallService) enrichCards(viewerID uint, cards []model.FuxiHallCard) error {
	if len(cards) == 0 {
		return nil
	}
	chars, err := s.repo.Characters(collectFuxiHallCharacterIDs(cards))
	if err != nil {
		return err
	}
	byID := make(map[int64]model.EveCharacter, len(chars))
	for _, char := range chars {
		byID[char.CharacterID] = char
	}
	for i := range cards {
		card := &cards[i]
		card.EnrichContributions()
		if char, found := byID[card.MainCharacterID]; found {
			card.CanEdit = viewerID != 0 && char.UserID == viewerID
			if char.FuxiLegionTenureDays != nil && *char.FuxiLegionTenureDays >= 0 {
				months := *char.FuxiLegionTenureDays * 12 / 365
				card.TenureMonths = &months
			}
		}
	}
	return nil
}

func (s *FuxiHallService) enrichCard(viewerID uint, card *model.FuxiHallCard) error {
	if card == nil {
		return nil
	}
	cards := []model.FuxiHallCard{*card}
	if err := s.enrichCards(viewerID, cards); err != nil {
		return err
	}
	*card = cards[0]
	return nil
}

// Writes are already committed when this runs; an enrichment failure must not report the
// save as failed (clients would retry and duplicate creates), so it degrades to the plain card.
func (s *FuxiHallService) enrichSavedCard(viewerID uint, card *model.FuxiHallCard) {
	if err := s.enrichCard(viewerID, card); err != nil {
		fuxiHallLogger().Warn("enrich saved fuxi hall card failed", zap.Uint("card_id", card.ID), zap.Error(err))
	}
}

func fuxiHallLogger() *zap.Logger {
	if global.Logger != nil {
		return global.Logger
	}
	return zap.NewNop()
}

func (s *FuxiHallService) PublicHall(viewerID uint) (*FuxiHallPublicPageResponse, error) {
	result, err := s.GetPublicPage(string(model.FuxiHallPageFame))
	if err != nil {
		return nil, err
	}
	if err := s.enrichCards(viewerID, result.Cards); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *FuxiHallService) UpdateOwnCard(userID, id uint, req *FuxiHallOwnCardRequest) (*model.FuxiHallCard, error) {
	if err := validateHallProfile(req.Nickname, req.DescriptionHTML); err != nil {
		return nil, err
	}
	var result *model.FuxiHallCard
	err := s.withHallLock(func(scoped *FuxiHallService, page *model.FuxiHallPage) error {
		card, err := scoped.repo.GetCardByID(id)
		if err != nil {
			return NewUserVisibleError("fuxiHall.errors.denied")
		}
		owned, err := scoped.repo.OwnsCharacter(userID, card.MainCharacterID)
		if err != nil {
			return err
		}
		if !owned || card.PageKey != page.PageKey {
			return NewUserVisibleError("fuxiHall.errors.denied")
		}
		if err := scoped.repo.UpdateCardFields(id, map[string]any{"nickname": strings.TrimSpace(req.Nickname), "description_html": sanitizeRichTextHTML(req.DescriptionHTML)}); err != nil {
			return err
		}
		result, err = scoped.repo.GetCardByID(id)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.recordFuxiCardAudit("fuxi_card_update", userID, id, model.AuditResultSuccess, map[string]any{"self_edit": true})
	s.enrichSavedCard(userID, result)
	return result, nil
}

// Resolve ESI names before opening the transaction that serializes hall writes.
func (s *FuxiHallService) resolveHallCharacterID(name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, NewUserVisibleError("主角色名称不能为空")
	}
	ctx, cancel := context.WithTimeout(context.Background(), esi.DefaultTimeout)
	defer cancel()
	return s.resolveCharacterID(ctx, name)
}
