package service

import (
	"amiya-eden/internal/model"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestQQOfficialRejoinFencesOldAuthorizationEvenWithoutExit(t *testing.T) {
	s, _, now := qqFixture(t)
	qqEnable(t, s)
	ctx := t.Context()
	group := qqGroupFixture(t, s, 1)
	if err := s.locked(ctx, nil, func(tx *gorm.DB, _ *model.QQOfficialApp, at time.Time) error {
		before := at.Add(-time.Minute)
		group.MembershipAt, group.PushAt, group.PushState = &before, &before, qqAllowedPush
		group.TestVerified, group.NotificationEnabled = true, true
		return s.repo.SaveGroup(ctx, tx, &group)
	}); err != nil {
		t.Fatal(err)
	}
	task := qqSubmitTest(t, s, group)
	*now = now.Add(time.Second)
	if err := s.locked(ctx, nil, func(tx *gorm.DB, _ *model.QQOfficialApp, at time.Time) error {
		if err := s.applyEvent(ctx, tx, model.QQOfficialEvent{Kind: "GROUP_ADD_ROBOT", GroupOpenID: group.OpenID, EventAt: at}, at); err != nil {
			return err
		}
		return s.applyEvent(ctx, tx, model.QQOfficialEvent{Kind: "GROUP_MSG_RECEIVE", GroupOpenID: group.OpenID, EventAt: at}, at)
	}); err != nil {
		t.Fatal(err)
	}
	current, err := s.repo.Group(ctx, nil, s.appID, group.ID)
	if err != nil || !current.Present || current.Authorized || current.NotificationEnabled || current.TestVerified || current.Generation == group.Generation {
		t.Fatalf("unexpected membership %+v: %v", current, err)
	}
	result, _, err := s.Task(ctx, task.ID)
	if err != nil || result.Status != model.QQCancelled {
		t.Fatalf("old task status=%s err=%v", result.Status, err)
	}
}
