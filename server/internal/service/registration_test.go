package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"errors"
	"testing"
	"time"
)

func registrationRowCounts(t *testing.T, env *callbackTestEnv, users, chars, roles int64) {
	t.Helper()
	for _, check := range []struct {
		model any
		count int64
	}{{&model.User{}, users}, {&model.EveCharacter{}, chars}, {&model.UserRole{}, roles}} {
		var count int64
		if err := env.db.Unscoped().Model(check.model).Count(&count).Error; err != nil || count != check.count {
			t.Fatalf("%T count=%d want=%d err=%v", check.model, count, check.count, err)
		}
	}
}

func TestRegistrationLifecycle(t *testing.T) {
	replaceRegisteredScopes(t)
	env := newCallbackTestEnv(t, []string{"publicData"})
	ctx := context.Background()
	result, err := env.svc.HandleCallback(ctx, "code", env.storeState(t, stateData{}), "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if result.RegistrationTicket == "" || result.Token != "" || result.User != nil {
		t.Fatalf("unexpected callback result: %#v", result)
	}
	registrationRowCounts(t, env, 0, 0, 0)
	for _, nickname := range []string{"", " \t ", "a", " 中 ", "😀"} {
		if _, err := env.svc.CompleteRegistration(ctx, result.RegistrationTicket, RegistrationInput{Nickname: nickname, QQ: "123456"}, ""); err == nil {
			t.Fatalf("short nickname accepted: %q", nickname)
		}
		registrationRowCounts(t, env, 0, 0, 0)
	}
	info, err := env.svc.GetRegistration(ctx, result.RegistrationTicket)
	if err != nil || info.CharacterName != "Director" {
		t.Fatalf("info=%v err=%v", info, err)
	}
	if _, err := env.svc.CompleteRegistration(ctx, result.RegistrationTicket, RegistrationInput{Nickname: "Test", QQ: " \t "}, "127.0.0.1"); err == nil {
		t.Fatal("blank contact accepted")
	}
	registrationRowCounts(t, env, 0, 0, 0)
	completed, err := env.svc.CompleteRegistration(ctx, result.RegistrationTicket, RegistrationInput{Nickname: " 中文 ", QQ: " 123456 "}, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if completed.Token == "" || completed.User.QQ != "123456" || completed.User.Nickname != "中文" {
		t.Fatal("registration not completed")
	}
	char := env.reloadCharacter(t)
	if char.AccessToken == "" || char.RefreshToken != "rt-callback-new" || char.UserID != completed.User.ID {
		t.Fatal("EVE credentials were not preserved")
	}
	registrationRowCounts(t, env, 1, 1, 1)
	if _, err := env.svc.CompleteRegistration(ctx, result.RegistrationTicket, RegistrationInput{Nickname: "Replay", QQ: "234567"}, ""); err == nil {
		t.Fatal("ticket replay accepted")
	}
	registrationRowCounts(t, env, 1, 1, 1)
}

func TestRegistrationDuplicateContactUsesPrimaryCharacter(t *testing.T) {
	env := newCallbackTestEnv(t, []string{"publicData"})
	owner := model.User{Nickname: "Not the primary name", QQ: "123456", PrimaryCharacterID: 9001, Status: 1}
	if err := env.db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	if err := env.db.Create(&model.EveCharacter{CharacterID: 9001, CharacterName: "Primary Owner", UserID: owner.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := env.db.Create(&model.EveCharacter{CharacterID: 9002, CharacterName: "Alternate", UserID: owner.ID}).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	pending, err := env.svc.beginRegistration(ctx, &model.EveCharacter{CharacterID: callbackTestCharacterID, CharacterName: "Director", RefreshToken: "keep-me"}, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.svc.CompleteRegistration(ctx, pending.RegistrationTicket, RegistrationInput{Nickname: "New", QQ: "123456"}, "")
	var conflict *ContactConflictError
	if !errors.As(err, &conflict) || conflict.Contact != "qq" || conflict.PrimaryCharacterName != "Primary Owner" || conflict.PrimaryCharacterID != 9001 {
		t.Fatalf("wrong conflict: %v", err)
	}
	registrationRowCounts(t, env, 1, 2, 0)
	if _, err := env.svc.GetRegistration(ctx, pending.RegistrationTicket); err != nil {
		t.Fatal("conflict consumed pending ticket")
	}
}

func TestRegistrationExpiryAndTransactionRollback(t *testing.T) {
	env := newCallbackTestEnv(t, []string{"publicData"})
	ctx := context.Background()
	pending, err := env.svc.beginRegistration(ctx, &model.EveCharacter{CharacterID: callbackTestCharacterID, CharacterName: "Director"}, "")
	if err != nil {
		t.Fatal(err)
	}
	env.mr.FastForward(registrationTTL + time.Second)
	if _, err := env.svc.CompleteRegistration(ctx, pending.RegistrationTicket, RegistrationInput{Nickname: "New", DiscordID: "123456"}, ""); err == nil {
		t.Fatal("expired ticket accepted")
	}
	registrationRowCounts(t, env, 0, 0, 0)
	if err := env.db.Migrator().DropTable(&model.UserRole{}); err != nil {
		t.Fatal(err)
	}
	user := &model.User{Nickname: "Rollback", QQ: "123456"}
	err = repository.NewUserRepository().CreateRegistration(user, &model.EveCharacter{CharacterID: 9001, CharacterName: "Rollback"})
	if err == nil {
		t.Fatal("expected role creation failure")
	}
	var count int64
	env.db.Model(&model.User{}).Count(&count)
	if count != 0 {
		t.Fatal("partial user survived rollback")
	}
	env.db.Model(&model.EveCharacter{}).Count(&count)
	if count != 0 {
		t.Fatal("partial character survived rollback")
	}
}

func TestRegistrationSameCharacterDifferentTickets(t *testing.T) {
	env := newCallbackTestEnv(t, []string{"publicData"})
	ctx := context.Background()
	char := &model.EveCharacter{CharacterID: callbackTestCharacterID, CharacterName: "Director"}
	a, _ := env.svc.beginRegistration(ctx, char, "")
	b, _ := env.svc.beginRegistration(ctx, char, "")
	if _, err := env.svc.CompleteRegistration(ctx, a.RegistrationTicket, RegistrationInput{Nickname: "First", QQ: "123456"}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := env.svc.CompleteRegistration(ctx, b.RegistrationTicket, RegistrationInput{Nickname: "Second", QQ: "234567"}, ""); !errors.Is(err, repository.ErrCharacterAlreadyRegistered) {
		t.Fatalf("got %v", err)
	}
	registrationRowCounts(t, env, 1, 1, 1)
}

func TestContactRequiredForAllProfileWriters(t *testing.T) {
	for _, requireComplete := range []bool{true, false} {
		blank := " \t "
		_, _, err := buildUserPatchUpdates(&model.User{Nickname: "Test", QQ: "123456"}, UserPatch{QQ: &blank}, requireComplete)
		if err == nil {
			t.Fatalf("last contact removal accepted requireComplete=%v", requireComplete)
		}
	}
}

func TestProfileContactConflictPreservesAccount(t *testing.T) {
	env := newCallbackTestEnv(t, []string{"publicData"})
	owner := &model.User{Nickname: "Nickname", QQ: "123456", PrimaryCharacterID: 9001}
	current := &model.User{Nickname: "Current", DiscordID: "987654"}
	if err := env.db.Create(owner).Error; err != nil {
		t.Fatal(err)
	}
	if err := env.db.Create(current).Error; err != nil {
		t.Fatal(err)
	}
	if err := env.db.Create(&model.EveCharacter{CharacterID: 9001, CharacterName: "Primary Owner", UserID: owner.ID}).Error; err != nil {
		t.Fatal(err)
	}
	qq := owner.QQ
	_, err := NewUserService().UpdateCurrentProfile(current.ID, UserPatch{QQ: &qq})
	var conflict *ContactConflictError
	if !errors.As(err, &conflict) || conflict.PrimaryCharacterName != "Primary Owner" {
		t.Fatalf("wrong conflict: %v", err)
	}
	var got model.User
	if err := env.db.First(&got, current.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.QQ != "" || got.DiscordID != "987654" {
		t.Fatal("conflict changed profile")
	}
}

func TestOrphanSSORequiresRegistration(t *testing.T) {
	replaceRegisteredScopes(t)
	env := newCallbackTestEnv(t, []string{"publicData"})
	env.seedUserAndCharacter(t, "publicData stale-admin-scope")
	if err := env.db.Unscoped().Delete(&model.User{}, callbackTestUserID).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	result, err := env.svc.HandleCallback(ctx, "code", env.storeState(t, stateData{}), "")
	if err != nil || result.RegistrationTicket == "" || result.Token != "" {
		t.Fatalf("result=%v err=%v", result, err)
	}
	registrationRowCounts(t, env, 0, 1, 0)
	completed, err := env.svc.CompleteRegistration(ctx, result.RegistrationTicket, RegistrationInput{Nickname: "Recovered", DiscordID: "123456"}, "")
	if err != nil {
		t.Fatal(err)
	}
	char := env.reloadCharacter(t)
	if char.UserID != completed.User.ID || char.Scopes != "publicData" {
		t.Fatal("orphan authorization not replaced with verified scopes")
	}
	registrationRowCounts(t, env, 1, 1, 1)
}

func TestSSORejectsExpiredState(t *testing.T) {
	env := newCallbackTestEnv(t, []string{"publicData"})
	if _, err := env.svc.HandleCallback(context.Background(), "code", "missing-state", ""); err == nil {
		t.Fatal("missing state accepted")
	}
	registrationRowCounts(t, env, 0, 0, 0)
}
