package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func useFuxiHallServiceTestDB(t *testing.T) {
	t.Helper()

	db := newServiceTestDB(t, "fuxi_hall_svc",
		&model.FuxiHallPage{},
		&model.FuxiHallCard{},
		&model.User{},
		&model.EveCharacter{},
		&model.Fleet{},
		&model.WelfareApplication{},
		&model.ShopOrder{},
		&model.SrpApplication{},
		&model.AdmissionApplication{},
		&model.MentorMenteeRelationship{},
		&model.NewbroRecruitment{},
		&model.NewbroRecruitmentEntry{},
	)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
}

func newFuxiHallServiceWithResolver(resolver func(ctx context.Context, characterName string) (int64, error)) *FuxiHallService {
	svc := NewFuxiHallService()
	svc.resolveCharacterID = resolver
	return svc
}

func TestFuxiHallRejectsInvalidPageKey(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := NewFuxiHallService()

	if _, err := svc.GetPublicPage("unknown"); err == nil {
		t.Fatal("expected invalid page_key error")
	}
}

func TestFuxiHallCreateCardValidatesRequiredFields(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 1001, nil })

	_, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "",
		MainCharacterName: "Main",
		TitleTags:         []string{"Title"},
	})
	if err == nil {
		t.Fatal("expected nickname required error")
	}
}

func TestFuxiHallCreateCardSanitizesDescriptionHTML(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 1001, nil })

	card, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "Alpha",
		MainCharacterName: "Main",
		TitleTags:         []string{"Lead"},
		DescriptionHTML:   `<p>Hello</p><script>alert(1)</script><a href="javascript:evil()">x</a>`,
	})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if card.DescriptionHTML != `<p>Hello</p><a>x</a>` {
		t.Fatalf("unexpected sanitized html: %q", card.DescriptionHTML)
	}
}

func TestFuxiHallUpdateCardValidatesControlledStyle(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 1002, nil })

	card, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "Beta",
		MainCharacterName: "Main",
		TitleTags:         []string{"Builder"},
	})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	badColor := "rgba(1,2,3,0.5)"
	_, err = svc.UpdateCard(1, card.ID, []string{model.RoleAdmin}, &FuxiHallUpdateCardRequest{AccentColor: &badColor})
	if err == nil {
		t.Fatal("expected invalid color error")
	}
}

func TestFuxiHallCreateCardIgnoresDeprecatedStyleFieldsFromJSON(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 1003, nil })

	payload := []byte(`{
		"page_key":"hall-of-fame",
		"nickname":"Legacy",
		"main_character_name":"Legacy Main",
		"title_tags":["Legacy Title"],
		"style_preset":"aurora",
		"badge_tone":"dawn",
		"cover_height":220
	}`)
	var req FuxiHallCreateCardRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	card, err := svc.CreateCard(1, &req)
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if card.Nickname != "Legacy" {
		t.Fatalf("expected card to be created from supported fields, got %q", card.Nickname)
	}
}

func TestFuxiHallReorderCardsIsAtomic(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(_ context.Context, characterName string) (int64, error) {
		switch characterName {
		case "A":
			return 2001, nil
		case "B":
			return 2002, nil
		default:
			return 2999, nil
		}
	})

	cardA, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "A",
		MainCharacterName: "A",
		TitleTags:         []string{"A"},
	})
	if err != nil {
		t.Fatalf("CreateCard A: %v", err)
	}
	cardB, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "B",
		MainCharacterName: "B",
		TitleTags:         []string{"B"},
	})
	if err != nil {
		t.Fatalf("CreateCard B: %v", err)
	}

	err = svc.ReorderCards(1, &FuxiHallReorderRequest{
		PageKey:    "hall-of-fame",
		OrderedIDs: []uint{cardB.ID, 999999},
	})
	if err == nil {
		t.Fatal("expected reorder error for invalid id")
	}

	cards, err := svc.ListCards("hall-of-fame", false)
	if err != nil {
		t.Fatalf("ListCards: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}
	if cards[0].ID != cardA.ID || cards[0].SortOrder != 1 {
		t.Fatalf("expected first card unchanged after failed reorder, got id=%d order=%d", cards[0].ID, cards[0].SortOrder)
	}
	if cards[1].ID != cardB.ID || cards[1].SortOrder != 2 {
		t.Fatalf("expected second card unchanged after failed reorder, got id=%d order=%d", cards[1].ID, cards[1].SortOrder)
	}
}

func TestFuxiHallReorderRejectsDuplicateIDs(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 2003, nil })

	card, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "Dup",
		MainCharacterName: "Dup",
		TitleTags:         []string{"Dup"},
	})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	err = svc.ReorderCards(1, &FuxiHallReorderRequest{
		PageKey:    "hall-of-fame",
		OrderedIDs: []uint{card.ID, card.ID},
	})
	if err == nil {
		t.Fatal("expected duplicate id validation error")
	}
}

func TestFuxiHallCreateCardRequiresTitleTags(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 3001, nil })

	_, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:             "hall-of-fame",
		Nickname:            "NoTags",
		MainCharacterName:   "Main",
		IsCurrentManagement: true,
	})
	if err == nil {
		t.Fatal("expected title_tags required error")
	}
}

func TestFuxiHallCreateCardNormalizesTitleTags(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 3002, nil })

	card, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "Tags",
		MainCharacterName: "Main",
		TitleTags:         []string{"  指挥  ", "", "后勤", "指挥"},
	})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if len(card.TitleTags) != 2 || card.TitleTags[0] != "指挥" || card.TitleTags[1] != "后勤" {
		t.Fatalf("unexpected normalized title tags: %#v", card.TitleTags)
	}
}

func TestFuxiHallCreateCardValidatesTitleTagsLimits(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 3003, nil })

	longTag := strings.Repeat("长", 33)
	_, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "TooLong",
		MainCharacterName: "Main",
		TitleTags:         []string{longTag},
	})
	if err == nil {
		t.Fatal("expected title_tags single item length validation error")
	}

	tooMany := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13"}
	_, err = svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "TooMany",
		MainCharacterName: "Main",
		TitleTags:         tooMany,
	})
	if err == nil {
		t.Fatal("expected title_tags total count validation error")
	}
}

func TestFuxiHallUpdateCardReplacesTitleTags(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 3005, nil })

	card, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "Replace",
		MainCharacterName: "Main",
		TitleTags:         []string{"旧标签"},
	})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	nextTags := []string{"  新标签  ", "新标签", "战略"}
	updated, err := svc.UpdateCard(1, card.ID, []string{model.RoleAdmin}, &FuxiHallUpdateCardRequest{TitleTags: &nextTags})
	if err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}
	if len(updated.TitleTags) != 2 || updated.TitleTags[0] != "新标签" || updated.TitleTags[1] != "战略" {
		t.Fatalf("unexpected updated title tags: %#v", updated.TitleTags)
	}
}

func TestFuxiHallCreateCardResolvesMainCharacterIDFromName(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(_ context.Context, characterName string) (int64, error) {
		if characterName != "XiaomaiTX" {
			t.Fatalf("unexpected character name: %s", characterName)
		}
		return 123456789, nil
	})

	card, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "Alpha",
		MainCharacterName: "XiaomaiTX",
		TitleTags:         []string{"Lead"},
	})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if card.MainCharacterID != 123456789 {
		t.Fatalf("expected resolved main_character_id 123456789, got %d", card.MainCharacterID)
	}
}

func TestFuxiHallCreateCardBlocksWhenCharacterResolveFails(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) {
		return 0, NewUserVisibleError("未找到精确匹配角色")
	})

	_, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "Alpha",
		MainCharacterName: "NoSuchName",
		TitleTags:         []string{"Lead"},
	})
	if err == nil {
		t.Fatal("expected create card to fail when resolver fails")
	}
}

func TestFuxiHallUpdateCardReResolvesCharacterIDWhenNameChanged(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(_ context.Context, characterName string) (int64, error) {
		if characterName == "Old Name" {
			return 4001, nil
		}
		if characterName == "New Name" {
			return 4002, nil
		}
		return 0, errors.New("unexpected name")
	})

	card, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{
		PageKey:           "hall-of-fame",
		Nickname:          "Gamma",
		MainCharacterName: "Old Name",
		TitleTags:         []string{"Tag"},
	})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if card.MainCharacterID != 4001 {
		t.Fatalf("expected initial id 4001, got %d", card.MainCharacterID)
	}

	newName := "New Name"
	updated, err := svc.UpdateCard(1, card.ID, []string{model.RoleAdmin}, &FuxiHallUpdateCardRequest{MainCharacterName: &newName})
	if err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}
	if updated.MainCharacterID != 4002 {
		t.Fatalf("expected updated id 4002, got %d", updated.MainCharacterID)
	}
}

func TestFuxiHallCardJSONContainsTitleTags(t *testing.T) {
	card := model.FuxiHallCard{
		Nickname:  "JSON",
		TitleTags: []string{"A", "B"},
	}
	raw, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("marshal card: %v", err)
	}
	if !strings.Contains(string(raw), `"title_tags":["A","B"]`) {
		t.Fatalf("expected title_tags in json, got: %s", string(raw))
	}
}

func TestFuxiHallListManageCardsUsesCachedContributions(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := NewFuxiHallService()

	user := model.User{PrimaryCharacterID: 9001}
	if err := global.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := global.DB.Create(&model.EveCharacter{CharacterID: 9001, UserID: user.ID, CharacterName: "Leader"}).Error; err != nil {
		t.Fatalf("create eve character: %v", err)
	}
	now := time.Now().UTC()
	if err := global.DB.Create(&model.Fleet{
		ID:              "fleet-1",
		Title:           "Fleet",
		StartAt:         now,
		EndAt:           now.Add(-time.Hour),
		Importance:      model.FleetImportanceOther,
		FCUserID:        user.ID,
		FCCharacterID:   9001,
		FCCharacterName: "Leader",
		AutoSrpMode:     model.FleetAutoSrpDisabled,
	}).Error; err != nil {
		t.Fatalf("create fleet: %v", err)
	}
	if err := global.DB.Create(&model.WelfareApplication{
		WelfareID:     1,
		CharacterID:   9001,
		CharacterName: "Leader",
		Status:        model.WelfareAppStatusDelivered,
		ReviewedBy:    user.ID,
	}).Error; err != nil {
		t.Fatalf("create welfare application: %v", err)
	}
	reviewerID := user.ID
	if err := global.DB.Create(&model.ShopOrder{
		OrderNo:           "SO-001",
		UserID:            user.ID,
		ProductID:         1,
		ProductName:       "Product",
		ProductType:       model.ProductTypeNormal,
		Quantity:          1,
		UnitPrice:         1,
		TotalPrice:        1,
		Status:            model.OrderStatusDelivered,
		MainCharacterName: "Leader",
		ReviewedBy:        &reviewerID,
	}).Error; err != nil {
		t.Fatalf("create shop order: %v", err)
	}

	if err := global.DB.Create(&model.FuxiHallCard{
		PageKey:               "hall-of-fame",
		Nickname:              "Leader",
		MainCharacterID:       9001,
		MainCharacterName:     "Leader",
		TitleTags:             []string{"Admin"},
		AccentColor:           "#3b82f6",
		AvatarShape:           "circle",
		FontScale:             14,
		Visible:               true,
		SortOrder:             1,
		WelfareDeliveryOffset: 2,
	}).Error; err != nil {
		t.Fatalf("create leadership card: %v", err)
	}
	if err := global.DB.Create(&model.FuxiHallCard{
		PageKey:           "hall-of-fame",
		Nickname:          "Contributor",
		MainCharacterID:   9001,
		MainCharacterName: "Leader",
		TitleTags:         []string{"Contributor"},
		AccentColor:       "#3b82f6",
		AvatarShape:       "circle",
		FontScale:         14,
		Visible:           true,
		SortOrder:         1,
	}).Error; err != nil {
		t.Fatalf("create contributors card: %v", err)
	}

	if err := svc.RefreshContributions(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	leadershipCards, err := svc.ListManageCards("hall-of-fame")
	if err != nil {
		t.Fatalf("ListManageCards leadership: %v", err)
	}
	if len(leadershipCards) != 2 {
		t.Fatalf("expected 1 leadership card, got %d", len(leadershipCards))
	}
	if leadershipCards[0].Contributions.Fleets != 1 {
		t.Fatalf("fleet led count = %d, want 1: %+v", leadershipCards[0].Contributions.Fleets, leadershipCards[0])
	}
	if leadershipCards[0].Contributions.Welfare != 4 {
		t.Fatalf("welfare delivery count = %d, want 4", leadershipCards[0].Contributions.Welfare)
	}

	if leadershipCards[1].Contributions.Fleets != 1 || leadershipCards[1].Contributions.Welfare != 2 {
		t.Fatalf("unified contributor statistics: %+v", leadershipCards[1])
	}

}

func TestFuxiHallAdminCanAdjustEveryOffset(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := newFuxiHallServiceWithResolver(func(context.Context, string) (int64, error) { return 9010, nil })
	card, err := svc.CreateCard(1, &FuxiHallCreateCardRequest{PageKey: "hall-of-fame", Nickname: "Offset", MainCharacterName: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	offsets := model.FuxiHallContributions{Fleets: 2, Welfare: 3, SRP: 4, Recruitment: 5, Mentoring: 6}
	if _, err = svc.UpdateCard(1, card.ID, []string{model.RoleAdmin}, &FuxiHallUpdateCardRequest{ContributionOffsets: &offsets}); err != nil {
		t.Fatal(err)
	}
	hall, err := svc.PublicHall(1)
	if err != nil {
		t.Fatal(err)
	}
	if hall.Cards[0].Contributions != offsets {
		t.Fatalf("offset totals: %+v", hall.Cards[0].Contributions)
	}
	offsets.SRP = -1
	if _, err = svc.UpdateCard(1, card.ID, []string{model.RoleAdmin}, &FuxiHallUpdateCardRequest{ContributionOffsets: &offsets}); err == nil {
		t.Fatal("negative offset accepted")
	}
}
