package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
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
		MainCharacterName: "Main",
		Title:             "Title",
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
		MainCharacterName: "Main",
		Title:             "Lead",
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
		MainCharacterName: "Main",
		Title:             "Builder",
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

func TestFuxiHallCreateCardValidatesNumericStyleBounds(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := NewFuxiHallService()

	tooShortCover := 80
	_, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "contributors",
		Nickname:          "Bound",
		MainCharacterName: "Bound",
		Title:             "Bound",
		CoverHeight:       &tooShortCover,
	})
	if err == nil {
		t.Fatal("expected cover_height bound error")
	}

	tooBigFont := 30
	_, err = svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "contributors",
		Nickname:          "Bound",
		MainCharacterName: "Bound",
		Title:             "Bound",
		FontScale:         &tooBigFont,
	})
	if err == nil {
		t.Fatal("expected font_scale bound error")
	}
}

func TestFuxiHallReorderCardsIsAtomic(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	svc := NewFuxiHallService()

	cardA, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "leadership",
		Nickname:          "A",
		MainCharacterName: "A",
		Title:             "A",
	})
	if err != nil {
		t.Fatalf("CreateCard A: %v", err)
	}
	cardB, err := svc.CreateCard(&FuxiHallCreateCardRequest{
		PageKey:           "leadership",
		Nickname:          "B",
		MainCharacterName: "B",
		Title:             "B",
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
		MainCharacterName: "Dup",
		Title:             "Dup",
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
