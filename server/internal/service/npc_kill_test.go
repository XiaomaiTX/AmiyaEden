package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestParseReason(t *testing.T) {
	result := parseReason("123: 2, 456:3, 123:4, invalid, 789: x, :1")

	if result[123] != 6 {
		t.Fatalf("expected npc 123 total 6, got %d", result[123])
	}
	if result[456] != 3 {
		t.Fatalf("expected npc 456 total 3, got %d", result[456])
	}
	if _, ok := result[789]; ok {
		t.Fatalf("did not expect invalid npc 789 entry to be present")
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 valid npc entries, got %d", len(result))
	}
}

func TestParseDateRange(t *testing.T) {
	start, end := parseDateRange("2026-03-01", "2026-03-31")
	if start == nil || start.Format("2006-01-02 15:04:05") != "2026-03-01 00:00:00" {
		t.Fatalf("unexpected start: %v", start)
	}
	if end == nil || end.Format("2006-01-02 15:04:05") != "2026-03-31 23:59:59" {
		t.Fatalf("unexpected end: %v", end)
	}

	start, end = parseDateRange("bad", "")
	if start != nil {
		t.Fatalf("expected nil start for invalid input, got %v", start)
	}
	if end != nil {
		t.Fatalf("expected nil end for empty input, got %v", end)
	}
}

func TestCalcSummaryIncludesEssTransfers(t *testing.T) {
	svc := NewNpcKillService()
	base := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	summary := svc.calcSummary([]model.EVECharacterWalletJournal{
		{ID: 1, RefType: "bounty_prizes", Amount: 100, Tax: -10, Date: base},
		{ID: 2, RefType: "ess_escrow_transfer", Amount: 50, Tax: 0, Date: base.Add(time.Minute)},
		{ID: 3, RefType: "bounty_prizes", Amount: 80, Tax: -8, Date: base.Add(2 * time.Minute)},
	})

	if summary.TotalBounty != 180 {
		t.Fatalf("expected bounty total 180, got %v", summary.TotalBounty)
	}
	if summary.TotalESS != 50 {
		t.Fatalf("expected ESS total 50, got %v", summary.TotalESS)
	}
	if summary.TotalTax != -18 {
		t.Fatalf("expected tax total -18, got %v", summary.TotalTax)
	}
	if summary.ActualIncome != 212 {
		t.Fatalf("expected actual income 212 with ESS contribution, got %v", summary.ActualIncome)
	}
	if summary.TotalRecords != 2 {
		t.Fatalf("expected 2 bounty records, got %d", summary.TotalRecords)
	}
}

func TestNormalizeTickerSet(t *testing.T) {
	set := normalizeTickerSet(" fuxi, FMA.1, ,fuxi,  test ")
	if len(set) != 3 {
		t.Fatalf("expected 3 unique tickers, got %d", len(set))
	}
	if _, ok := set["FUXI"]; !ok {
		t.Fatalf("expected FUXI to exist")
	}
	if _, ok := set["FMA.1"]; !ok {
		t.Fatalf("expected FMA.1 to exist")
	}
	if _, ok := set["TEST"]; !ok {
		t.Fatalf("expected TEST to exist")
	}
}

func TestFilterCharactersByTicker(t *testing.T) {
	resetCorporationTickerCache()
	setCachedCorpTicker(1001, "FUXI")
	setCachedCorpTicker(1002, "FMA.1")
	setCachedCorpTicker(1003, "TEST")

	svc := NewNpcKillService()
	chars := []model.EveCharacter{
		{CharacterID: 11, CorporationID: 1001},
		{CharacterID: 22, CorporationID: 1002},
		{CharacterID: 33, CorporationID: 1003},
	}

	filtered, err := svc.filterCharactersByTicker(chars, normalizeTickerSet("fuxi, fma.1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 characters after filter, got %d", len(filtered))
	}
	if filtered[0].CharacterID != 11 || filtered[1].CharacterID != 22 {
		t.Fatalf("unexpected filtered order/content: %+v", filtered)
	}
}

func TestFilterCharactersByTickerNoMatch(t *testing.T) {
	resetCorporationTickerCache()
	setCachedCorpTicker(2001, "AAA")
	setCachedCorpTicker(2002, "BBB")

	svc := NewNpcKillService()
	chars := []model.EveCharacter{
		{CharacterID: 101, CorporationID: 2001},
		{CharacterID: 102, CorporationID: 2002},
	}

	filtered, err := svc.filterCharactersByTicker(chars, normalizeTickerSet("fuxi"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filtered) != 0 {
		t.Fatalf("expected empty filter result, got %d", len(filtered))
	}
}

func TestCalcBySystemReportsSDEQueryErrorWhenSystemLookupFails(t *testing.T) {
	global.SetLogger(zap.NewNop())

	db := newServiceTestDB(t, "npc-kill-system-lookup-error", &model.SystemConfig{}, &model.SdeVersion{})
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	svc := NewNpcKillService()
	result := svc.calcBySystem([]model.EVECharacterWalletJournal{
		{RefType: "bounty_prizes", ContextID: 30003685, Amount: 1},
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 system summary, got %d", len(result))
	}
	if result[0].SolarSystemName != "Unknown System #30003685" {
		t.Fatalf("unexpected fallback name: %q", result[0].SolarSystemName)
	}

	status, err := NewSdeService().GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.LastQuerySource != "npc_kill.GetSolarSystemNames" {
		t.Fatalf("LastQuerySource = %q", status.LastQuerySource)
	}
	if status.LastQueryError == "" {
		t.Fatal("expected LastQueryError to be recorded")
	}
}

func TestCalcByNpcReportsSDEQueryErrorWhenTypeLookupFails(t *testing.T) {
	global.SetLogger(zap.NewNop())

	db := newServiceTestDB(t, "npc-kill-type-lookup-error", &model.SystemConfig{}, &model.SdeVersion{})
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	svc := NewNpcKillService()
	result := svc.calcByNpc([]model.EVECharacterWalletJournal{
		{RefType: "bounty_prizes", Reason: "123: 1", Amount: 1},
	}, "zh")

	if len(result) != 1 {
		t.Fatalf("expected 1 npc summary, got %d", len(result))
	}
	if result[0].NpcName != "Unknown NPC #123" {
		t.Fatalf("unexpected fallback name: %q", result[0].NpcName)
	}

	status, err := NewSdeService().GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.LastQuerySource != "sde_repository.GetTypes" {
		t.Fatalf("LastQuerySource = %q", status.LastQuerySource)
	}
	if status.LastQueryError == "" {
		t.Fatal("expected LastQueryError to be recorded")
	}
}
