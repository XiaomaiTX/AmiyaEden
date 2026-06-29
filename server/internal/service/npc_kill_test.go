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

func TestCalcSummaryIncludesEssTransfersAndCorporateRewardPayouts(t *testing.T) {
	svc := NewNpcKillService()
	base := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	summary := svc.calcSummary([]model.EVECharacterWalletJournal{
		{ID: 1, RefType: "bounty_prizes", Amount: 100, Tax: -10, Date: base},
		{ID: 2, RefType: "ess_escrow_transfer", Amount: 50, Tax: 0, Date: base.Add(time.Minute)},
		{ID: 3, RefType: "bounty_prizes", Amount: 80, Tax: -8, Date: base.Add(2 * time.Minute)},
		{ID: 4, RefType: "corporate_reward_payout", Amount: 40, Tax: 0, Date: base.Add(3 * time.Minute)},
	})

	if summary.TotalBounty != 220 {
		t.Fatalf("expected bounty total 220 with incursion contribution, got %v", summary.TotalBounty)
	}
	if summary.TotalESS != 50 {
		t.Fatalf("expected ESS total 50, got %v", summary.TotalESS)
	}
	if summary.TotalIncursion != 40 {
		t.Fatalf("expected incursion total 40, got %v", summary.TotalIncursion)
	}
	if summary.TotalTax != -18 {
		t.Fatalf("expected tax total -18, got %v", summary.TotalTax)
	}
	if summary.ActualIncome != 252 {
		t.Fatalf("expected actual income 252 without double-counting incursion, got %v", summary.ActualIncome)
	}
	if summary.TotalRecords != 3 {
		t.Fatalf("expected 3 bounty records including corporate reward payout, got %d", summary.TotalRecords)
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

func TestBuildCorpMemberSummariesByUser(t *testing.T) {
	journals := []model.EVECharacterWalletJournal{
		{CharacterID: 1001, RefType: "bounty_prizes", Amount: 100, Tax: -10},
		{CharacterID: 1002, RefType: "bounty_prizes", Amount: 50, Tax: -5},
		{CharacterID: 1002, RefType: "ess_escrow_transfer", Amount: 20, Tax: 0},
		{CharacterID: 2001, RefType: "bounty_prizes", Amount: 90, Tax: -9},
		{CharacterID: 2001, RefType: "corporate_reward_payout", Amount: 40, Tax: 0},
		{CharacterID: 2001, RefType: "agent_mission_reward", Amount: 10, Tax: 0},
	}
	charUserMap := map[int64]uint{
		1001: 1,
		1002: 1,
		2001: 2,
	}
	charNameMap := map[int64]string{
		1001: "Zulu Pilot",
		1002: "Alpha Pilot",
		2001: "Bravo Pilot",
	}
	userNicknameMap := map[uint]string{
		1: "  Commander One  ",
		2: "",
	}

	got := buildCorpMemberSummaries(journals, charUserMap, charNameMap, userNicknameMap)
	if len(got) != 2 {
		t.Fatalf("expected 2 users, got %d", len(got))
	}

	var user1 *NpcKillCorpMemberSummary
	var user2 *NpcKillCorpMemberSummary
	for i := range got {
		switch got[i].UserID {
		case 1:
			user1 = &got[i]
		case 2:
			user2 = &got[i]
		}
	}
	if user1 == nil || user2 == nil {
		t.Fatalf("expected both user summaries, got %+v", got)
	}

	if user1.DisplayName != "Commander One" {
		t.Fatalf("user1 display name = %q, want %q", user1.DisplayName, "Commander One")
	}
	if user1.CharacterCount != 2 {
		t.Fatalf("user1 character_count = %d, want 2", user1.CharacterCount)
	}
	if user1.TotalBounty != 150 {
		t.Fatalf("user1 total_bounty = %v, want 150", user1.TotalBounty)
	}
	if user1.TotalESS != 20 {
		t.Fatalf("user1 total_ess = %v, want 20", user1.TotalESS)
	}
	if user1.RecordCount != 2 {
		t.Fatalf("user1 record_count = %d, want 2", user1.RecordCount)
	}
	if user1.ActualIncome != 155 {
		t.Fatalf("user1 actual_income = %v, want 155", user1.ActualIncome)
	}

	if user2.DisplayName != "Bravo Pilot" {
		t.Fatalf("user2 display name = %q, want %q", user2.DisplayName, "Bravo Pilot")
	}
	if user2.CharacterCount != 1 {
		t.Fatalf("user2 character_count = %d, want 1", user2.CharacterCount)
	}
	if user2.TotalMission != 10 {
		t.Fatalf("user2 total_mission = %v, want 10", user2.TotalMission)
	}
	if user2.TotalBounty != 130 {
		t.Fatalf("user2 total_bounty = %v, want 130", user2.TotalBounty)
	}
	if user2.TotalIncursion != 40 {
		t.Fatalf("user2 total_incursion = %v, want 40", user2.TotalIncursion)
	}
	if user2.RecordCount != 2 {
		t.Fatalf("user2 record_count = %d, want 2", user2.RecordCount)
	}
	if user2.ActualIncome != 131 {
		t.Fatalf("user2 actual_income = %v, want 131", user2.ActualIncome)
	}
}

func TestNpcKillFilterHelpers(t *testing.T) {
	refTypes := normalizeNpcIncomeRefTypes([]string{"bounty_prizes", "bad", "corporate_reward_payout", "bounty_prizes"})
	if len(refTypes) != 2 || refTypes[0] != "bounty_prizes" || refTypes[1] != "corporate_reward_payout" {
		t.Fatalf("unexpected normalized ref types: %+v", refTypes)
	}

	emptyIntersection := normalizeNpcIncomeRefTypes([]string{"bad"})
	if len(emptyIntersection) != 1 || emptyIntersection[0] == "bad" {
		t.Fatalf("expected unsupported ref type sentinel, got %+v", emptyIntersection)
	}

	charIDs := filterAllowedCharacterIDs([]int64{1, 2, 3}, []int64{3, 9, 1})
	if len(charIDs) != 2 || charIDs[0] != 1 || charIDs[1] != 3 {
		t.Fatalf("unexpected filtered character IDs: %+v", charIDs)
	}
}

func TestCalcTrendAndJournalsIncludeCorporateRewardPayoutButNotNpcSystemStats(t *testing.T) {
	global.SetLogger(zap.NewNop())

	db := newServiceTestDB(t, "npc-kill-corporate-reward", &model.SystemConfig{}, &model.SdeVersion{}, &model.MapSolarSystem{})
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	svc := NewNpcKillService()
	base := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	journals := []model.EVECharacterWalletJournal{
		{ID: 1, CharacterID: 1001, RefType: "bounty_prizes", ContextID: 30000142, Reason: "123: 2", Amount: 100, Tax: -10, Date: base},
		{ID: 2, CharacterID: 1001, RefType: "corporate_reward_payout", ContextID: 0, Reason: "", Amount: 40, Tax: 0, Date: base.Add(time.Minute)},
	}

	byNpc := svc.calcByNpc(journals, "zh")
	if len(byNpc) != 1 || byNpc[0].NpcID != 123 || byNpc[0].Count != 2 {
		t.Fatalf("expected only bounty_prizes to affect npc stats, got %+v", byNpc)
	}

	bySystem := svc.calcBySystem(journals)
	if len(bySystem) != 1 || bySystem[0].SolarSystemID != 30000142 || bySystem[0].Count != 1 || bySystem[0].Amount != 100 {
		t.Fatalf("expected only bounty_prizes to affect system stats, got %+v", bySystem)
	}

	trend := svc.calcTrend(journals)
	if len(trend) != 1 {
		t.Fatalf("expected trend to aggregate bounty_prizes and corporate_reward_payout on the same day, got %+v", trend)
	}
	if trend[0].Amount != 140 || trend[0].Count != 2 {
		t.Fatalf("unexpected trend amounts: %+v", trend)
	}

	items := svc.buildJournalItems(journals, map[int64]string{1001: "Pilot One"})
	if len(items) != 2 {
		t.Fatalf("expected 2 journal items, got %+v", items)
	}
	if items[1].RefType != "corporate_reward_payout" {
		t.Fatalf("expected second journal to be corporate_reward_payout, got %+v", items[1])
	}
	if items[1].SolarSystemName != "" || items[1].SolarSystemID != 0 {
		t.Fatalf("expected corporate_reward_payout journal to omit solar system details, got %+v", items[1])
	}
}

func TestGetAllNpcKillsAppliesCharacterAndRefTypeFilters(t *testing.T) {
	global.SetLogger(zap.NewNop())

	db := newServiceTestDB(
		t,
		"npc-kill-get-all",
		&model.User{},
		&model.EveCharacter{},
		&model.EVECharacterWalletJournal{},
		&model.MapSolarSystem{},
		&model.SystemConfig{},
		&model.SdeVersion{},
	)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 1}, Nickname: "Commander One", Role: "user"}).Error; err != nil {
		t.Fatalf("seed user1: %v", err)
	}
	if err := db.Create(&model.EveCharacter{CharacterID: 1001, CharacterName: "Alpha Pilot", UserID: 1, CorporationID: 3001}).Error; err != nil {
		t.Fatalf("seed char1001: %v", err)
	}
	if err := db.Create(&model.EveCharacter{CharacterID: 1002, CharacterName: "Beta Pilot", UserID: 1, CorporationID: 3001}).Error; err != nil {
		t.Fatalf("seed char1002: %v", err)
	}
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 2}, Nickname: "Commander Two", Role: "user"}).Error; err != nil {
		t.Fatalf("seed user2: %v", err)
	}
	if err := db.Create(&model.EveCharacter{CharacterID: 2001, CharacterName: "Gamma Pilot", UserID: 2, CorporationID: 3002}).Error; err != nil {
		t.Fatalf("seed char2001: %v", err)
	}
	if err := db.Create(&model.MapSolarSystem{SolarSystemID: 30000143, SolarSystemName: "Beta System"}).Error; err != nil {
		t.Fatalf("seed solar system: %v", err)
	}

	base := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	journals := []model.EVECharacterWalletJournal{
		{ID: 1, CharacterID: 1001, RefType: "bounty_prizes", ContextID: 30000142, Reason: "123: 1", Amount: 100, Tax: -10, Date: base},
		{ID: 2, CharacterID: 1002, RefType: "bounty_prizes", ContextID: 30000143, Reason: "456: 2", Amount: 60, Tax: -6, Date: base.Add(time.Minute)},
		{ID: 3, CharacterID: 1002, RefType: "corporate_reward_payout", ContextID: 0, Reason: "", Amount: 40, Tax: 0, Date: base.Add(2 * time.Minute)},
		{ID: 4, CharacterID: 2001, RefType: "bounty_prizes", ContextID: 30000142, Reason: "999: 9", Amount: 999, Tax: -99, Date: base.Add(3 * time.Minute)},
	}
	if err := db.Create(&journals).Error; err != nil {
		t.Fatalf("seed journals: %v", err)
	}

	svc := NewNpcKillService()
	resp, err := svc.GetAllNpcKills(1, &NpcKillAllRequest{
		StartDate:    "2026-03-27",
		EndDate:      "2026-03-27",
		Language:     "zh",
		CharacterIDs: []int64{1002, 9999},
		UserIDs:      []uint{1},
		RefTypes:     []string{"bounty_prizes", "corporate_reward_payout"},
	})
	if err != nil {
		t.Fatalf("GetAllNpcKills returned error: %v", err)
	}

	if resp.Summary.TotalBounty != 100 {
		t.Fatalf("summary total_bounty = %v, want 100", resp.Summary.TotalBounty)
	}
	if resp.Summary.TotalIncursion != 40 {
		t.Fatalf("summary total_incursion = %v, want 40", resp.Summary.TotalIncursion)
	}
	if resp.Summary.TotalTax != -6 {
		t.Fatalf("summary total_tax = %v, want -6", resp.Summary.TotalTax)
	}
	if resp.Summary.ActualIncome != 94 {
		t.Fatalf("summary actual_income = %v, want 94", resp.Summary.ActualIncome)
	}
	if resp.Summary.TotalRecords != 2 {
		t.Fatalf("summary total_records = %d, want 2", resp.Summary.TotalRecords)
	}

	if len(resp.ByNpc) != 1 || resp.ByNpc[0].NpcID != 456 || resp.ByNpc[0].Count != 2 {
		t.Fatalf("unexpected by_npc result: %+v", resp.ByNpc)
	}
	if len(resp.BySystem) != 1 || resp.BySystem[0].SolarSystemID != 30000143 || resp.BySystem[0].Count != 1 || resp.BySystem[0].Amount != 60 {
		t.Fatalf("unexpected by_system result: %+v", resp.BySystem)
	}
	if len(resp.Trend) != 1 || resp.Trend[0].Amount != 100 || resp.Trend[0].Count != 2 {
		t.Fatalf("unexpected trend result: %+v", resp.Trend)
	}
	if len(resp.Journals) != 2 {
		t.Fatalf("expected 2 journals after filters, got %+v", resp.Journals)
	}
	if resp.Journals[0].CharacterName != "Beta Pilot" || resp.Journals[1].CharacterName != "Beta Pilot" {
		t.Fatalf("expected filtered journals to include selected character name, got %+v", resp.Journals)
	}
	if resp.Journals[0].RefType != "corporate_reward_payout" || resp.Journals[1].RefType != "bounty_prizes" {
		t.Fatalf("expected corporate_reward_payout to survive filters alongside bounty_prizes, got %+v", resp.Journals)
	}
}
