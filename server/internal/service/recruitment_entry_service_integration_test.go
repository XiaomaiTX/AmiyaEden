package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"amiya-eden/global"
	"amiya-eden/internal/model"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type fakeRecruitmentEntryUserRepo struct {
	user *model.User
	err  error
}

func (f *fakeRecruitmentEntryUserRepo) GetByID(uint) (*model.User, error) {
	return f.user, f.err
}

func (f *fakeRecruitmentEntryUserRepo) ListByQQ(string) ([]model.User, error) {
	if f.user == nil {
		return nil, f.err
	}
	return []model.User{*f.user}, f.err
}

func (f *fakeRecruitmentEntryUserRepo) GetByIDForUpdateTx(*gorm.DB, uint) (*model.User, error) {
	return f.user, f.err
}

type scriptedRecruitmentEntryUserRepo struct {
	getByID            func(uint) (*model.User, error)
	listByQQ           func(string) ([]model.User, error)
	getByIDForUpdateTx func(*gorm.DB, uint) (*model.User, error)
}

func (f *scriptedRecruitmentEntryUserRepo) GetByID(id uint) (*model.User, error) {
	if f.getByID == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return f.getByID(id)
}

func (f *scriptedRecruitmentEntryUserRepo) ListByQQ(qq string) ([]model.User, error) {
	if f.listByQQ == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return f.listByQQ(qq)
}

func (f *scriptedRecruitmentEntryUserRepo) GetByIDForUpdateTx(tx *gorm.DB, id uint) (*model.User, error) {
	if f.getByIDForUpdateTx == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return f.getByIDForUpdateTx(tx, id)
}

func newRecruitmentEntryServiceTestDB(t *testing.T) *gorm.DB {
	db := newServiceTestDB(t, "recruit_entry_svc_test",
		&model.SystemConfig{},
		&model.NewbroRecruitment{},
		&model.NewbroRecruitmentEntry{},
		&model.User{},
		&model.UserRole{},
		&model.EveCharacter{},
		&model.SystemWallet{},
		&model.WalletTransaction{},
	)
	return db
}

func TestRecruitmentEntryService_SubmitQQDeduplicatesSameRecruitmentAndQQ(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigNewbroRecruitQQURL,
		Value: "https://example.com/qq",
		Desc:  "test qq url",
	}).Error; err != nil {
		t.Fatalf("seed recruit qq url: %v", err)
	}

	recruitment := &model.NewbroRecruitment{UserID: 100, Code: "abc123", GeneratedAt: time.Now()}
	if err := db.Create(recruitment).Error; err != nil {
		t.Fatalf("seed recruitment: %v", err)
	}

	svc := NewRecruitmentEntryService()
	firstURL, err := svc.SubmitQQ("abc123", "123456", time.Now())
	if err != nil {
		t.Fatalf("first submit qq: %v", err)
	}
	secondURL, err := svc.SubmitQQ("abc123", "123456", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("second submit qq: %v", err)
	}

	if firstURL != "https://example.com/qq" || secondURL != firstURL {
		t.Fatalf("expected both submits to return the configured QQ URL, got %q and %q", firstURL, secondURL)
	}

	var count int64
	if err := db.Model(&model.NewbroRecruitmentEntry{}).
		Where("recruitment_id = ? AND qq = ?", recruitment.ID, "123456").
		Count(&count).Error; err != nil {
		t.Fatalf("count recruitment entries: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one deduplicated recruitment entry, got %d", count)
	}
}

func TestRecruitmentEntryService_SubmitQQRejectsDirectReferralRecords(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigNewbroRecruitQQURL,
		Value: "https://example.com/qq",
		Desc:  "test qq url",
	}).Error; err != nil {
		t.Fatalf("seed recruit qq url: %v", err)
	}

	recruitment := &model.NewbroRecruitment{
		UserID:      100,
		Code:        "direct-record",
		Source:      model.RecruitmentSourceDirectReferral,
		GeneratedAt: time.Now(),
	}
	if err := db.Create(recruitment).Error; err != nil {
		t.Fatalf("seed direct referral recruitment: %v", err)
	}

	svc := NewRecruitmentEntryService()
	_, err := svc.SubmitQQ(recruitment.Code, "123456", time.Now())
	if err == nil {
		t.Fatal("expected direct referral records to reject public QQ submissions")
	}
	if err.Error() != "该记录不支持公开提交" {
		t.Fatalf("expected direct referral submit error, got %v", err)
	}
}

func TestRecruitmentEntryService_ProcessOngoingEntriesRewardsOnlyFirstMatchedRecruit(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	recruitmentA := &model.NewbroRecruitment{UserID: 100, Code: "link-a", GeneratedAt: now.Add(-48 * time.Hour)}
	recruitmentB := &model.NewbroRecruitment{UserID: 200, Code: "link-b", GeneratedAt: now.Add(-48 * time.Hour)}
	if err := db.Create(recruitmentA).Error; err != nil {
		t.Fatalf("seed recruitment A: %v", err)
	}
	if err := db.Create(recruitmentB).Error; err != nil {
		t.Fatalf("seed recruitment B: %v", err)
	}

	entryA := &model.NewbroRecruitmentEntry{
		RecruitmentID: recruitmentA.ID,
		QQ:            "123456",
		EnteredAt:     now.Add(-4 * time.Hour),
		Status:        model.RecruitEntryStatusOngoing,
	}
	entryB := &model.NewbroRecruitmentEntry{
		RecruitmentID: recruitmentB.ID,
		QQ:            "123456",
		EnteredAt:     now.Add(-3 * time.Hour),
		Status:        model.RecruitEntryStatusOngoing,
	}
	if err := db.Create(entryA).Error; err != nil {
		t.Fatalf("seed entry A: %v", err)
	}
	if err := db.Create(entryB).Error; err != nil {
		t.Fatalf("seed entry B: %v", err)
	}

	matchedUser := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-2 * time.Hour)},
		Nickname:  "matched-user",
		QQ:        "123456",
		Role:      model.RoleUser,
	}
	if err := db.Create(matchedUser).Error; err != nil {
		t.Fatalf("seed matched user: %v", err)
	}

	svc := NewRecruitmentEntryService()
	result, err := svc.ProcessOngoingEntries(now)
	if err != nil {
		t.Fatalf("ProcessOngoingEntries() error = %v", err)
	}

	if result.ValidCount != 1 {
		t.Fatalf("expected one valid reward, got %d", result.ValidCount)
	}
	if result.StalledCount != 1 {
		t.Fatalf("expected one stalled duplicate, got %d", result.StalledCount)
	}
	if result.TotalCoinAwarded != model.SysConfigDefaultNewbroRecruitRewardAmount {
		t.Fatalf("expected total rewarded amount %v, got %v", model.SysConfigDefaultNewbroRecruitRewardAmount, result.TotalCoinAwarded)
	}

	var validCount int64
	if err := db.Model(&model.NewbroRecruitmentEntry{}).
		Where("status = ?", model.RecruitEntryStatusValid).
		Count(&validCount).Error; err != nil {
		t.Fatalf("count valid entries: %v", err)
	}
	if validCount != 1 {
		t.Fatalf("expected exactly one valid entry row, got %d", validCount)
	}

	var stalledCount int64
	if err := db.Model(&model.NewbroRecruitmentEntry{}).
		Where("status = ?", model.RecruitEntryStatusStalled).
		Count(&stalledCount).Error; err != nil {
		t.Fatalf("count stalled entries: %v", err)
	}
	if stalledCount != 1 {
		t.Fatalf("expected exactly one stalled entry row, got %d", stalledCount)
	}

	var txs []model.WalletTransaction
	if err := db.Order("id ASC").Find(&txs).Error; err != nil {
		t.Fatalf("list wallet transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected one wallet transaction, got %d", len(txs))
	}
	if txs[0].UserID != recruitmentA.UserID {
		t.Fatalf("expected earliest recruiter %d to receive reward, got %d", recruitmentA.UserID, txs[0].UserID)
	}
	wantRefID := fmt.Sprintf("recruit_matched_user:%d", matchedUser.ID)
	if txs[0].RefID != wantRefID {
		t.Fatalf("expected wallet ref id %q, got %q", wantRefID, txs[0].RefID)
	}
}

func TestRecruitmentEntryService_ProcessOngoingEntriesDrainsBacklogBeyondBatchLimit(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	recruitment := &model.NewbroRecruitment{UserID: 100, Code: "bulk-link", GeneratedAt: now.Add(-120 * 24 * time.Hour)}
	if err := db.Create(recruitment).Error; err != nil {
		t.Fatalf("seed recruitment: %v", err)
	}

	for index := 0; index < 201; index++ {
		entry := &model.NewbroRecruitmentEntry{
			RecruitmentID: recruitment.ID,
			QQ:            fmt.Sprintf("10%04d", index),
			EnteredAt:     now.Add(-100 * 24 * time.Hour),
			Status:        model.RecruitEntryStatusOngoing,
		}
		if err := db.Create(entry).Error; err != nil {
			t.Fatalf("seed entry %d: %v", index, err)
		}
	}

	svc := NewRecruitmentEntryService()
	result, err := svc.ProcessOngoingEntries(now)
	if err != nil {
		t.Fatalf("ProcessOngoingEntries() error = %v", err)
	}

	if result.ProcessedCount != 201 {
		t.Fatalf("expected 201 processed entries, got %d", result.ProcessedCount)
	}
	if result.StalledCount != 201 {
		t.Fatalf("expected 201 stalled entries, got %d", result.StalledCount)
	}

	var ongoingCount int64
	if err := db.Model(&model.NewbroRecruitmentEntry{}).
		Where("status = ?", model.RecruitEntryStatusOngoing).
		Count(&ongoingCount).Error; err != nil {
		t.Fatalf("count ongoing entries: %v", err)
	}
	if ongoingCount != 0 {
		t.Fatalf("expected backlog drain to leave 0 ongoing entries, got %d", ongoingCount)
	}
}

func TestRecruitmentEntryService_ProcessOngoingEntriesKeepsEntryOngoingOnUserLookupError(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	oldLogger := global.CurrentLogger()
	global.DB = db
	global.SetLogger(zap.NewNop())
	t.Cleanup(func() {
		global.DB = oldDB
		global.SetLogger(oldLogger)
	})

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	recruitment := &model.NewbroRecruitment{UserID: 100, Code: "lookup-error", GeneratedAt: now.Add(-120 * 24 * time.Hour)}
	if err := db.Create(recruitment).Error; err != nil {
		t.Fatalf("seed recruitment: %v", err)
	}
	entry := &model.NewbroRecruitmentEntry{
		RecruitmentID: recruitment.ID,
		QQ:            "123456",
		EnteredAt:     now.Add(-100 * 24 * time.Hour),
		Status:        model.RecruitEntryStatusOngoing,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	svc := NewRecruitmentEntryService()
	svc.userRepo = &fakeRecruitmentEntryUserRepo{err: errors.New("lookup failed")}

	result, err := svc.ProcessOngoingEntries(now)
	if err != nil {
		t.Fatalf("ProcessOngoingEntries() error = %v", err)
	}
	if result.ProcessedCount != 0 {
		t.Fatalf("expected lookup errors to skip processing, got %d processed entries", result.ProcessedCount)
	}

	var refreshed model.NewbroRecruitmentEntry
	if err := db.First(&refreshed, entry.ID).Error; err != nil {
		t.Fatalf("reload entry: %v", err)
	}
	if refreshed.Status != model.RecruitEntryStatusOngoing {
		t.Fatalf("expected entry to remain ongoing after lookup error, got %q", refreshed.Status)
	}
}

func TestRecruitmentEntryService_ProcessOngoingEntriesKeepsEntryOngoingWhenQQMatchesMultipleUsers(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	oldLogger := global.CurrentLogger()
	global.DB = db
	global.SetLogger(zap.NewNop())
	t.Cleanup(func() {
		global.DB = oldDB
		global.SetLogger(oldLogger)
	})

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	recruitment := &model.NewbroRecruitment{UserID: 100, Code: "ambiguous-qq", GeneratedAt: now.Add(-120 * 24 * time.Hour)}
	if err := db.Create(recruitment).Error; err != nil {
		t.Fatalf("seed recruitment: %v", err)
	}
	entry := &model.NewbroRecruitmentEntry{
		RecruitmentID: recruitment.ID,
		QQ:            "123456",
		EnteredAt:     now.Add(-2 * time.Hour),
		Status:        model.RecruitEntryStatusOngoing,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	userA := &model.User{BaseModel: model.BaseModel{CreatedAt: now.Add(-time.Hour)}, Nickname: "user-a", QQ: entry.QQ, Role: model.RoleUser}
	userB := &model.User{BaseModel: model.BaseModel{CreatedAt: now.Add(-30 * time.Minute)}, Nickname: "user-b", QQ: entry.QQ, Role: model.RoleUser}
	if err := db.Create(userA).Error; err != nil {
		t.Fatalf("seed user A: %v", err)
	}
	if err := db.Create(userB).Error; err != nil {
		t.Fatalf("seed user B: %v", err)
	}

	svc := NewRecruitmentEntryService()
	result, err := svc.ProcessOngoingEntries(now)
	if err != nil {
		t.Fatalf("ProcessOngoingEntries() error = %v", err)
	}
	if result.ProcessedCount != 0 {
		t.Fatalf("expected ambiguous QQ to skip processing, got %d processed entries", result.ProcessedCount)
	}

	var refreshed model.NewbroRecruitmentEntry
	if err := db.First(&refreshed, entry.ID).Error; err != nil {
		t.Fatalf("reload entry: %v", err)
	}
	if refreshed.Status != model.RecruitEntryStatusOngoing {
		t.Fatalf("expected ambiguous QQ entry to remain ongoing, got %q", refreshed.Status)
	}

	var txCount int64
	if err := db.Model(&model.WalletTransaction{}).Count(&txCount).Error; err != nil {
		t.Fatalf("count wallet transactions: %v", err)
	}
	if txCount != 0 {
		t.Fatalf("expected ambiguous QQ to avoid wallet credits, got %d transactions", txCount)
	}
}

func TestRecruitmentEntryService_ProcessOngoingEntriesKeepsEntryOngoingWhenRecruitmentMissing(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	oldLogger := global.CurrentLogger()
	global.DB = db
	global.SetLogger(zap.NewNop())
	t.Cleanup(func() {
		global.DB = oldDB
		global.SetLogger(oldLogger)
	})

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	entry := &model.NewbroRecruitmentEntry{
		RecruitmentID: 999999,
		QQ:            "123456",
		EnteredAt:     now.Add(-2 * time.Hour),
		Status:        model.RecruitEntryStatusOngoing,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	matchedUser := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-time.Hour)},
		Nickname:  "matched-user",
		QQ:        entry.QQ,
		Role:      model.RoleUser,
	}
	if err := db.Create(matchedUser).Error; err != nil {
		t.Fatalf("seed matched user: %v", err)
	}

	svc := NewRecruitmentEntryService()
	result, err := svc.ProcessOngoingEntries(now)
	if err != nil {
		t.Fatalf("ProcessOngoingEntries() error = %v", err)
	}
	if result.ProcessedCount != 0 {
		t.Fatalf("expected missing recruitment to skip processing, got %d processed entries", result.ProcessedCount)
	}

	var refreshed model.NewbroRecruitmentEntry
	if err := db.First(&refreshed, entry.ID).Error; err != nil {
		t.Fatalf("reload entry: %v", err)
	}
	if refreshed.Status != model.RecruitEntryStatusOngoing {
		t.Fatalf("expected entry to remain ongoing when recruitment is missing, got %q", refreshed.Status)
	}

	var txCount int64
	if err := db.Model(&model.WalletTransaction{}).Count(&txCount).Error; err != nil {
		t.Fatalf("count wallet transactions: %v", err)
	}
	if txCount != 0 {
		t.Fatalf("expected missing recruitment to avoid wallet credits, got %d transactions", txCount)
	}
}

func TestRecruitmentEntryService_GetDirectReferralStatusAllowsCardWhenCurrentUserQQOnlyHasUnrewardedRecruitEntry(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	currentUser := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-48 * time.Hour)},
		Nickname:  "recent-user",
		QQ:        "556677",
		Role:      model.RoleUser,
	}
	if err := db.Create(currentUser).Error; err != nil {
		t.Fatalf("seed current user: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: currentUser.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed current user role: %v", err)
	}

	recruitment := &model.NewbroRecruitment{UserID: 88, Code: "existing-link", GeneratedAt: now.Add(-72 * time.Hour)}
	if err := db.Create(recruitment).Error; err != nil {
		t.Fatalf("seed recruitment: %v", err)
	}
	if err := db.Create(&model.NewbroRecruitmentEntry{
		RecruitmentID: recruitment.ID,
		QQ:            currentUser.QQ,
		EnteredAt:     now.Add(-24 * time.Hour),
		Status:        model.RecruitEntryStatusOngoing,
	}).Error; err != nil {
		t.Fatalf("seed existing recruitment entry: %v", err)
	}

	svc := NewRecruitmentEntryService()
	status, err := svc.GetDirectReferralStatus(currentUser.ID, now)
	if err != nil {
		t.Fatalf("GetDirectReferralStatus() error = %v", err)
	}
	if !status.ShowCard {
		t.Fatal("expected direct referral card to remain visible when the current user only has an unrewarded recruit entry")
	}
	if status.NeedsProfileQQ {
		t.Fatal("expected a user with QQ set to not require profile QQ before referral")
	}
}

func TestRecruitmentEntryService_GetDirectReferralStatusAllowsCardWhenCurrentUserHasPendingRecruitLinkEntry(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	currentUser := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-24 * time.Hour)},
		Nickname:  "recent-user",
		QQ:        "556677",
		Role:      model.RoleUser,
	}
	if err := db.Create(currentUser).Error; err != nil {
		t.Fatalf("seed current user: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: currentUser.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed current user role: %v", err)
	}

	recruitment := &model.NewbroRecruitment{UserID: 88, Code: "existing-link", GeneratedAt: now.Add(-72 * time.Hour)}
	if err := db.Create(recruitment).Error; err != nil {
		t.Fatalf("seed recruitment: %v", err)
	}
	if err := db.Create(&model.NewbroRecruitmentEntry{
		RecruitmentID: recruitment.ID,
		QQ:            currentUser.QQ,
		EnteredAt:     now.Add(-48 * time.Hour),
		Status:        model.RecruitEntryStatusOngoing,
		Source:        model.RecruitEntrySourceLink,
	}).Error; err != nil {
		t.Fatalf("seed pending recruit-link entry: %v", err)
	}

	svc := NewRecruitmentEntryService()
	status, err := svc.GetDirectReferralStatus(currentUser.ID, now)
	if err != nil {
		t.Fatalf("GetDirectReferralStatus() error = %v", err)
	}
	if !status.ShowCard {
		t.Fatal("expected direct referral card to remain visible when the current user only has a pending recruit-link entry")
	}
	if status.NeedsProfileQQ {
		t.Fatal("expected user with saved QQ to not require profile QQ")
	}
}

func TestRecruitmentEntryService_GetDirectReferralStatusHidesCardWhenCurrentUserAlreadyHasRewardedRecruitRecord(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	currentUser := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-48 * time.Hour)},
		Nickname:  "recent-user",
		QQ:        "556677",
		Role:      model.RoleUser,
	}
	if err := db.Create(currentUser).Error; err != nil {
		t.Fatalf("seed current user: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: currentUser.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed current user role: %v", err)
	}

	rewardedAt := now.Add(-6 * time.Hour)
	walletRefID := buildRecruitRewardRefID(currentUser.ID)
	recruitment := &model.NewbroRecruitment{UserID: 88, Code: "valid-link", GeneratedAt: now.Add(-72 * time.Hour)}
	if err := db.Create(recruitment).Error; err != nil {
		t.Fatalf("seed recruitment: %v", err)
	}
	if err := db.Create(&model.NewbroRecruitmentEntry{
		RecruitmentID: recruitment.ID,
		QQ:            currentUser.QQ,
		EnteredAt:     now.Add(-24 * time.Hour),
		Status:        model.RecruitEntryStatusValid,
		MatchedUserID: currentUser.ID,
		RewardedAt:    &rewardedAt,
		WalletRefID:   &walletRefID,
	}).Error; err != nil {
		t.Fatalf("seed rewarded recruitment entry: %v", err)
	}

	svc := NewRecruitmentEntryService()
	status, err := svc.GetDirectReferralStatus(currentUser.ID, now)
	if err != nil {
		t.Fatalf("GetDirectReferralStatus() error = %v", err)
	}
	if status.ShowCard {
		t.Fatal("expected direct referral card to be hidden when the current user already has a rewarded recruit record")
	}
}

func TestRecruitmentEntryService_GetDirectReferralStatusRequiresProfileQQWhenCurrentUserQQMissing(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	currentUser := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-48 * time.Hour)},
		Nickname:  "recent-user",
		Role:      model.RoleUser,
	}
	if err := db.Create(currentUser).Error; err != nil {
		t.Fatalf("seed current user: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: currentUser.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed current user role: %v", err)
	}

	svc := NewRecruitmentEntryService()
	status, err := svc.GetDirectReferralStatus(currentUser.ID, now)
	if err != nil {
		t.Fatalf("GetDirectReferralStatus() error = %v", err)
	}
	if !status.ShowCard {
		t.Fatal("expected direct referral card to remain visible for an otherwise eligible user")
	}
	if !status.NeedsProfileQQ {
		t.Fatal("expected direct referral status to require the current user to save their own QQ first")
	}
	if _, err := svc.LookupDirectReferrer(currentUser.ID, "123456", now); err == nil {
		t.Fatal("expected direct referral check to require the current user profile QQ")
	}
	if _, err := svc.ConfirmDirectReferral(currentUser.ID, 999, now); err == nil {
		t.Fatal("expected direct referral confirmation to require the current user profile QQ")
	}
}

func TestRecruitmentEntryService_LookupDirectReferrerReturnsPrimaryCharacterSummary(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	currentUser := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-48 * time.Hour)},
		Nickname:  "recent-user",
		QQ:        "556677",
		Role:      model.RoleUser,
	}
	if err := db.Create(currentUser).Error; err != nil {
		t.Fatalf("seed current user: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: currentUser.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed current user role: %v", err)
	}

	referrer := &model.User{
		BaseModel:          model.BaseModel{CreatedAt: now.Add(-30 * 24 * time.Hour)},
		Nickname:           "referrer-user",
		QQ:                 "123456",
		Role:               model.RoleUser,
		PrimaryCharacterID: 900001,
	}
	if err := db.Create(referrer).Error; err != nil {
		t.Fatalf("seed referrer: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: referrer.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed referrer role: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		UserID:        referrer.ID,
		CharacterID:   referrer.PrimaryCharacterID,
		CharacterName: "Referrer Main",
	}).Error; err != nil {
		t.Fatalf("seed referrer main character: %v", err)
	}

	svc := NewRecruitmentEntryService()
	candidate, err := svc.LookupDirectReferrer(currentUser.ID, referrer.QQ, now)
	if err != nil {
		t.Fatalf("LookupDirectReferrer() error = %v", err)
	}
	if candidate == nil {
		t.Fatal("expected direct referrer candidate to be returned")
	}
	if candidate.UserID != referrer.ID {
		t.Fatalf("expected referrer user id %d, got %d", referrer.ID, candidate.UserID)
	}
	if candidate.Nickname != referrer.Nickname {
		t.Fatalf("expected referrer nickname %q, got %q", referrer.Nickname, candidate.Nickname)
	}
	if candidate.PrimaryCharacterID != referrer.PrimaryCharacterID {
		t.Fatalf("expected referrer primary character id %d, got %d", referrer.PrimaryCharacterID, candidate.PrimaryCharacterID)
	}
	if candidate.PrimaryCharacterName != "Referrer Main" {
		t.Fatalf("expected referrer primary character name %q, got %q", "Referrer Main", candidate.PrimaryCharacterName)
	}
}

func TestRecruitmentEntryService_ConfirmDirectReferralCreatesRewardedDirectReferralEntry(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	currentUser := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-48 * time.Hour)},
		Nickname:  "recent-user",
		QQ:        "556677",
		Role:      model.RoleUser,
	}
	if err := db.Create(currentUser).Error; err != nil {
		t.Fatalf("seed current user: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: currentUser.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed current user role: %v", err)
	}

	referrer := &model.User{
		BaseModel:          model.BaseModel{CreatedAt: now.Add(-30 * 24 * time.Hour)},
		Nickname:           "referrer-user",
		QQ:                 "123456",
		Role:               model.RoleUser,
		PrimaryCharacterID: 900001,
	}
	if err := db.Create(referrer).Error; err != nil {
		t.Fatalf("seed referrer: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: referrer.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed referrer role: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		UserID:        referrer.ID,
		CharacterID:   referrer.PrimaryCharacterID,
		CharacterName: "Referrer Main",
	}).Error; err != nil {
		t.Fatalf("seed referrer main character: %v", err)
	}

	svc := NewRecruitmentEntryService()
	candidate, err := svc.LookupDirectReferrer(currentUser.ID, referrer.QQ, now)
	if err != nil {
		t.Fatalf("LookupDirectReferrer() error = %v", err)
	}
	confirmed, err := svc.ConfirmDirectReferral(currentUser.ID, candidate.UserID, now)
	if err != nil {
		t.Fatalf("ConfirmDirectReferral() error = %v", err)
	}
	if confirmed == nil {
		t.Fatal("expected confirmed direct referrer summary to be returned")
	}
	if confirmed.UserID != referrer.ID {
		t.Fatalf("expected confirmed referrer id %d, got %d", referrer.ID, confirmed.UserID)
	}

	var recruitment model.NewbroRecruitment
	if err := db.First(&recruitment).Error; err != nil {
		t.Fatalf("load created recruitment: %v", err)
	}
	if recruitment.UserID != referrer.ID {
		t.Fatalf("expected direct referral recruitment to belong to referrer %d, got %d", referrer.ID, recruitment.UserID)
	}
	if recruitment.Source != model.RecruitmentSourceDirectReferral {
		t.Fatalf("expected recruitment source %q, got %q", model.RecruitmentSourceDirectReferral, recruitment.Source)
	}

	var entry model.NewbroRecruitmentEntry
	if err := db.First(&entry).Error; err != nil {
		t.Fatalf("load created recruitment entry: %v", err)
	}
	if entry.RecruitmentID != recruitment.ID {
		t.Fatalf("expected recruitment entry to attach to recruitment %d, got %d", recruitment.ID, entry.RecruitmentID)
	}
	if entry.QQ != currentUser.QQ {
		t.Fatalf("expected direct referral entry QQ %q, got %q", currentUser.QQ, entry.QQ)
	}
	if entry.Status != model.RecruitEntryStatusValid {
		t.Fatalf("expected direct referral entry status %q, got %q", model.RecruitEntryStatusValid, entry.Status)
	}
	if entry.Source != model.RecruitEntrySourceDirectReferral {
		t.Fatalf("expected direct referral entry source %q, got %q", model.RecruitEntrySourceDirectReferral, entry.Source)
	}
	if entry.MatchedUserID != currentUser.ID {
		t.Fatalf("expected matched user id %d, got %d", currentUser.ID, entry.MatchedUserID)
	}
	if entry.RewardedAt == nil || !entry.RewardedAt.Equal(now) {
		t.Fatalf("expected rewarded_at %v, got %v", now, entry.RewardedAt)
	}
	wantRefID := fmt.Sprintf("recruit_matched_user:%d", currentUser.ID)
	if entry.WalletRefID == nil || *entry.WalletRefID != wantRefID {
		t.Fatalf("expected wallet_ref_id %q, got %#v", wantRefID, entry.WalletRefID)
	}

	var txs []model.WalletTransaction
	if err := db.Order("id ASC").Find(&txs).Error; err != nil {
		t.Fatalf("list wallet transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected one wallet transaction, got %d", len(txs))
	}
	if txs[0].UserID != referrer.ID {
		t.Fatalf("expected referrer %d to receive wallet credit, got %d", referrer.ID, txs[0].UserID)
	}
	if txs[0].Amount != model.SysConfigDefaultNewbroRecruitRewardAmount {
		t.Fatalf("expected reward amount %v, got %v", model.SysConfigDefaultNewbroRecruitRewardAmount, txs[0].Amount)
	}
	if txs[0].RefType != model.WalletRefRecruitReward {
		t.Fatalf("expected wallet ref type %q, got %q", model.WalletRefRecruitReward, txs[0].RefType)
	}
	if txs[0].RefID != wantRefID {
		t.Fatalf("expected wallet ref id %q, got %q", wantRefID, txs[0].RefID)
	}
}

func TestRecruitmentEntryService_ConfirmDirectReferralAllowsPendingRecruitLinkEntry(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	currentUser := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-48 * time.Hour)},
		Nickname:  "recent-user",
		QQ:        "556677",
		Role:      model.RoleUser,
	}
	if err := db.Create(currentUser).Error; err != nil {
		t.Fatalf("seed current user: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: currentUser.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed current user role: %v", err)
	}

	referrer := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-30 * 24 * time.Hour)},
		Nickname:  "referrer-user",
		QQ:        "123456",
		Role:      model.RoleUser,
	}
	if err := db.Create(referrer).Error; err != nil {
		t.Fatalf("seed referrer: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: referrer.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed referrer role: %v", err)
	}

	pendingRecruitment := &model.NewbroRecruitment{UserID: 88, Code: "existing-link", GeneratedAt: now.Add(-72 * time.Hour)}
	if err := db.Create(pendingRecruitment).Error; err != nil {
		t.Fatalf("seed pending recruitment: %v", err)
	}
	if err := db.Create(&model.NewbroRecruitmentEntry{
		RecruitmentID: pendingRecruitment.ID,
		QQ:            currentUser.QQ,
		EnteredAt:     now.Add(-24 * time.Hour),
		Status:        model.RecruitEntryStatusOngoing,
		Source:        model.RecruitEntrySourceLink,
	}).Error; err != nil {
		t.Fatalf("seed pending recruitment entry: %v", err)
	}

	svc := NewRecruitmentEntryService()
	candidate, err := svc.LookupDirectReferrer(currentUser.ID, referrer.QQ, now)
	if err != nil {
		t.Fatalf("LookupDirectReferrer() error = %v", err)
	}
	if _, err := svc.ConfirmDirectReferral(currentUser.ID, candidate.UserID, now); err != nil {
		t.Fatalf("ConfirmDirectReferral() error = %v", err)
	}
}

func TestRecruitmentEntryService_LookupDirectReferrerRejectsAmbiguousQQ(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	currentUser := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-48 * time.Hour)},
		Nickname:  "recent-user",
		QQ:        "556677",
		Role:      model.RoleUser,
	}
	if err := db.Create(currentUser).Error; err != nil {
		t.Fatalf("seed current user: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: currentUser.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed current user role: %v", err)
	}

	referrerA := &model.User{BaseModel: model.BaseModel{CreatedAt: now.Add(-20 * 24 * time.Hour)}, Nickname: "ref-a", QQ: "123456", Role: model.RoleUser}
	referrerB := &model.User{BaseModel: model.BaseModel{CreatedAt: now.Add(-19 * 24 * time.Hour)}, Nickname: "ref-b", QQ: "234567", Role: model.RoleUser}
	if err := db.Create(referrerA).Error; err != nil {
		t.Fatalf("seed referrer A: %v", err)
	}
	if err := db.Create(referrerB).Error; err != nil {
		t.Fatalf("seed referrer B: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: referrerA.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed referrer A role: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: referrerB.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed referrer B role: %v", err)
	}
	if err := db.Model(&model.User{}).Where("id = ?", referrerB.ID).Update("qq", referrerA.QQ).Error; err != nil {
		t.Fatalf("make QQ ambiguous: %v", err)
	}

	svc := NewRecruitmentEntryService()
	if _, err := svc.LookupDirectReferrer(currentUser.ID, referrerA.QQ, now); err == nil {
		t.Fatal("expected lookup direct referral to reject an ambiguous QQ")
	}
}

func TestRecruitmentEntryService_ConfirmDirectReferralRejectsWhenCurrentUserQQClearsBeforeCommit(t *testing.T) {
	db := newRecruitmentEntryServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	currentUser := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-48 * time.Hour)},
		Nickname:  "recent-user",
		QQ:        "556677",
		Role:      model.RoleUser,
	}
	if err := db.Create(currentUser).Error; err != nil {
		t.Fatalf("seed current user: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: currentUser.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed current user role: %v", err)
	}

	referrer := &model.User{
		BaseModel: model.BaseModel{CreatedAt: now.Add(-20 * 24 * time.Hour)},
		Nickname:  "referrer-user",
		QQ:        "123456",
		Role:      model.RoleUser,
	}
	if err := db.Create(referrer).Error; err != nil {
		t.Fatalf("seed referrer: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: referrer.ID, RoleCode: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed referrer role: %v", err)
	}

	svc := NewRecruitmentEntryService()
	svc.userRepo = &scriptedRecruitmentEntryUserRepo{
		getByID: func(id uint) (*model.User, error) {
			switch id {
			case currentUser.ID:
				userCopy := *currentUser
				return &userCopy, nil
			case referrer.ID:
				userCopy := *referrer
				return &userCopy, nil
			default:
				return nil, gorm.ErrRecordNotFound
			}
		},
		listByQQ: func(qq string) ([]model.User, error) {
			if qq == referrer.QQ {
				return []model.User{*referrer}, nil
			}
			return nil, gorm.ErrRecordNotFound
		},
		getByIDForUpdateTx: func(_ *gorm.DB, id uint) (*model.User, error) {
			switch id {
			case currentUser.ID:
				userCopy := *currentUser
				userCopy.QQ = ""
				return &userCopy, nil
			case referrer.ID:
				userCopy := *referrer
				return &userCopy, nil
			default:
				return nil, gorm.ErrRecordNotFound
			}
		},
	}

	if _, err := svc.ConfirmDirectReferral(currentUser.ID, referrer.ID, now); err == nil {
		t.Fatal("expected confirmation to fail when the current user QQ is cleared before the transaction commits")
	}

	var entryCount int64
	if err := db.Model(&model.NewbroRecruitmentEntry{}).Count(&entryCount).Error; err != nil {
		t.Fatalf("count direct referral entries: %v", err)
	}
	if entryCount != 0 {
		t.Fatalf("expected no direct referral entries to be written, got %d", entryCount)
	}
}
