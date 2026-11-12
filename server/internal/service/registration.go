package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/pkg/cache"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const registrationPrefix = "sso:registration:"
const registrationTTL = 15 * time.Minute

type pendingRegistration struct {
	Character    model.EveCharacter `json:"character"`
	RedirectURL  string             `json:"redirect_url"`
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token"`
}

type RegistrationInfo struct {
	CharacterID   int64  `json:"character_id"`
	CharacterName string `json:"character_name"`
}

type RegistrationInput struct {
	Nickname  string `json:"nickname"`
	QQ        string `json:"qq"`
	DiscordID string `json:"discord_id"`
}

type ContactConflictError struct {
	Contact              string `json:"contact"`
	PrimaryCharacterID   int64  `json:"primary_character_id"`
	PrimaryCharacterName string `json:"primary_character_name"`
}

func (e *ContactConflictError) Error() string { return "该联系方式已被其他用户使用" }

var (
	errContactCheckFailed = errors.New("联系方式校验失败，请稍后重试")
	errUserSaveFailed     = errors.New("保存失败，请稍后重试")
)

func (s *UserService) contactWriteError(user *model.User, writeErr error) error {
	var sqlErr interface{ SQLState() string }
	if errors.As(writeErr, &sqlErr) {
		if sqlErr.SQLState() == "23514" {
			return errors.New("请至少填写 QQ 号码或 Discord ID")
		}
		if sqlErr.SQLState() == "23505" {
			if conflict := s.validateUniqueContacts(user); conflict != nil {
				return conflict
			}
		}
	}
	global.Logger.Error("保存用户资料失败", zap.Uint("user_id", user.ID), zap.Error(writeErr))
	return errUserSaveFailed
}

func (s *UserService) contactConflict(owner *model.User, contact string) error {
	name := fmt.Sprintf("Capsuleer#%d", owner.PrimaryCharacterID)
	character, err := s.charRepo.GetByCharacterID(owner.PrimaryCharacterID)
	if err == nil && character.UserID == owner.ID && character.CharacterName != "" {
		name = character.CharacterName
	}
	return &ContactConflictError{Contact: contact, PrimaryCharacterID: owner.PrimaryCharacterID, PrimaryCharacterName: name}
}

func (s *EveSSOService) beginRegistration(ctx context.Context, character *model.EveCharacter, redirectURL string) (*CallbackResult, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	ticket := hex.EncodeToString(b)
	if err := cache.Set(ctx, registrationPrefix+ticket, pendingRegistration{Character: *character, RedirectURL: redirectURL, AccessToken: character.AccessToken, RefreshToken: character.RefreshToken}, registrationTTL); err != nil {
		return nil, errors.New("暂时无法开始注册，请稍后重新登录")
	}
	return &CallbackResult{RegistrationTicket: ticket, RedirectURL: redirectURL}, nil
}

func loadRegistration(ctx context.Context, ticket string) (*pendingRegistration, error) {
	if len(ticket) != 64 {
		return nil, errors.New("注册会话已失效，请重新登录")
	}
	if _, err := hex.DecodeString(ticket); err != nil {
		return nil, errors.New("注册会话已失效，请重新登录")
	}
	var pending pendingRegistration
	if err := cache.Get(ctx, registrationPrefix+ticket, &pending); err != nil {
		return nil, errors.New("注册会话已失效，请重新登录")
	}
	return &pending, nil
}

func (s *EveSSOService) GetRegistration(ctx context.Context, ticket string) (*RegistrationInfo, error) {
	pending, err := loadRegistration(ctx, ticket)
	if err != nil {
		return nil, err
	}
	return &RegistrationInfo{CharacterID: pending.Character.CharacterID, CharacterName: pending.Character.CharacterName}, nil
}

func (s *EveSSOService) CompleteRegistration(ctx context.Context, ticket string, input RegistrationInput, clientIP string) (*CallbackResult, error) {
	if _, err := loadRegistration(ctx, ticket); err != nil {
		return nil, err
	}
	lockKey := registrationPrefix + ticket + ":lock"
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	lockResult, err := global.Redis.SetArgs(ctx, lockKey, lockValue, redis.SetArgs{Mode: "NX", TTL: 30 * time.Second}).Result()
	if err != nil || lockResult != "OK" {
		return nil, errors.New("注册正在处理，请稍后重试")
	}
	defer global.Redis.Eval(context.Background(), `if redis.call('get', KEYS[1]) == ARGV[1] then return redis.call('del', KEYS[1]) end return 0`, []string{lockKey}, lockValue)
	// Recheck after acquiring the lock; another request may already have consumed it.
	pending, err := loadRegistration(ctx, ticket)
	if err != nil {
		return nil, err
	}
	user := buildDefaultSSOUser(pending.Character.CharacterID, clientIP, time.Now(), model.RoleGuest)
	nickname, qq, discord := strings.TrimSpace(input.Nickname), strings.TrimSpace(input.QQ), strings.TrimSpace(input.DiscordID)
	if utf8.RuneCountInString(nickname) < 2 {
		return nil, errors.New("昵称至少 2 个字符")
	}
	user, _, err = buildUserPatchUpdates(user, UserPatch{Nickname: &nickname, QQ: &qq, DiscordID: &discord}, true)
	if err != nil {
		return nil, err
	}
	userSvc := NewUserService()
	if err := userSvc.validateUniqueContacts(user); err != nil {
		return nil, err
	}
	role, affiliation := s.resolveInitialSSOState(ctx, pending.Character.CharacterID)
	for _, id := range global.Config.App.SuperAdmins {
		if id == pending.Character.CharacterID {
			role = model.RoleSuperAdmin
			break
		}
	}
	user.Role = role
	character := pending.Character
	character.AccessToken = pending.AccessToken
	character.RefreshToken = pending.RefreshToken
	character.ID = 0
	character.UserID = 0
	applyAffiliationToCharacter(&character, affiliation)
	if err := s.userRepo.CreateRegistration(user, &character); err != nil {
		// A concurrent account may have claimed the contact since the preflight check.
		user.ID = 0
		var conflict *ContactConflictError
		if errors.As(userSvc.validateUniqueContacts(user), &conflict) {
			return nil, conflict
		}
		if errors.Is(err, repository.ErrCharacterAlreadyRegistered) {
			return nil, err
		}
		global.Logger.Error("注册事务失败", zap.Error(err))
		return nil, errors.New("注册失败，请稍后重试")
	}
	_ = cache.Del(ctx, registrationPrefix+ticket)
	if OnNewCharacterSyncFunc != nil {
		OnNewCharacterSyncFunc(character.CharacterID, user.ID)
	}
	if OnNewCharacterFunc != nil {
		OnNewCharacterFunc(character.CharacterID, user.ID)
	}
	token, user, err := s.loadUserAndGenerateToken(user.ID)
	if err != nil {
		global.Logger.Error("注册后签发令牌失败", zap.Int64("character_id", character.CharacterID), zap.Error(err))
		return nil, errors.New("注册已完成，请重新登录")
	}
	return &CallbackResult{Token: token, User: user, Character: &character, RedirectURL: pending.RedirectURL}, nil
}
