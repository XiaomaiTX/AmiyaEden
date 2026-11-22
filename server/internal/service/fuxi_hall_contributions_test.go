package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"errors"
	"testing"
	"time"
)

func TestFuxiHallContributionSnapshotAndAdditiveRecruitment(t *testing.T) {
	useFuxiHallServiceTestDB(t)
	create := func(v interface{}) {
		t.Helper()
		if err := global.DB.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	user := model.User{PrimaryCharacterID: 9100}
	create(&user)
	create(&model.EveCharacter{CharacterID: 9100, UserID: user.ID})
	now := time.Now().UTC()
	create(&model.Fleet{ID: "ended", FCUserID: user.ID, StartAt: now.Add(-2 * time.Hour), EndAt: now.Add(-time.Hour)})
	create(&model.Fleet{ID: "future", FCUserID: user.ID, StartAt: now, EndAt: now.Add(time.Hour)})
	create(&model.Fleet{ID: "deleted", FCUserID: user.ID, EndAt: now.Add(-time.Hour), DeletedAt: &now})
	create(&model.WelfareApplication{WelfareID: 1, ReviewedBy: user.ID, Status: model.WelfareAppStatusDelivered})
	create(&model.WelfareApplication{WelfareID: 1, ReviewedBy: user.ID, Status: "pending"})
	create(&model.ShopOrder{OrderNo: "delivered", ReviewedBy: &user.ID, Status: model.OrderStatusDelivered})
	create(&model.ShopOrder{OrderNo: "pending", ReviewedBy: &user.ID, Status: "pending"})
	create(&model.SrpApplication{CharacterID: 1, KillmailID: 1, PaidBy: &user.ID, PayoutStatus: model.SrpPayoutPaid})
	create(&model.SrpApplication{CharacterID: 1, KillmailID: 2, PaidBy: &user.ID, PayoutStatus: model.SrpPayoutNotPaid})
	// The same newcomer contributes once for invitation AND once for processing.
	create(&model.AdmissionApplication{UserID: 200, Kind: model.AdmissionKindNewMember, Status: model.AdmissionStatusInvited, HandledBy: &user.ID})
	create(&model.AdmissionApplication{UserID: 201, Kind: model.AdmissionKindAlt, Status: model.AdmissionStatusInvited, HandledBy: &user.ID})
	create(&model.AdmissionApplication{UserID: 202, Kind: model.AdmissionKindAlt, Status: model.AdmissionStatusRejected, HandledBy: &user.ID})
	link := model.NewbroRecruitment{UserID: user.ID, Code: "referral"}
	create(&link)
	create(&model.NewbroRecruitmentEntry{RecruitmentID: link.ID, QQ: "1", MatchedUserID: 200, Status: model.RecruitEntryStatusValid})
	create(&model.NewbroRecruitmentEntry{RecruitmentID: link.ID, QQ: "2", MatchedUserID: 200, Status: model.RecruitEntryStatusValid})
	create(&model.NewbroRecruitmentEntry{RecruitmentID: link.ID, QQ: "3", MatchedUserID: 203, Status: model.RecruitEntryStatusOngoing})
	create(&model.MentorMenteeRelationship{MentorUserID: user.ID, MenteeUserID: 200, Status: model.MentorRelationStatusActive, RespondedAt: &now})
	create(&model.MentorMenteeRelationship{MentorUserID: user.ID, MenteeUserID: 200, Status: model.MentorRelationStatusGraduated, RespondedAt: &now})
	create(&model.MentorMenteeRelationship{MentorUserID: user.ID, MenteeUserID: 201, Status: model.MentorRelationStatusPending})
	card := model.FuxiHallCard{PageKey: string(model.FuxiHallPageFame), MainCharacterID: 9100, WelfareDeliveryOffset: 10, FleetLedOffset: 2, SRPOffset: 3, RecruitmentOffset: 4, MentoringOffset: 5}
	create(&card)
	svc := NewFuxiHallService()
	if err := svc.RefreshContributions(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	want := model.FuxiHallContributions{Fleets: 3, Welfare: 12, SRP: 4, Recruitment: 7, Mentoring: 6}
	assertSnapshot := func() {
		t.Helper()
		hall, err := svc.PublicHall(user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(hall.Cards) != 1 || hall.Cards[0].Contributions != want || hall.Cards[0].ContributionsUpdatedAt == nil {
			t.Fatalf("snapshot: %+v", hall.Cards)
		}
	}
	assertSnapshot()
	// Public reads use the stored snapshot even when source records change.
	create(&model.SrpApplication{CharacterID: 1, KillmailID: 3, PaidBy: &user.ID, PayoutStatus: model.SrpPayoutPaid})
	assertSnapshot()
	if err := svc.RefreshContributions(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	want.SRP++
	assertSnapshot()
	// A failed aggregate and cancellation both preserve the complete previous snapshot.
	if err := global.DB.Migrator().DropTable(&model.ShopOrder{}); err != nil {
		t.Fatal(err)
	}
	if err := svc.RefreshContributions(context.Background(), 0); err == nil {
		t.Fatal("expected source failure")
	}
	assertSnapshot()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := svc.RefreshContributions(ctx, 0); err == nil {
		t.Fatal("expected cancellation")
	}
	assertSnapshot()
}

func TestFuxiHallRefreshRejectsConcurrentRun(t *testing.T) {
	hallContributionRefresh.Lock()
	defer hallContributionRefresh.Unlock()
	err := NewFuxiHallService().RefreshContributions(context.Background(), 0)
	if !errors.Is(err, ErrFuxiHallRefreshRunning) || !IsUserVisibleError(err) {
		t.Fatalf("expected ErrFuxiHallRefreshRunning user-visible error, got: %v", err)
	}
}
