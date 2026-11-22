package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFuxiHallOwnProfileOwnershipAndTenure(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 9001, nil })
	days := 800
	char := model.EveCharacter{CharacterID: 9001, UserID: 42, FuxiLegionTenureDays: &days}
	if err := global.DB.Create(&char).Error; err != nil {
		t.Fatal(err)
	}
	card, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:             "hall-of-fame",
		Nickname:            "Before",
		MainCharacterName:   "Main",
		TitleTags:           []string{"Director"},
		IsCurrentManagement: true,
		ContributionOffsets: &model.FuxiHallContributions{Fleets: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if card.Contributions.Fleets != 5 || card.TenureMonths == nil || *card.TenureMonths != 26 || card.CanEdit {
		t.Fatalf("CreateCard should return enriched card, got: %+v", card)
	}
	ownerCard, err := svc.UpdateCard(42, card.ID, nil, &FuxiHallUpdateCardRequest{
		ContributionOffsets: &model.FuxiHallContributions{Fleets: 7},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ownerCard.Contributions.Fleets != 7 || ownerCard.TenureMonths == nil || *ownerCard.TenureMonths != 26 || !ownerCard.CanEdit {
		t.Fatalf("UpdateCard should return enriched card, got: %+v", ownerCard)
	}
	for _, viewer := range []uint{42, 43} {
		hall, err := svc.PublicHall(viewer)
		if err != nil {
			t.Fatal(err)
		}
		got := hall.Cards[0]
		if got.CanEdit != (viewer == 42) || got.TenureMonths == nil || *got.TenureMonths != 26 {
			t.Fatalf("unexpected viewer enrichment: %+v", got)
		}
	}
	var req FuxiHallOwnCardRequest
	if err := json.Unmarshal([]byte(`{"nickname":" After ","description_html":"<p>Bio</p><script>evil()</script>","floor_key":"top","visible":false,"is_current_management":false,"main_character_id":123}`), &req); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.UpdateOwnCard(43, card.ID, &req); err == nil {
		t.Fatal("cross-user edit allowed")
	}
	saved, err := svc.UpdateOwnCard(42, card.ID, &req)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Nickname != "After" || saved.DescriptionHTML != "<p>Bio</p>" || saved.FloorKey != "first" || !saved.Visible || !saved.IsCurrentManagement || saved.MainCharacterID != 9001 {
		t.Fatalf("unexpected self edit: %+v", saved)
	}
	if saved.Contributions.Fleets != 7 || !saved.CanEdit || saved.TenureMonths == nil || *saved.TenureMonths != 26 {
		t.Fatalf("UpdateOwnCard should return enriched card, got: %+v", saved)
	}
	if err := global.DB.Model(&model.EveCharacter{}).Where("character_id = ?", 9001).Update("user_id", 43).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = svc.UpdateOwnCard(42, card.ID, &req); err == nil {
		t.Fatal("previous owner can edit after rebinding")
	}
	if err := global.DB.Model(&model.EveCharacter{}).Where("character_id = ?", 9001).Update("fuxi_legion_tenure_days", 0).Error; err != nil {
		t.Fatal(err)
	}
	zeroHall, err := svc.PublicHall(43)
	if err != nil {
		t.Fatal(err)
	}
	if zeroHall.Cards[0].TenureMonths == nil || *zeroHall.Cards[0].TenureMonths != 0 {
		t.Fatal("known zero must be distinct from missing tenure")
	}
	if err := global.DB.Delete(&char).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = svc.UpdateOwnCard(43, card.ID, &req); err == nil {
		t.Fatal("unbound character edit allowed")
	}
	hall, err := svc.PublicHall(43)
	if err != nil {
		t.Fatal(err)
	}
	if hall.Cards[0].CanEdit || hall.Cards[0].TenureMonths != nil {
		t.Fatal("missing record should have no edit permission or tenure")
	}
}

func TestFuxiHallFloorCapacityAndHiddenCards(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 9002, nil })
	floors := []model.FuxiHallFloor{{Key: "first", Capacity: 1}}
	if _, err := svc.UpdatePageConfig(1, "hall-of-fame", &FuxiHallUpdatePageRequest{Floors: &floors}); err != nil {
		t.Fatal(err)
	}
	req := FuxiHallCreateCardRequest{PageKey: "hall-of-fame", Nickname: "Card", MainCharacterName: "Main"}
	card, err := svc.CreateCard(1, &req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.CreateCard(1, &req); err == nil {
		t.Fatal("overfilled floor")
	}
	hidden := false
	req.Visible = &hidden
	other, err := svc.CreateCard(1, &req)
	if err != nil {
		t.Fatal(err)
	}
	if other.Visible {
		t.Fatal("hidden card became visible")
	}
	visible := true
	if _, err = svc.UpdateCard(1, other.ID, nil, &FuxiHallUpdateCardRequest{Visible: &visible}); err == nil {
		t.Fatal("visibility bypassed capacity")
	}
	removed := []model.FuxiHallFloor{{Key: "top", Capacity: 3}}
	if _, err = svc.UpdatePageConfig(1, "hall-of-fame", &FuxiHallUpdatePageRequest{Floors: &removed}); err == nil {
		t.Fatal("removed occupied floor")
	}
	floors[0].Capacity = 3
	if _, err = svc.UpdatePageConfig(1, "hall-of-fame", &FuxiHallUpdatePageRequest{Floors: &floors}); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.UpdateCard(1, other.ID, nil, &FuxiHallUpdateCardRequest{Visible: &visible}); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.UpdateCard(1, card.ID, nil, &FuxiHallUpdateCardRequest{IsCurrentManagement: &visible}); err == nil {
		t.Fatal("manager without title accepted")
	}
	if err := validateHallProfile(strings.Repeat("x", 257), ""); err == nil {
		t.Fatal("oversized nickname accepted")
	}
}

func TestFuxiHallResolutionPrecedesTransaction(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	db, err := global.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	svc := newFuxiHallServiceWithResolver(func(ctx context.Context, name string) (int64, error) {
		calls++
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("resolver context has no deadline")
		}
		if db.Stats().InUse != 0 {
			t.Fatal("ESI lookup holds a database connection")
		}
		return int64(9000 + calls), nil
	})
	card, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{PageKey: "hall-of-fame", Nickname: "Member", MainCharacterName: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	name := "Replacement"
	updated, err := svc.UpdateCard(1, card.ID, nil, &FuxiHallUpdateCardRequest{MainCharacterName: &name})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || updated.MainCharacterID != 9002 {
		t.Fatalf("unexpected resolution: calls=%d card=%+v", calls, updated)
	}
}

func TestFuxiHallUpdateMissingCard(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := NewFuxiHallService()
	_, err := svc.UpdateCard(1, 1234, nil, &FuxiHallUpdateCardRequest{})
	if err == nil || !IsUserVisibleError(err) || err.Error() != "fuxiHall.errors.cardNotFound" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFuxiHallCardAuditWaitsForHallLockRelease(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	if err := global.DB.AutoMigrate(&model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := global.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	// With one connection, an audit written through the pool while the hall transaction holds
	// that connection would block forever.
	sqlDB.SetMaxOpenConns(1)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 9003, nil })
	done := make(chan error, 1)
	go func() {
		card, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{PageKey: "hall-of-fame", Nickname: "Audited", MainCharacterName: "Main"})
		if err == nil {
			nickname := "Audited again"
			_, err = svc.UpdateCard(1, card.ID, nil, &FuxiHallUpdateCardRequest{Nickname: &nickname})
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("card write deadlocked on audit while holding the hall lock")
	}
	var actions []string
	if err := global.DB.Model(&model.AuditEvent{}).Order("id").Pluck("action", &actions).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Join(actions, ",") != "fuxi_card_create,fuxi_card_update" {
		t.Fatalf("unexpected audit actions: %v", actions)
	}
}

func TestFuxiHallSavedCardSurvivesEnrichmentFailure(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 9004, nil })
	if err := global.DB.Migrator().DropTable(&model.EveCharacter{}); err != nil {
		t.Fatal(err)
	}
	card, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{PageKey: "hall-of-fame", Nickname: "Saved", MainCharacterName: "Main"})
	if err != nil || card == nil || card.ID == 0 {
		t.Fatalf("committed create must not report failure: card=%+v err=%v", card, err)
	}
	nickname := "Saved again"
	updated, err := svc.UpdateCard(1, card.ID, nil, &FuxiHallUpdateCardRequest{Nickname: &nickname})
	if err != nil || updated.Nickname != nickname {
		t.Fatalf("committed update must not report failure: card=%+v err=%v", updated, err)
	}
}
