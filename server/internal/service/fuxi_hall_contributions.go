package service

import (
	"amiya-eden/internal/model"
	"context"
	"sync"
	"time"
)

var hallContributionRefresh sync.Mutex

// ErrFuxiHallRefreshRunning indicates a contribution snapshot refresh is already in progress.
var ErrFuxiHallRefreshRunning = NewUserVisibleError("fuxiHall.errors.refreshRunning")

func validateHallOffsets(v model.FuxiHallContributions) error {
	for _, n := range []int64{v.Fleets, v.Welfare, v.SRP, v.Recruitment, v.Mentoring} {
		if n < 0 || n > 1_000_000_000 {
			return NewUserVisibleError("fuxiHall.errors.offsets")
		}
	}
	return nil
}

func hallOffsetFields(v model.FuxiHallContributions) map[string]interface{} {
	return map[string]interface{}{"fleet_led_offset": v.Fleets, "welfare_delivery_offset": v.Welfare, "srp_offset": v.SRP, "recruitment_offset": v.Recruitment, "mentoring_offset": v.Mentoring}
}

// Both the daily task and manual refresh share the lock and atomic snapshot write.
// Aggregate queries run outside the hall page row lock so concurrent card/page edits do not block.
// Failed/cancelled queries leave the previous complete snapshot intact.
func (s *FuxiHallService) RefreshContributions(ctx context.Context, operatorID uint) error {
	if !hallContributionRefresh.TryLock() {
		return ErrFuxiHallRefreshRunning
	}
	defer hallContributionRefresh.Unlock()
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	if _, err := scoped.ensurePage(string(model.FuxiHallPageFame)); err != nil {
		return err
	}
	cards, err := scoped.repo.ListCardsByPage(string(model.FuxiHallPageFame), false)
	if err != nil {
		return err
	}
	mapping, err := scoped.resolveFuxiHallUsers(collectFuxiHallCharacterIDs(cards))
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	counts, err := scoped.repo.ContributionCounts(collectFuxiHallUserIDs(mapping), now)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	computedByCardID := make(map[uint]struct {
		characterID int64
		counts      model.FuxiHallContributions
	}, len(cards))
	for _, card := range cards {
		computedByCardID[card.ID] = struct {
			characterID int64
			counts      model.FuxiHallContributions
		}{
			characterID: card.MainCharacterID,
			counts:      counts[mapping[card.MainCharacterID]],
		}
	}
	err = scoped.withHallLock(func(tx *FuxiHallService, page *model.FuxiHallPage) error {
		currentCards, err := tx.repo.ListCardsByPage(page.PageKey, false)
		if err != nil {
			return err
		}
		for i := range currentCards {
			computed, ok := computedByCardID[currentCards[i].ID]
			if !ok || computed.characterID != currentCards[i].MainCharacterID {
				continue
			}
			currentCards[i].ContributionCounts = computed.counts
			currentCards[i].ContributionsUpdatedAt = &now
			if err = tx.repo.SaveContributionSnapshot(&currentCards[i]); err != nil {
				return err
			}
		}
		return ctx.Err()
	})
	if err == nil && operatorID > 0 {
		_ = s.auditSvc.RecordEvent(ctx, AuditRecordInput{Category: "content_admin", Action: "fuxi_contributions_refresh", ActorUserID: operatorID, ResourceType: "fuxi_hall_page", ResourceID: string(model.FuxiHallPageFame), Result: model.AuditResultSuccess})
	}
	return err
}
