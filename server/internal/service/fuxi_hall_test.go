package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"encoding/json"
	"strings"
	"testing"
)

func useFuxiHallServiceTestDB(t *testing.T) {
	t.Helper()

	db := newServiceTestDB(t, "fuxi_hall_svc",
		&model.FuxiHallPage{},
		&model.FuxiHallCard{},
	)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
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
	svc := NewFuxiHallService()

	_, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "leadership",
		Nickname:          "",
		MainCharacterID:   1001,
		MainCharacterName: "Main",
		TitleTags:         []string{"Title"},
	})
	if err == nil {
		t.Fatal("expected nickname required error")
	}
}

func TestFuxiHallCreateCardSanitizesDescriptionHTML(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := NewFuxiHallService()

	card, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "leadership",
		Nickname:          "Alpha",
		MainCharacterID:   1001,
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
	svc := NewFuxiHallService()

	card, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "contributors",
		Nickname:          "Beta",
		MainCharacterID:   1002,
		MainCharacterName: "Main",
		TitleTags:         []string{"Builder"},
	})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	badColor := "rgba(1,2,3,0.5)"
	_, err = svc.UpdateCard(card.ID, &FuxiHallUpdateCardRequest{AccentColor: &badColor})
	if err == nil {
		t.Fatal("expected invalid color error")
	}
}

func TestFuxiHallCreateCardIgnoresDeprecatedStyleFieldsFromJSON(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := NewFuxiHallService()

	payload := []byte(`{
		"page_key":"leadership",
		"nickname":"Legacy",
		"main_character_id":1003,
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

	card, err := svc.CreateCard(&req)
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if card.Nickname != "Legacy" {
		t.Fatalf("expected card to be created from supported fields, got %q", card.Nickname)
	}
}

func TestFuxiHallReorderCardsIsAtomic(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := NewFuxiHallService()

	cardA, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "leadership",
		Nickname:          "A",
		MainCharacterID:   2001,
		MainCharacterName: "A",
		TitleTags:         []string{"A"},
	})
	if err != nil {
		t.Fatalf("CreateCard A: %v", err)
	}
	cardB, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "leadership",
		Nickname:          "B",
		MainCharacterID:   2002,
		MainCharacterName: "B",
		TitleTags:         []string{"B"},
	})
	if err != nil {
		t.Fatalf("CreateCard B: %v", err)
	}

	err = svc.ReorderCards(&FuxiHallReorderRequest{
		PageKey:    "leadership",
		OrderedIDs: []uint{cardB.ID, 999999},
	})
	if err == nil {
		t.Fatal("expected reorder error for invalid id")
	}

	cards, err := svc.ListCards("leadership", false)
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
	svc := NewFuxiHallService()

	card, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "leadership",
		Nickname:          "Dup",
		MainCharacterID:   2003,
		MainCharacterName: "Dup",
		TitleTags:         []string{"Dup"},
	})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	err = svc.ReorderCards(&FuxiHallReorderRequest{
		PageKey:    "leadership",
		OrderedIDs: []uint{card.ID, card.ID},
	})
	if err == nil {
		t.Fatal("expected duplicate id validation error")
	}
}

func TestFuxiHallCreateCardRequiresTitleTags(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := NewFuxiHallService()

	_, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "leadership",
		Nickname:          "NoTags",
		MainCharacterID:   3001,
		MainCharacterName: "Main",
	})
	if err == nil {
		t.Fatal("expected title_tags required error")
	}
}

func TestFuxiHallCreateCardNormalizesTitleTags(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := NewFuxiHallService()

	card, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "contributors",
		Nickname:          "Tags",
		MainCharacterID:   3002,
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
	svc := NewFuxiHallService()

	longTag := strings.Repeat("长", 33)
	_, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "contributors",
		Nickname:          "TooLong",
		MainCharacterID:   3003,
		MainCharacterName: "Main",
		TitleTags:         []string{longTag},
	})
	if err == nil {
		t.Fatal("expected title_tags single item length validation error")
	}

	tooMany := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13"}
	_, err = svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "contributors",
		Nickname:          "TooMany",
		MainCharacterID:   3004,
		MainCharacterName: "Main",
		TitleTags:         tooMany,
	})
	if err == nil {
		t.Fatal("expected title_tags total count validation error")
	}
}

func TestFuxiHallUpdateCardReplacesTitleTags(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := NewFuxiHallService()

	card, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "leadership",
		Nickname:          "Replace",
		MainCharacterID:   3005,
		MainCharacterName: "Main",
		TitleTags:         []string{"旧标签"},
	})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	nextTags := []string{"  新标签  ", "新标签", "战略"}
	updated, err := svc.UpdateCard(card.ID, &FuxiHallUpdateCardRequest{TitleTags: &nextTags})
	if err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}
	if len(updated.TitleTags) != 2 || updated.TitleTags[0] != "新标签" || updated.TitleTags[1] != "战略" {
		t.Fatalf("unexpected updated title tags: %#v", updated.TitleTags)
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
