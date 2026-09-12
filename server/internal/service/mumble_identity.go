package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"
)

const (
	mumblePasswordMemoryKiB uint32 = 64 * 1024
	mumblePasswordTime      uint32 = 3
	mumblePasswordThreads   uint8  = 1
	mumblePasswordKeyLen    uint32 = 32
)

var (
	ErrMumbleCredentialNotFound = errors.New("mumble 凭据不存在")
	ErrMumbleIdentityDenied     = errors.New("mumble 身份不具备登录资格")
)

type MumbleIdentityService struct {
	identityRepo *repository.MumbleIdentityRepository
	userRepo     *repository.UserRepository
	charRepo     *repository.EveCharacterRepository
	roleRepo     *repository.RoleRepository
	cfgRepo      *repository.SysConfigRepository
	tickerRepo   *repository.EntityTickerCacheRepository
	auditSvc     *AuditService
}

// MumbleCredentialStatus is the user-facing credential payload. The stable
// Mumble user id is protocol-only and must never appear here; ServerAddress
// and ServerPort are display-only connection hints from system config.
type MumbleCredentialStatus struct {
	Created           bool       `json:"created"`
	Enabled           bool       `json:"enabled"`
	ServerAddress     string     `json:"server_address,omitempty"`
	ServerPort        int        `json:"server_port,omitempty"`
	CredentialVersion uint       `json:"credential_version,omitempty"`
	PasswordRotatedAt *time.Time `json:"password_rotated_at,omitempty"`
}

type MumbleClaims struct {
	Eligible        bool     `json:"eligible"`
	StableUserID    uint32   `json:"user_id,omitempty"`
	Name            string   `json:"name,omitempty"`
	Groups          []string `json:"groups,omitempty"`
	IdentityVersion uint     `json:"identity_version,omitempty"`
	PolicyVersion   uint64   `json:"policy_version,omitempty"`
}

func NewMumbleIdentityService() *MumbleIdentityService {
	return &MumbleIdentityService{
		identityRepo: repository.NewMumbleIdentityRepository(), userRepo: repository.NewUserRepository(),
		charRepo: repository.NewEveCharacterRepository(), roleRepo: repository.NewRoleRepository(),
		cfgRepo: repository.NewSysConfigRepository(), tickerRepo: repository.NewEntityTickerCacheRepository(), auditSvc: NewAuditService(),
	}
}

func (s *MumbleIdentityService) GetCredentialStatus(userID uint) (MumbleCredentialStatus, error) {
	identity, err := s.identityRepo.GetBySeatUserID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.withConnectionInfo(MumbleCredentialStatus{}), nil
	}
	if err != nil {
		return MumbleCredentialStatus{}, err
	}
	return s.withConnectionInfo(credentialStatus(identity)), nil
}

// CreateCredential creates the only copy of a Mumble app password. The secret
// is returned once and is never written to storage, audit records, or logs.
func (s *MumbleIdentityService) CreateCredential(userID uint) (MumbleCredentialStatus, string, error) {
	if _, err := s.userRepo.GetByID(userID); err != nil {
		return MumbleCredentialStatus{}, "", err
	}
	if existing, err := s.identityRepo.GetBySeatUserID(userID); err == nil && existing.CredentialHash != "" && existing.CredentialEnabled {
		return MumbleCredentialStatus{}, "", errors.New("mumble 凭据已存在，请使用轮换功能")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return MumbleCredentialStatus{}, "", err
	}
	secret, hash, err := newMumblePassword()
	if err != nil {
		return MumbleCredentialStatus{}, "", err
	}
	now := time.Now()
	identity, err := s.identityRepo.GetBySeatUserID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		identity = &model.MumbleIdentity{SeatUserID: userID, CredentialHash: hash, CredentialEnabled: true, CredentialVersion: 1, IdentityVersion: 1, PasswordRotatedAt: &now}
		if err := s.identityRepo.Create(identity); err != nil {
			return MumbleCredentialStatus{}, "", err
		}
	} else if err != nil {
		return MumbleCredentialStatus{}, "", err
	} else {
		version := identity.CredentialVersion + 1
		if err := s.identityRepo.UpdateCredential(userID, hash, true, version, now); err != nil {
			return MumbleCredentialStatus{}, "", err
		}
		identity.CredentialEnabled, identity.CredentialVersion, identity.IdentityVersion, identity.PasswordRotatedAt = true, version, version, &now
	}
	s.recordAudit("mumble_credential_create", userID, model.AuditResultSuccess, nil)
	return s.withConnectionInfo(credentialStatus(identity)), secret, nil
}

func (s *MumbleIdentityService) RotateCredential(userID uint) (MumbleCredentialStatus, string, error) {
	identity, err := s.identityRepo.GetBySeatUserID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return MumbleCredentialStatus{}, "", ErrMumbleCredentialNotFound
	}
	if err != nil {
		return MumbleCredentialStatus{}, "", err
	}
	secret, hash, err := newMumblePassword()
	if err != nil {
		return MumbleCredentialStatus{}, "", err
	}
	now, version := time.Now(), identity.CredentialVersion+1
	if err := s.identityRepo.UpdateCredential(userID, hash, true, version, now); err != nil {
		return MumbleCredentialStatus{}, "", err
	}
	identity.CredentialEnabled, identity.CredentialVersion, identity.IdentityVersion, identity.PasswordRotatedAt = true, version, version, &now
	s.recordAudit("mumble_credential_rotate", userID, model.AuditResultSuccess, nil)
	NotifyMumbleIdentityChanged(global.BackgroundContext(), userID)
	return s.withConnectionInfo(credentialStatus(identity)), secret, nil
}

func (s *MumbleIdentityService) RevokeCredential(userID uint) error {
	identity, err := s.identityRepo.GetBySeatUserID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrMumbleCredentialNotFound
	}
	if err != nil {
		return err
	}
	if err := s.identityRepo.Revoke(userID, identity.CredentialVersion+1); err != nil {
		return err
	}
	s.recordAudit("mumble_credential_revoke", userID, model.AuditResultSuccess, nil)
	NotifyMumbleIdentityChanged(global.BackgroundContext(), userID)
	return nil
}

// Authenticate validates the Mumble-only credential and resolves current claims.
// All credential and eligibility failures are intentionally returned as one denial.
func (s *MumbleIdentityService) Authenticate(username, password string) (MumbleClaims, error) {
	claims, identity, err := s.resolveByPrimaryCharacterName(username)
	if err != nil || identity == nil || !identity.CredentialEnabled || !verifyMumblePassword(identity.CredentialHash, password) || !claims.Eligible {
		s.recordAudit("mumble_login_deny", 0, model.AuditResultFailed, map[string]any{"reason": "invalid_credentials_or_entitlement"})
		return MumbleClaims{}, ErrMumbleIdentityDenied
	}
	now := time.Now()
	_ = s.identityRepo.TouchLastUsed(identity.ID, now)
	s.recordAudit("mumble_login_allow", identity.SeatUserID, model.AuditResultSuccess, nil)
	return claims, nil
}

// Resolve returns current claims by the externally visible stable identity.
// It is intentionally credential-free for periodic session revalidation.
func (s *MumbleIdentityService) Resolve(stableUserID uint32) (MumbleClaims, error) {
	if stableUserID == 0 {
		return MumbleClaims{}, ErrMumbleIdentityDenied
	}
	identity, err := s.identityRepo.GetByStableUserID(stableUserID)
	if err != nil {
		return MumbleClaims{}, err
	}
	claims, err := s.resolveBySeatUserID(identity.SeatUserID, identity)
	if err != nil {
		return MumbleClaims{}, err
	}
	return claims, nil
}

// ResolveByName resolves a current primary-character canonical name without
// accepting a password. It is used by protocol identity lookups and reuses the
// same eligibility calculation as authentication.
func (s *MumbleIdentityService) ResolveByName(name string) (MumbleClaims, error) {
	claims, _, err := s.resolveByPrimaryCharacterName(name)
	return claims, err
}

func (s *MumbleIdentityService) resolveByPrimaryCharacterName(username string) (MumbleClaims, *model.MumbleIdentity, error) {
	name := strings.TrimSpace(username)
	if name == "" || name == "SuperUser" {
		return MumbleClaims{}, nil, ErrMumbleIdentityDenied
	}
	char, err := s.charRepo.GetByCharacterName(name)
	if err != nil {
		return MumbleClaims{}, nil, err
	}
	user, err := s.userRepo.GetByPrimaryCharacterID(char.CharacterID)
	if err != nil || user.ID != char.UserID {
		return MumbleClaims{}, nil, ErrMumbleIdentityDenied
	}
	identity, err := s.identityRepo.GetBySeatUserID(user.ID)
	if err != nil {
		return MumbleClaims{}, nil, err
	}
	claims, err := s.resolveBySeatUserID(user.ID, identity)
	return claims, identity, err
}

func (s *MumbleIdentityService) resolveBySeatUserID(userID uint, identity *model.MumbleIdentity) (MumbleClaims, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return MumbleClaims{}, err
	}
	char, err := s.charRepo.GetByCharacterID(user.PrimaryCharacterID)
	if err != nil || user.PrimaryCharacterID == 0 || char.UserID != user.ID {
		return MumbleClaims{Eligible: false}, nil
	}
	roles, err := s.roleRepo.GetUserRoleCodes(user.ID)
	if err != nil {
		return MumbleClaims{}, err
	}
	roles = model.NormalizeRoleCodes(roles, user.Role)
	eligible := user.Status == 1 && model.HasNonGuestRole(roles) && identity != nil && identity.CredentialEnabled && identity.CredentialHash != ""
	if !eligible {
		return MumbleClaims{Eligible: false, StableUserID: identity.StableMumbleUserID(), IdentityVersion: identity.IdentityVersion, PolicyVersion: mumblePolicyVersion(user, char, roles, identity)}, nil
	}
	groups := []string{"fuxi_authenticated"}
	for _, role := range roles {
		if model.IsValidRoleCode(role) {
			groups = append(groups, "fuxi_role_"+role)
		}
	}
	displayName, err := s.formatMumbleDisplayName(user.Nickname, char, roles)
	if err != nil {
		return MumbleClaims{}, err
	}
	return MumbleClaims{Eligible: true, StableUserID: identity.StableMumbleUserID(), Name: displayName, Groups: groups, IdentityVersion: identity.IdentityVersion, PolicyVersion: mumblePolicyVersion(user, char, roles, identity)}, nil
}

func (s *MumbleIdentityService) formatMumbleDisplayName(nickname string, char *model.EveCharacter, roles []string) (string, error) {
	template := defaultMumbleDisplayNameTemplate
	if s.cfgRepo != nil {
		template = strings.TrimSpace(s.cfgRepo.GetString(model.SysConfigMumbleDisplayNameTemplate, defaultMumbleDisplayNameTemplate))
	}
	if template == "" {
		template = defaultMumbleDisplayNameTemplate
	}
	corporationTicker, allianceTicker := "", ""
	if strings.Contains(template, "{corporation_ticker}") || strings.Contains(template, "{alliance_ticker}") {
		var err error
		var ready bool
		corporationTicker, allianceTicker, ready, err = s.resolveMumbleAffiliationTickers(char)
		if err != nil {
			return "", err
		}
		if !ready {
			// 归属任务尚未写入快照时不让显示模板影响语音登录资格；下次
			// 周期身份重验会在快照就绪后推送完整昵称。
			return char.CharacterName, nil
		}
	}
	replacements := strings.NewReplacer(
		"{alliance_ticker}", allianceTicker,
		"{corporation_ticker}", corporationTicker,
		"{nickname}", nickname,
		"{character_name}", char.CharacterName,
		"{roles}", strings.Join(roles, ","),
	)
	displayName := strings.TrimSpace(replacements.Replace(template))
	if displayName == "" || displayName == "SuperUser" || len(displayName) > 128 {
		return "", ErrMumbleIdentityDenied
	}
	return displayName, nil
}

func (s *MumbleIdentityService) resolveMumbleAffiliationTickers(char *model.EveCharacter) (string, string, bool, error) {
	if char == nil {
		return "", "", false, ErrMumbleIdentityDenied
	}
	resolve := func(kind string, entityID int64) (string, bool, error) {
		if entityID <= 0 {
			return "", true, nil
		}
		if s.tickerRepo == nil {
			return "", false, ErrMumbleIdentityDenied
		}
		ticker, found, err := s.tickerRepo.GetFreshTicker(kind, entityID, time.Now())
		if err != nil {
			return "", false, err
		}
		return ticker, found, nil
	}
	corporationTicker, corporationReady, err := resolve(model.EntityTickerTypeCorporation, char.CorporationID)
	if err != nil {
		return "", "", false, err
	}
	allianceID := int64(0)
	if char.AllianceID != nil {
		allianceID = *char.AllianceID
	}
	allianceTicker, allianceReady, err := resolve(model.EntityTickerTypeAlliance, allianceID)
	if err != nil {
		return "", "", false, err
	}
	return corporationTicker, allianceTicker, corporationReady && allianceReady, nil
}

func credentialStatus(identity *model.MumbleIdentity) MumbleCredentialStatus {
	if identity == nil {
		return MumbleCredentialStatus{}
	}
	return MumbleCredentialStatus{Created: true, Enabled: identity.CredentialEnabled && identity.CredentialHash != "", CredentialVersion: identity.CredentialVersion, PasswordRotatedAt: identity.PasswordRotatedAt}
}

// withConnectionInfo fills the display-only public connection fields from
// system config so every user-facing credential response stays self-sufficient.
func (s *MumbleIdentityService) withConnectionInfo(status MumbleCredentialStatus) MumbleCredentialStatus {
	if s.cfgRepo == nil {
		return status
	}
	status.ServerAddress = strings.TrimSpace(s.cfgRepo.GetString(model.SysConfigMumblePublicAddress, ""))
	status.ServerPort = s.cfgRepo.GetInt(model.SysConfigMumblePublicPort, 0)
	return status
}

func (s *MumbleIdentityService) recordAudit(action string, targetUserID uint, result string, details map[string]any) {
	if s.auditSvc != nil {
		_ = s.auditSvc.RecordEvent(global.BackgroundContext(), AuditRecordInput{Category: "mumble", Action: action, TargetUserID: targetUserID, ResourceType: "mumble_identity", ResourceID: fmt.Sprintf("%d", targetUserID), Result: result, Details: details})
	}
}

func mumblePolicyVersion(user *model.User, char *model.EveCharacter, roles []string, identity *model.MumbleIdentity) uint64 {
	h := fnv.New64a()
	_, _ = fmt.Fprintf(h, "%d|%d|%d|%s|%d", user.Status, user.PrimaryCharacterID, char.UserID, strings.Join(roles, ","), identity.CredentialVersion)
	return h.Sum64()
}

func newMumblePassword() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}
	key := argon2.IDKey([]byte(secret), salt, mumblePasswordTime, mumblePasswordMemoryKiB, mumblePasswordThreads, mumblePasswordKeyLen)
	hash := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", mumblePasswordMemoryKiB, mumblePasswordTime, mumblePasswordThreads, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
	return secret, hash, nil
}

func verifyMumblePassword(encoded, secret string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || secret == "" {
		return false
	}
	var memory, iterations uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil || memory == 0 || iterations == 0 || threads == 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(secret), salt, iterations, memory, threads, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
