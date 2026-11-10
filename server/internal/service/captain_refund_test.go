package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"encoding/json"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestUnifiedNewbroBoundariesAndMissingData(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	birthday := now.AddDate(0, 0, -90)
	older := birthday.Add(-time.Nanosecond)
	for _, tc := range []struct {
		name   string
		chars  []NewbroCharacterSnapshot
		want   bool
		reason string
	}{
		{"inclusive limits", []NewbroCharacterSnapshot{{1, 4_000_000, true, &birthday}, {2, 6_000_000, true, &birthday}}, true, ""},
		{"sum exceeds", []NewbroCharacterSnapshot{{1, 6_000_000, true, &birthday}, {2, 6_000_000, true, &birthday}}, false, "skill_points_too_high"},
		{"oldest character", []NewbroCharacterSnapshot{{1, 1, true, &birthday}, {2, 1, true, &older}}, false, "character_too_old"},
		{"missing birthday", []NewbroCharacterSnapshot{{1, 1, true, nil}}, false, "missing_character_data"},
		{"missing skills", []NewbroCharacterSnapshot{{1, 0, false, &birthday}}, false, "missing_character_data"},
		{"known zero skills", []NewbroCharacterSnapshot{{1, 0, true, &birthday}}, true, ""},
		{"no characters", nil, false, "no_characters"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := EvaluateUnifiedNewbro(tc.chars, 10_000_000, 90, now)
			if r.IsCurrentlyNewbro != tc.want || r.DisqualifiedReason != tc.reason {
				t.Fatalf("unexpected result: %+v", r)
			}
		})
	}
}
func TestRefundMatchingAndTimeBoundaries(t *testing.T) {
	start := time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	calc := CaptainRefundCalculation{Request: CaptainRefundRequest{CaptainCharacterID: 1, SystemID: 30000001, Start: start, End: end}, Settings: CaptainRefundSettings{TaxRate: 30}, Members: []RefundMember{{CharacterID: 1, UserID: 1}, {CharacterID: 2, UserID: 2, Eligibility: NewbroEligibilityResult{IsCurrentlyNewbro: true}}, {CharacterID: 3, UserID: 3}}}
	journal := func(id, char int64, at time.Time, amount float64) model.EVECharacterWalletJournal {
		return model.EVECharacterWalletJournal{ID: id, CharacterID: char, Date: at, Amount: amount, RefType: "bounty_prizes", ContextID: 30000001}
	}
	rows := []model.EVECharacterWalletJournal{journal(1, 1, start, 1_000_000), journal(2, 2, start.Add(5*time.Minute), 2_000_000), journal(3, 3, start.Add(4*time.Minute), 3_000_000), journal(4, 2, start.Add(5*time.Minute+time.Nanosecond), 10_000_000), journal(5, 2, end, 10_000_000)}
	rows = append(rows, rows[1]) // Duplicate input must never double count.
	calculateRefundJournals(&calc, rows)
	if !calc.HasNewbro || calc.Bounty != 6_000_000 || calc.Reward != 1.8 || len(calc.Journals) != 3 {
		t.Fatalf("unexpected calculation: %+v", calc)
	}
	for _, j := range calc.Journals {
		if j.CharacterID != 1 && j.MatchJournalID != 1 {
			t.Fatal("missing match evidence")
		}
	}
	calc.Members[1].UserID = 1
	calc.HasNewbro = false
	calc.Journals = nil
	calc.Bounty = 0
	calculateRefundJournals(&calc, rows)
	if calc.HasNewbro {
		t.Fatal("applicant alt cannot supply newbro slot")
	}
}
func refundTestDB(t *testing.T) (*gorm.DB, *CaptainRefundService, CaptainRefundRequest) {
	t.Helper()
	db := newServiceTestDB(t, "captain_refund", &model.User{}, &model.UserRole{}, &model.EveCharacter{}, &model.EveCharacterSkill{}, &model.EVECharacterWallet{}, &model.EVECharacterWalletJournal{}, &model.SystemConfig{}, &model.CaptainRefund{}, &model.CaptainRefundClaim{}, &model.SystemWallet{}, &model.WalletTransaction{}, &model.AuditEvent{})
	oldDB, oldRedis := global.DB, global.Redis
	global.DB = db
	global.Redis = nil
	t.Cleanup(func() { global.DB = oldDB; global.Redis = oldRedis; clearCorpPolicyCache() })
	for _, uid := range []uint{1, 2, 3} {
		seedWalletCapabilityEnabledUserForTests(t, db, uid, int64(uid), 1001)
	}
	setWalletCapabilityPolicyForTests(t, db, 1001, true)
	birthday := time.Now().AddDate(0, 0, -20)
	for _, uid := range []uint{1, 2, 3} {
		if err := db.Model(&model.EveCharacter{}).Where("character_id = ?", uid).Updates(map[string]interface{}{"birthday": birthday, "character_name": []string{"", "Captain", "New Member", "Other Captain"}[uid]}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&model.EveCharacterSkill{CharacterID: int64(uid), TotalSP: 1_000_000}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range []model.UserRole{{UserID: 1, RoleCode: model.RoleCaptain}, {UserID: 3, RoleCode: model.RoleCaptain}, {UserID: 9, RoleCode: model.RoleAdmin}} {
		if err := db.Create(&r).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := NewCaptainRefundService()
	if _, err := svc.UpdateSettings(9, CaptainRefundSettings{MaxTotalSP: 10_000_000, MaxAgeDays: 90, TaxRate: 30, AllowedSystemIDs: []int64{30000001}}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	start := now.Add(-time.Hour)
	end := now.Add(-time.Minute)
	for i := int64(1); i <= 3; i++ {
		if err := db.Create(&model.EVECharacterWalletJournal{ID: i, CharacterID: i, Amount: 1_000_000, ContextID: 30000001, ContextIDType: "solar_system_id", RefType: "bounty_prizes", Date: start.Add(10 * time.Minute)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db, svc, CaptainRefundRequest{CaptainCharacterID: 1, MemberNames: []string{"new member", "NEW MEMBER", "Missing", "Captain"}, SystemID: 30000001, Start: start, End: end}
}
func TestRefundSubmissionDedupRejectReleaseAndApproval(t *testing.T) {
	db, svc, req := refundTestDB(t)
	preview, err := svc.Calculate(1, req, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !preview.CanSubmit || len(preview.Members) != 2 || len(preview.IgnoredNames) != 1 || preview.Reward != 0.6 {
		t.Fatalf("bad preview %+v", preview)
	}
	mentor, err := NewMentorEligibilityService().EvaluateEligibility(2)
	if err != nil || !mentor.IsEligible {
		t.Fatalf("shared mentor rule: %+v %v", mentor, err)
	}
	row, err := svc.Create(1, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Create(1, req); err == nil {
		t.Fatal("duplicate allowed")
	}
	swapped := req
	swapped.CaptainCharacterID = 3
	swapped.MemberNames = []string{"New Member"}
	c, err := svc.Calculate(3, swapped, time.Now())
	if err != nil || c.CanSubmit || len(c.Conflicts) != 1 {
		t.Fatalf("cross captain claim failed %+v %v", c, err)
	}
	if err = svc.Review(9, row.ID, false, ""); err == nil {
		t.Fatal("empty rejection accepted")
	}
	if err = svc.Review(9, row.ID, false, "wrong time"); err != nil {
		t.Fatal(err)
	}
	replacement, err := svc.Create(1, req)
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.Review(9, replacement.ID, true, ""); err != nil {
		t.Fatal(err)
	}
	if err = svc.Review(9, replacement.ID, true, ""); err == nil {
		t.Fatal("repeat approval allowed")
	}
	var wallet model.SystemWallet
	if err = db.Where("user_id = ?", 1).First(&wallet).Error; err != nil || wallet.Balance != 0.6 {
		t.Fatalf("wallet %+v %v", wallet, err)
	}
	var count int64
	db.Model(&model.WalletTransaction{}).Where("ref_type = ?", "captain_refund").Count(&count)
	if count != 1 {
		t.Fatalf("wallet paid %d times", count)
	}
	if _, err = svc.Detail(3, replacement.ID, false); err == nil {
		t.Fatal("other captain can read application")
	}
}
func TestRefundApprovalFailureRollsBack(t *testing.T) {
	db, svc, req := refundTestDB(t)
	row, err := svc.Create(1, req)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a storage failure during payout after the approval transition.
	if err = db.Callback().Create().Before("gorm:create").Register("refund_fail_wallet", func(tx *gorm.DB) {
		if tx.Statement.Table == "wallet_transaction" {
			_ = tx.AddError(gorm.ErrInvalidTransaction)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err = svc.Review(9, row.ID, true, ""); err == nil {
		t.Fatal("expected payout failure")
	}
	got, err := svc.Detail(1, row.ID, false)
	if err != nil || got.Status != "pending" {
		t.Fatalf("approval was not rolled back %+v %v", got, err)
	}
	var count int64
	db.Model(&model.CaptainRefundClaim{}).Count(&count)
	if count != 2 {
		t.Fatalf("claims lost on failure: %d", count)
	}
}
func TestRefundInvalidInputsAndSettingsZero(t *testing.T) {
	_, svc, req := refundTestDB(t)
	for _, mutate := range []func(*CaptainRefundRequest){func(r *CaptainRefundRequest) { r.CaptainCharacterID = 2 }, func(r *CaptainRefundRequest) { r.SystemID = 1 }, func(r *CaptainRefundRequest) { r.Start = time.Now().Add(-31 * 24 * time.Hour) }, func(r *CaptainRefundRequest) { r.End = time.Now().Add(time.Hour) }, func(r *CaptainRefundRequest) { r.End = r.Start }} {
		r := req
		mutate(&r)
		if _, err := svc.Calculate(1, r, time.Now()); err == nil {
			t.Fatal("invalid request accepted")
		}
	}
	if _, err := svc.Calculate(2, req, time.Now()); err == nil {
		t.Fatal("non captain accepted")
	}
	cfg, err := svc.UpdateSettings(9, CaptainRefundSettings{AllowedSystemIDs: []int64{30000001}})
	if err != nil || cfg.TaxRate != 0 {
		t.Fatal(err)
	}
	if got := svc.Settings(); got.MaxTotalSP != 0 || got.MaxAgeDays != 0 || got.TaxRate != 0 {
		t.Fatalf("zero settings replaced %+v", got)
	}
	raw, _ := json.Marshal(cfg)
	if len(raw) == 0 {
		t.Fatal("empty settings payload")
	}
}

func TestRefundConcurrentClaimsAndApproval(t *testing.T) {
	db, svc, _ := refundTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	// Two transactions race to reserve the same globally identified bounty.
	results := make(chan error, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			results <- svc.repo.Transaction(func(tx *gorm.DB) error {
				row := model.CaptainRefund{UserID: 1, Status: "pending", Snapshot: `{}`, Reward: 1}
				return svc.repo.Create(tx, &row, []model.CaptainRefundClaim{{CharacterID: 100, JournalID: 500}})
			})
		}()
	}
	close(start)
	successes := 0
	for i := 0; i < 2; i++ {
		if <-results == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent claims succeeded %d times", successes)
	}
	var row model.CaptainRefund
	if err = db.First(&row).Error; err != nil {
		t.Fatal(err)
	}
	start = make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() { <-start; results <- svc.Review(9, row.ID, true, "") }()
	}
	close(start)
	successes = 0
	for i := 0; i < 2; i++ {
		if <-results == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent approval succeeded %d times", successes)
	}
	var count int64
	db.Model(&model.WalletTransaction{}).Where("ref_type = ?", model.WalletRefCaptainRefund).Count(&count)
	if count != 1 {
		t.Fatalf("paid %d times", count)
	}
}

func TestRefundSnapshotFrozenAndNoEffectiveNewbro(t *testing.T) {
	db, svc, req := refundTestDB(t)
	row, err := svc.Create(1, req)
	if err != nil {
		t.Fatal(err)
	}
	cfg := svc.Settings()
	cfg.TaxRate = 90
	cfg.MaxTotalSP = 0
	if _, err = svc.UpdateSettings(9, cfg); err != nil {
		t.Fatal(err)
	}
	frozen, err := svc.Detail(1, row.ID, false)
	if err != nil || frozen.Calculation.Settings.TaxRate != 30 || frozen.Reward != 0.6 || !frozen.Calculation.Members[1].Eligibility.IsCurrentlyNewbro {
		t.Fatalf("snapshot changed %+v %v", frozen, err)
	}
	if err = svc.Review(9, row.ID, false, "retry"); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Create(1, req); err == nil {
		t.Fatal("no newbro accepted")
	}
	cfg.MaxTotalSP = 10_000_000
	if _, err = svc.UpdateSettings(9, cfg); err != nil {
		t.Fatal(err)
	}
	// Newbro exists, but has no matching bounty in the requested window.
	if err = db.Model(&model.EVECharacterWalletJournal{}).Where("character_id = ?", 2).Update("date", req.Start.Add(40*time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Create(1, req); err == nil {
		t.Fatal("nonparticipating newbro accepted")
	}
}
