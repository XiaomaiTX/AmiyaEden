package service

import (
	"amiya-eden/config"
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/utils"
	"amiya-eden/pkg/cache"
	"amiya-eden/pkg/eve"
	"amiya-eden/pkg/eve/esi"
	"amiya-eden/pkg/jwt"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────
//  Scope 注册机制（供其他模块使用）
// ─────────────────────────────────────────────

// RegisteredScope 其他模块注册的 ESI Scope
type RegisteredScope struct {
	Module      string `json:"module"`      // 注册模块名
	Scope       string `json:"scope"`       // ESI scope 字符串
	Description string `json:"description"` // 描述（向用户展示）
	Required    bool   `json:"required"`    // 是否为必选 scope
}

var (
	scopeMu          sync.RWMutex
	registeredScopes []RegisteredScope
)

// RegisterScope 供其他模块调用，注册所需的 ESI scope
//
//	module:      模块标识，如 "killmail"
//	scope:       ESI scope，如 "esi-killmails.read_killmails.v1"
//	description: 向用户展示的说明
//	required:    是否为必选（false 时用户可选择跳过）
func RegisterScope(module, scope, description string, required bool) {
	scopeMu.Lock()
	defer scopeMu.Unlock()
	registeredScopes = append(registeredScopes, RegisteredScope{
		Module:      module,
		Scope:       scope,
		Description: description,
		Required:    required,
	})
}

// GetRegisteredScopes 获取所有已注册的 scope 列表
func GetRegisteredScopes() []RegisteredScope {
	scopeMu.RLock()
	defer scopeMu.RUnlock()
	result := make([]RegisteredScope, len(registeredScopes))
	copy(result, registeredScopes)
	return result
}

// buildAllScopes 构建登录时使用的完整 scope 列表（publicData + 所有已注册 scope）
func buildLoginScopes(extraScopes []string) []string {
	scopeSet := map[string]struct{}{
		"publicData": {},
	}
	scopeMu.RLock()
	for _, rs := range registeredScopes {
		if !rs.Required {
			continue
		}
		if s := strings.TrimSpace(rs.Scope); s != "" {
			scopeSet[s] = struct{}{}
		}
	}
	scopeMu.RUnlock()

	for _, s := range extraScopes {
		if s != "" {
			scopeSet[s] = struct{}{}
		}
	}

	scopes := make([]string, 0, len(scopeSet))
	for s := range scopeSet {
		scopes = append(scopes, s)
	}
	return scopes
}

func ValidateExtraScopes(extraScopes []string, userRoles []string) error {
	return validateExtraScopes(extraScopes, userRoles)
}

func validateExtraScopes(extraScopes []string, userRoles []string) error {
	for _, scope := range extraScopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" {
			continue
		}
		if isAdminOnlyScope(trimmed) && !model.ContainsAnyRole(userRoles, model.RoleSuperAdmin, model.RoleAdmin) {
			return fmt.Errorf("scope %s 仅管理员可申请", trimmed)
		}
	}
	return nil
}

// isAdminOnlyScope 判断 scope 是否仅允许管理员持有（当前仅军团击杀邮件 scope）
func isAdminOnlyScope(scope string) bool {
	return strings.TrimSpace(scope) == corpKillmailScope
}

// allowedScopeSet 返回允许持久化到 eve_characters.scopes 的 scope 集合：
// 全部已注册 scope（含 Optional）+ publicData。已下线模块的残留 scope 会在合并时被剔除。
func allowedScopeSet() map[string]struct{} {
	set := map[string]struct{}{"publicData": {}}
	scopeMu.RLock()
	defer scopeMu.RUnlock()
	for _, rs := range registeredScopes {
		if s := strings.TrimSpace(rs.Scope); s != "" {
			set[s] = struct{}{}
		}
	}
	return set
}

// candidateScopeSet 计算 (existing ∪ granted) ∩ allowed 的候选 scope 集合
func candidateScopeSet(existing string, granted []string, allowed map[string]struct{}) map[string]struct{} {
	set := make(map[string]struct{}, len(allowed))
	for _, s := range strings.Fields(existing) {
		if _, ok := allowed[s]; ok {
			set[s] = struct{}{}
		}
	}
	for _, s := range granted {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := allowed[s]; ok {
			set[s] = struct{}{}
		}
	}
	return set
}

// containsAdminOnlyScope 判断 scope 集合中是否含管理员专属可选 scope
func containsAdminOnlyScope(set map[string]struct{}) bool {
	for s := range set {
		if isAdminOnlyScope(s) {
			return true
		}
	}
	return false
}

// scopeSnapshot 将 scope 集合按字典序排序后以空格连接为快照字符串；
// keepAdminOnly=false 时剥离管理员专属可选 scope
func scopeSnapshot(set map[string]struct{}, keepAdminOnly bool) string {
	scopes := make([]string, 0, len(set))
	for s := range set {
		if !keepAdminOnly && isAdminOnlyScope(s) {
			continue
		}
		scopes = append(scopes, s)
	}
	sort.Strings(scopes)
	return strings.Join(scopes, " ")
}

// ─────────────────────────────────────────────
//  SSO Service
// ─────────────────────────────────────────────

const (
	stateCachePrefix            = "eve:sso:state:"
	stateCacheTTL               = 10 * time.Minute
	affiliationResponseMaxBytes = 1 << 20
	corpKillmailScope           = "esi-killmails.read_corporation_killmails.v1"
)

// OnNewCharacterFunc 新人物首次出现时触发的钩子（由 jobs 层注入以避免循环依赖）
// 在后台 goroutine 中运行，用于全量 ESI 刷新。全量刷新完成后应执行一次权限检查。
var OnNewCharacterFunc func(characterID int64, userID uint)

// OnNewCharacterSyncFunc 新人物同步钩子：在返回 JWT 之前调用。
// 负责刷新最小安全相关数据（如 affiliation / corp roles）并完成权限重算。
// 必须快速返回（< 1s），调用方不会使用 goroutine。
var OnNewCharacterSyncFunc func(characterID int64, userID uint)

// OnExistingCharacterSyncFunc 已有人物完成绑定/重新登录时触发的同步钩子。
// 必须在签发 JWT 前完成，用于刷新 affiliation / corp roles 并重算权限。
var OnExistingCharacterSyncFunc func(characterID int64, userID uint)

// stateData OAuth state 中存储的数据
type stateData struct {
	ExtraScopes         []string `json:"extra_scopes,omitempty"`
	RedirectURL         string   `json:"redirect_url,omitempty"`
	BindToUserID        uint     `json:"bind_to_user_id,omitempty"` // >0 时表示「绑定人物」流程，而非登录
	ScopeRefreshAttempted bool   `json:"scope_refresh_attempted,omitempty"` // 已发起过一次可选 scope 补授，防循环
}

// EveSSOService EVE SSO 业务逻辑层
type EveSSOService struct {
	charRepo  *repository.EveCharacterRepository
	userRepo  *repository.UserRepository
	roleSvc   *RoleService
	eveClient *eve.Client
	esiClient *esi.Client
}

func configuredEveSSOConfig() config.EveSSOConfig {
	resolvedConfig := &config.Config{}
	if global.Config != nil {
		*resolvedConfig = *global.Config
	}
	config.ApplyDefaults(resolvedConfig)
	return resolvedConfig.EveSSO
}

func NewEveSSOService() *EveSSOService {
	cfg := configuredEveSSOConfig()
	return &EveSSOService{
		charRepo: repository.NewEveCharacterRepository(),
		userRepo: repository.NewUserRepository(),
		roleSvc:  NewRoleService(),
		eveClient: eve.NewClientWithEndpoints(
			cfg.ClientID,
			cfg.ClientSecret,
			cfg.CallbackURL,
			cfg.SSOAuthorizeURL,
			cfg.SSOTokenURL,
		),
		esiClient: esi.NewClientWithConfig(cfg.ESIBaseURL, cfg.ESIAPIPrefix),
	}
}

func (s *EveSSOService) loadUserAndGenerateToken(userID uint) (string, *model.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return "", nil, err
	}

	token, err := jwt.GenerateToken(user.ID, user.PrimaryCharacterID, user.Role, global.Config.JWT.ExpireDay)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

type characterAffiliationSnapshot struct {
	AllianceID    *int64 `json:"alliance_id,omitempty"`
	CorporationID int64  `json:"corporation_id"`
	FactionID     *int64 `json:"faction_id,omitempty"`
}

func resolveInitialSSORole(corporationID int64, allowCorporations []int64) string {
	for _, allowedID := range allowCorporations {
		if allowedID == corporationID {
			return model.RoleUser
		}
	}
	return model.RoleGuest
}

func buildDefaultSSOUser(primaryCharacterID int64, clientIP string, now time.Time, role string) *model.User {
	if role == "" {
		role = model.RoleGuest
	}

	user := &model.User{
		Nickname:           "",
		Status:             1,
		Role:               role,
		PrimaryCharacterID: primaryCharacterID,
		LastLoginAt:        &now,
		LastLoginIP:        clientIP,
	}
	return user
}

func (s *EveSSOService) fetchCharacterAffiliation(ctx context.Context, characterID int64) (*characterAffiliationSnapshot, error) {
	var results []characterAffiliationSnapshot

	if err := s.esiClient.PostJSONWithLimit(ctx, "/characters/affiliation/", "", []int64{characterID}, &results, affiliationResponseMaxBytes); err != nil {
		return nil, fmt.Errorf("fetch affiliation: %w", err)
	}
	if len(results) == 0 {
		return nil, errors.New("人物归属信息为空")
	}
	return &results[0], nil
}

func (s *EveSSOService) resolveInitialSSOState(ctx context.Context, characterID int64) (string, *characterAffiliationSnapshot) {
	affiliation, err := s.fetchCharacterAffiliation(ctx, characterID)
	if err != nil {
		global.Logger.Warn("首次登录查询人物归属失败，回退为 guest",
			zap.Int64("character_id", characterID),
			zap.Error(err))
		return model.RoleGuest, nil
	}
	return resolveInitialSSORole(affiliation.CorporationID, utils.GetAllowCorporations()), affiliation
}

func (s *EveSSOService) createDefaultSSOUser(ctx context.Context, primaryCharacterID int64, clientIP string, now time.Time, initialRole string) (*model.User, error) {
	finalRole := initialRole
	for _, adminCharID := range global.Config.App.SuperAdmins {
		if adminCharID == primaryCharacterID {
			finalRole = model.RoleSuperAdmin
			global.Logger.Info("从配置文件授予超级管理员职权",
				zap.Int64("character_id", primaryCharacterID))
			break
		}
	}

	user := buildDefaultSSOUser(primaryCharacterID, clientIP, now, finalRole)
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	s.roleSvc.EnsureUserHasRole(ctx, user.ID, user.Role)
	return user, nil
}

// SyncConfigSuperAdmins 根据配置文件同步用户的 super_admin 职权
// 如果用户的任意人物 ID 在配置列表中则授予，否则移除
func SyncConfigSuperAdmins(ctx context.Context, userID uint) {
	charRepo := repository.NewEveCharacterRepository()
	roleRepo := repository.NewRoleRepository()
	roleSvc := NewRoleService()

	chars, err := charRepo.ListByUserID(userID)
	if err != nil {
		global.Logger.Error("SyncConfigSuperAdmins 查询人物失败", zap.Uint("userID", userID), zap.Error(err))
		return
	}

	userCharIDs := make(map[int64]struct{}, len(chars))
	for _, c := range chars {
		userCharIDs[c.CharacterID] = struct{}{}
	}

	shouldSuperAdmin := false
	for _, adminCharID := range global.Config.App.SuperAdmins {
		if _, ok := userCharIDs[adminCharID]; ok {
			shouldSuperAdmin = true
			break
		}
	}

	currentCodes, err := roleRepo.GetUserRoleCodes(userID)
	if err != nil {
		global.Logger.Error("SyncConfigSuperAdmins 查询人物失败", zap.Uint("userID", userID), zap.Error(err))
		return
	}

	hasSuperAdmin := model.ContainsRole(currentCodes, model.RoleSuperAdmin)

	if shouldSuperAdmin && !hasSuperAdmin {
		if err := roleRepo.AddUserRole(userID, model.RoleSuperAdmin); err != nil {
			global.Logger.Error("SyncConfigSuperAdmins 授予 super_admin 失败", zap.Uint("userID", userID), zap.Error(err))
			return
		}
		roleSvc.InvalidateUserCache(ctx, userID)
		global.Logger.Info("SyncConfigSuperAdmins 授予超级管理员", zap.Uint("userID", userID))
	} else if !shouldSuperAdmin && hasSuperAdmin {
		if err := roleRepo.RemoveUserRole(userID, model.RoleSuperAdmin); err != nil {
			global.Logger.Error("SyncConfigSuperAdmins 移除 super_admin 失败", zap.Uint("userID", userID), zap.Error(err))
			return
		}
		roleSvc.InvalidateUserCache(ctx, userID)
		global.Logger.Info("SyncConfigSuperAdmins 移除超级管理员", zap.Uint("userID", userID))
	}
}

func applyAffiliationToCharacter(char *model.EveCharacter, affiliation *characterAffiliationSnapshot) {
	if char == nil || affiliation == nil {
		return
	}
	char.CorporationID = affiliation.CorporationID
	char.AllianceID = affiliation.AllianceID
	char.FactionID = affiliation.FactionID
}

// mergedScopesForOwner 计算已有人物的合并 scope 快照：
// (库中现有 scopes ∪ 本次授予 scopes) ∩ 当前注册 scope 集合。
// 仅当候选集包含管理员专属可选 scope 时才查询归属用户当前职权（常规登录零额外查询），
// 查询失败返回错误（fail-closed，与 authorizeBindExtraScopes 一致）。
func (s *EveSSOService) mergedScopesForOwner(ctx context.Context, existing string, granted []string, ownerUserID uint) (string, error) {
	candidates := candidateScopeSet(existing, granted, allowedScopeSet())
	if !containsAdminOnlyScope(candidates) {
		return scopeSnapshot(candidates, true), nil
	}
	keepAdminOnly := false
	if ownerUserID > 0 {
		userRoles, err := s.roleSvc.GetUserRoleNames(ctx, ownerUserID)
		if err != nil {
			return "", fmt.Errorf("查询用户职权失败: %w", err)
		}
		keepAdminOnly = model.ContainsAnyRole(userRoles, model.RoleSuperAdmin, model.RoleAdmin)
	}
	return scopeSnapshot(candidates, keepAdminOnly), nil
}

// applyMergedScopes 将合并后的 scope 快照写回内存中的 char（调用方负责持久化）。
// 合并幂等：普通登录不再剥离此前显式授予的可选 scope。
func (s *EveSSOService) applyMergedScopes(ctx context.Context, char *model.EveCharacter, grantedScopes []string) error {
	merged, err := s.mergedScopesForOwner(ctx, char.Scopes, grantedScopes, char.UserID)
	if err != nil {
		return err
	}
	char.Scopes = merged
	return nil
}

// applyCallbackTokenFields 将本次 SSO 授权得到的 token 写入内存中的 char（调用方负责持久化）
func applyCallbackTokenFields(char *model.EveCharacter, accessToken, refreshToken string, tokenExpiry time.Time, characterName string) {
	char.AccessToken = accessToken
	char.RefreshToken = refreshToken
	char.TokenExpiry = tokenExpiry
	char.CharacterName = characterName
	char.TokenInvalid = false
}

// GetAuthURL 生成 EVE SSO 授权 URL，并将 state 存入 Redis
//
//	extraScopes: 额外需要的 scope，传 nil 则使用所有已注册 scope
//	redirectURL: 登录成功后前端跳转地址
func (s *EveSSOService) GetAuthURL(ctx context.Context, extraScopes []string, redirectURL string) (string, error) {
	// 生成随机 state
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := hex.EncodeToString(b)

	// 存入 Redis
	data := stateData{ExtraScopes: extraScopes, RedirectURL: redirectURL}
	if err := cache.Set(ctx, stateCachePrefix+state, data, stateCacheTTL); err != nil {
		global.Logger.Warn("存储 SSO state 失败", zap.Error(err))
		// Redis 不可用时仍允许继续（降级）
	}

	scopes := buildLoginScopes(extraScopes)
	return s.eveClient.BuildAuthURL(state, scopes), nil
}

// GetBindAuthURL 生成「绑定新人物」的 EVE SSO 授权 URL
// 与 GetAuthURL 不同的是，state 中会记录当前登录用户 ID，回调时将人物绑到该用户下
func (s *EveSSOService) GetBindAuthURL(ctx context.Context, userID uint, extraScopes []string, redirectURL string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := hex.EncodeToString(b)

	data := stateData{ExtraScopes: extraScopes, RedirectURL: redirectURL, BindToUserID: userID}
	if err := cache.Set(ctx, stateCachePrefix+state, data, stateCacheTTL); err != nil {
		global.Logger.Warn("存储 SSO state 失败", zap.Error(err))
	}

	scopes := buildLoginScopes(extraScopes)
	return s.eveClient.BuildAuthURL(state, scopes), nil
}

// missingOptionalScopesForReauth 计算需要透明补授的可选 scope：
// 历史授权中已存在、本次授予链缺失、且当前已注册的非管理员专属 Optional scope。
// 管理员专属 scope 不参与补授——公共登录链无法通过 nil-roles 校验，
// 维持管理员经 ESI 检查页 bind 流程显式授予的现状。
func missingOptionalScopesForReauth(existing string, granted []string) []string {
	grantedSet := make(map[string]struct{}, len(granted))
	for _, s := range granted {
		if s = strings.TrimSpace(s); s != "" {
			grantedSet[s] = struct{}{}
		}
	}

	optionalSet := make(map[string]struct{})
	scopeMu.RLock()
	for _, rs := range registeredScopes {
		if !rs.Required && rs.Scope != "" && !isAdminOnlyScope(rs.Scope) {
			optionalSet[rs.Scope] = struct{}{}
		}
	}
	scopeMu.RUnlock()

	seen := make(map[string]struct{}, len(optionalSet))
	missing := make([]string, 0)
	for _, s := range strings.Fields(existing) {
		if _, ok := grantedSet[s]; ok {
			continue
		}
		if _, optional := optionalSet[s]; !optional {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		missing = append(missing, s)
	}
	sort.Strings(missing)
	return missing
}

// buildScopeReauthURL 生成补授缺失可选 scope 的授权 URL（请求范围为必需 scope + 缺失项）。
// state 写入失败时返回错误——ScopeRefreshAttempted 标记无法持久化时不得发起补授，
// 避免缓存降级场景下的无限重定向。
func (s *EveSSOService) buildScopeReauthURL(ctx context.Context, sd stateData, missingScopes []string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := hex.EncodeToString(b)

	next := stateData{
		ExtraScopes:           missingScopes,
		RedirectURL:           sd.RedirectURL,
		BindToUserID:          sd.BindToUserID,
		ScopeRefreshAttempted: true,
	}
	if err := cache.Set(ctx, stateCachePrefix+state, next, stateCacheTTL); err != nil {
		return "", fmt.Errorf("存储补授 state 失败: %w", err)
	}

	return s.eveClient.BuildAuthURL(state, buildLoginScopes(missingScopes)), nil
}

// scopeReauthResult 判定是否需要透明补授并生成授权 URL。
// missing 为空、本轮已补授过、或补授 state 无法持久化时返回 ok=false，
// 调用方按普通登录继续（链收缩由任务层 403 降级与 ESI 检查页兜底）。
func (s *EveSSOService) scopeReauthResult(ctx context.Context, sd stateData, missing []string) (string, bool) {
	if len(missing) == 0 || sd.ScopeRefreshAttempted {
		return "", false
	}
	reauthURL, err := s.buildScopeReauthURL(ctx, sd, missing)
	if err != nil {
		global.Logger.Warn("生成可选 scope 补授 URL 失败，跳过补授", zap.Error(err))
		return "", false
	}
	return reauthURL, true
}

// CallbackResult EVE SSO 回调处理结果
type CallbackResult struct {
	Token       string              `json:"token"` // 我们系统颁发的 JWT
	User        *model.User         `json:"user"`
	Character   *model.EveCharacter `json:"character"`
	RedirectURL string              `json:"redirect_url"` // 前端跳转地址（可能为空）
	// ReauthURL 非空时表示需要透明补授缺失的可选 scope：handler 应 302 到该 EVE 授权地址
	// 而不是返回 JWT。EVE 对历史已授过的 scope 不再弹确认页，用户近乎无感。
	ReauthURL string `json:"-"`
}

// HandleCallback 处理 EVE SSO 回调，完成 Token 交换、用户创建/更新，颁发本系统 JWT
func (s *EveSSOService) HandleCallback(ctx context.Context, code, state, clientIP string) (*CallbackResult, error) {
	if code == "" {
		return nil, errors.New("授权码不能为空")
	}

	// 读取并删除 state（防重放）
	var sd stateData
	if err := cache.Get(ctx, stateCachePrefix+state, &sd); err != nil {
		// state 不存在或已过期——记录但不强制失败（降级兼容）
		global.Logger.Warn("EVE SSO state 未找到或已过期", zap.String("state", state))
	} else {
		_ = cache.Del(ctx, stateCachePrefix+state)
	}
	if sd.BindToUserID > 0 {
		if err := s.authorizeBindExtraScopes(ctx, sd.BindToUserID, sd.ExtraScopes); err != nil {
			return nil, err
		}
	}

	// 1. 用授权码换取 Token
	tokenResp, err := s.eveClient.ExchangeCode(ctx, code)
	if err != nil {
		return nil, err
	}
	// 2. 解析 JWT access_token 获取人物信息
	claims, err := eve.ParseAccessToken(tokenResp.AccessToken)
	if err != nil {
		return nil, err
	}
	grantedScopes := claims.GetScopes()
	if err := s.authorizeCallbackScopes(ctx, sd.BindToUserID, grantedScopes); err != nil {
		return nil, err
	}
	characterID, err := claims.GetCharacterID()
	if err != nil {
		return nil, err
	}

	tokenExpiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	scopesStr := strings.Join(grantedScopes, " ")
	initialRole, affiliation := s.resolveInitialSSOState(ctx, characterID)

	// 3. 查找或创建 EveCharacter
	char, err := s.charRepo.GetByCharacterID(characterID)
	now := time.Now()

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		// 该人物第一次出现

		// ── 绑定流程：将新人物绑定到已登录的用户 ──
		if sd.BindToUserID > 0 {
			user, err := s.userRepo.GetByID(sd.BindToUserID)
			if err != nil {
				return nil, errors.New("绑定目标用户不存在")
			}

			char = &model.EveCharacter{
				CharacterID:   characterID,
				CharacterName: claims.Name,
				UserID:        user.ID,
				AccessToken:   tokenResp.AccessToken,
				RefreshToken:  tokenResp.RefreshToken,
				TokenExpiry:   tokenExpiry,
				Scopes:        scopesStr,
			}
			if err := s.charRepo.Create(char); err != nil {
				return nil, err
			}

			// 如果用户还没有主人物，自动设为主人物
			if user.PrimaryCharacterID == 0 {
				user.PrimaryCharacterID = characterID
				if err := s.userRepo.Update(user); err != nil {
					return nil, err
				}
			}

			// 同步执行 affiliation / corp roles 拉取与权限重算，确保 JWT 生成前职权已正确设置
			if OnNewCharacterSyncFunc != nil {
				OnNewCharacterSyncFunc(characterID, user.ID)
			}

			// 触发新人物全量 ESI 刷新（后台异步）
			if OnNewCharacterFunc != nil {
				OnNewCharacterFunc(characterID, user.ID)
			}

			jwtToken, user, err := s.loadUserAndGenerateToken(user.ID)
			if err != nil {
				return nil, err
			}
			return &CallbackResult{Token: jwtToken, User: user, Character: char, RedirectURL: sd.RedirectURL}, nil
		}

		// ── 登录流程：首次登录，创建新用户 + 新人物 ──
		user, err := s.createDefaultSSOUser(ctx, characterID, clientIP, now, initialRole)
		if err != nil {
			return nil, err
		}

		char = &model.EveCharacter{
			CharacterID:   characterID,
			CharacterName: claims.Name,
			UserID:        user.ID,
			AccessToken:   tokenResp.AccessToken,
			RefreshToken:  tokenResp.RefreshToken,
			TokenExpiry:   tokenExpiry,
			Scopes:        scopesStr,
		}
		applyAffiliationToCharacter(char, affiliation)
		if err := s.charRepo.Create(char); err != nil {
			return nil, err
		}

		// 同步执行 affiliation / corp roles 拉取与权限重算，确保 JWT 生成前职权已正确设置
		if OnNewCharacterSyncFunc != nil {
			OnNewCharacterSyncFunc(characterID, user.ID)
		}

		// 触发新人物全量 ESI 刷新（后台异步）
		if OnNewCharacterFunc != nil {
			OnNewCharacterFunc(characterID, user.ID)
		}

		jwtToken, user, err := s.loadUserAndGenerateToken(user.ID)
		if err != nil {
			return nil, err
		}
		return &CallbackResult{Token: jwtToken, User: user, Character: char, RedirectURL: sd.RedirectURL}, nil
	}

	// 已有人物

	// ── 绑定流程：该人物已存在 ──
	if sd.BindToUserID > 0 {
		if char.UserID != sd.BindToUserID {
			// 保存原用户ID
			oldUserID := char.UserID

			// 检查原用户是否存在（是否被软删除）；仍存在则拒绝抢绑
			if _, err := s.userRepo.GetByID(oldUserID); err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("该人物已绑定到其他账号，无法再次绑定")
			}

			// 原用户已被软删除（孤儿人物），直接重新绑定到新用户
			char, err = s.updateExistingCharacterLocked(characterID, func(c *model.EveCharacter) error {
				// 锁内复查归属，防止并发期间人物被绑到其他账号
				if c.UserID != oldUserID {
					return errors.New("该人物已绑定到其他账号，无法再次绑定")
				}
				c.UserID = sd.BindToUserID
				applyCallbackTokenFields(c, tokenResp.AccessToken, tokenResp.RefreshToken, tokenExpiry, claims.Name)
				return s.applyMergedScopes(ctx, c, grantedScopes)
			})
			if err != nil {
				return nil, err
			}

			// 获取目标用户
			user, err := s.userRepo.GetByID(sd.BindToUserID)
			if err != nil {
				return nil, err
			}

			// 如果用户还没有主人物，自动设为主人物
			if user.PrimaryCharacterID == 0 {
				user.PrimaryCharacterID = characterID
				if err := s.userRepo.Update(user); err != nil {
					return nil, err
				}
			}

			// 同步刷新 affiliation / corp roles 并重算权限，确保 JWT 基于最新安全状态签发
			if OnExistingCharacterSyncFunc != nil {
				OnExistingCharacterSyncFunc(characterID, user.ID)
			}

			global.Logger.Info("孤儿人物重新绑定到新用户（绑定流程）",
				zap.Int64("characterID", characterID),
				zap.Uint("oldUserID", oldUserID),
				zap.Uint("newUserID", sd.BindToUserID))

			jwtToken, user, err := s.loadUserAndGenerateToken(user.ID)
			if err != nil {
				return nil, err
			}
			return &CallbackResult{Token: jwtToken, User: user, Character: char, RedirectURL: sd.RedirectURL}, nil
		}

		// 人物已属于当前用户，更新 Token 即可
		var reauthScopes []string
		char, err = s.updateExistingCharacterLocked(characterID, func(c *model.EveCharacter) error {
			// 锁内复查归属，防止并发期间人物被绑到其他账号
			if c.UserID != sd.BindToUserID {
				return errors.New("该人物已绑定到其他账号，无法再次绑定")
			}
			// 用合并前的库中快照计算缺失项，供本次回调后透明补授
			reauthScopes = missingOptionalScopesForReauth(c.Scopes, grantedScopes)
			applyCallbackTokenFields(c, tokenResp.AccessToken, tokenResp.RefreshToken, tokenExpiry, claims.Name)
			return s.applyMergedScopes(ctx, c, grantedScopes)
		})
		if err != nil {
			return nil, err
		}
		if reauthURL, ok := s.scopeReauthResult(ctx, sd, reauthScopes); ok {
			return &CallbackResult{ReauthURL: reauthURL}, nil
		}
		user, err := s.userRepo.GetByID(sd.BindToUserID)
		if err != nil {
			return nil, err
		}
		// 同步刷新 affiliation / corp roles 并重算权限，确保 JWT 基于最新安全状态签发
		if OnExistingCharacterSyncFunc != nil {
			OnExistingCharacterSyncFunc(characterID, user.ID)
		}
		jwtToken, user, err := s.loadUserAndGenerateToken(user.ID)
		if err != nil {
			return nil, err
		}
		return &CallbackResult{Token: jwtToken, User: user, Character: char, RedirectURL: sd.RedirectURL}, nil
	}

	// ── 登录流程：已有人物重新登录 ──
	var reauthScopes []string
	char, err = s.updateExistingCharacterLocked(characterID, func(c *model.EveCharacter) error {
		// 用合并前的库中快照计算缺失项，供本次回调后透明补授
		reauthScopes = missingOptionalScopesForReauth(c.Scopes, grantedScopes)
		applyCallbackTokenFields(c, tokenResp.AccessToken, tokenResp.RefreshToken, tokenExpiry, claims.Name)
		applyAffiliationToCharacter(c, affiliation)
		return s.applyMergedScopes(ctx, c, grantedScopes)
	})
	if err != nil {
		return nil, err
	}
	if reauthURL, ok := s.scopeReauthResult(ctx, sd, reauthScopes); ok {
		return &CallbackResult{ReauthURL: reauthURL}, nil
	}

	// 更新用户最后登录信息
	user, err := s.userRepo.GetByID(char.UserID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		oldUserID := char.UserID
		orphanInitialRole := initialRole
		if affiliation == nil {
			orphanInitialRole = resolveInitialSSORole(char.CorporationID, utils.GetAllowCorporations())
		}
		user, err = s.createDefaultSSOUser(ctx, characterID, clientIP, now, orphanInitialRole)
		if err != nil {
			return nil, err
		}

		char, err = s.updateExistingCharacterLocked(characterID, func(c *model.EveCharacter) error {
			c.UserID = user.ID
			// 按新归属用户的职权重新合并，管理员专属可选 scope 会被剥离
			return s.applyMergedScopes(ctx, c, grantedScopes)
		})
		if err != nil {
			return nil, err
		}

		global.Logger.Info("孤儿人物重新创建用户（登录流程）",
			zap.Int64("characterID", characterID),
			zap.Uint("oldUserID", oldUserID),
			zap.Uint("newUserID", user.ID))
	}
	user.LastLoginAt = &now
	user.LastLoginIP = clientIP
	// 如果用户还没有主人物，自动设为当前登录人物
	if user.PrimaryCharacterID == 0 {
		user.PrimaryCharacterID = characterID
	}
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	// 同步刷新 affiliation / corp roles 并重算权限，确保 JWT 基于最新安全状态签发
	if OnExistingCharacterSyncFunc != nil {
		OnExistingCharacterSyncFunc(characterID, user.ID)
	}

	// 根据配置文件同步 super_admin 职权
	SyncConfigSuperAdmins(context.Background(), user.ID)

	jwtToken, user, err := s.loadUserAndGenerateToken(user.ID)
	if err != nil {
		return nil, err
	}
	return &CallbackResult{Token: jwtToken, User: user, Character: char, RedirectURL: sd.RedirectURL}, nil
}

func (s *EveSSOService) authorizeBindExtraScopes(ctx context.Context, userID uint, extraScopes []string) error {
	userRoles, err := s.roleSvc.GetUserRoleNames(ctx, userID)
	if err != nil {
		return err
	}
	return ValidateExtraScopes(extraScopes, userRoles)
}

func (s *EveSSOService) authorizeCallbackScopes(ctx context.Context, bindToUserID uint, grantedScopes []string) error {
	if bindToUserID > 0 {
		return s.authorizeBindExtraScopes(ctx, bindToUserID, grantedScopes)
	}
	return ValidateExtraScopes(grantedScopes, nil)
}

// ─────────────────────────────────────────────
//  Token 刷新并发控制
// ─────────────────────────────────────────────

// tokenRefreshLocks 每个 characterID 一把锁，防止并发刷新同一人物的 token。
// 使用 sync.Map 避免额外的全局锁；条目会随进程中见过的人物 ID 集合增长，并在进程重启后重置。
var tokenRefreshLocks sync.Map

// getCharacterLock 返回指定人物的互斥锁（懒初始化）
func getCharacterLock(characterID int64) *sync.Mutex {
	mu, _ := tokenRefreshLocks.LoadOrStore(characterID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// updateExistingCharacterLocked 在 per-character 锁内完成已有人物的读-改-写：
// 重读最新行 → mutate → 整行写回。用于 SSO 回调路径，防止与 refreshCharacterToken
// 的并发刷新互相覆盖对方刚写入的 refresh_token（EVE refresh token 为轮换式）。
// mutate 内只允许内存修改与本地 DB/Redis 读取，禁止网络调用；
// 返回更新后的行。调用方必须在拿回结果并释放锁之后才能执行 OnExistingCharacterSyncFunc
// 等会经 GetValidToken 重入同一把锁的钩子。
func (s *EveSSOService) updateExistingCharacterLocked(
	characterID int64,
	mutate func(char *model.EveCharacter) error,
) (*model.EveCharacter, error) {
	mu := getCharacterLock(characterID)
	mu.Lock()
	defer mu.Unlock()

	char, err := s.charRepo.GetByCharacterID(characterID)
	if err != nil {
		return nil, err
	}
	if err := mutate(char); err != nil {
		return nil, err
	}
	if err := s.charRepo.Update(char); err != nil {
		return nil, err
	}
	return char, nil
}

// isTokenErrorPermanent 判断 ESI token 刷新错误是否为不可恢复错误。
// 仅当 ESI 明确返回 invalid_grant / invalid_token 时才视为永久失效。
func isTokenErrorPermanent(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "invalid_grant") ||
		strings.Contains(msg, "invalid_token")
}

// GetValidToken 获取指定人物的有效 access_token（如即将过期则自动刷新）
// 供其他模块调用，用于发起 ESI 请求
func (s *EveSSOService) GetValidToken(ctx context.Context, characterID int64) (string, error) {
	char, err := s.charRepo.GetByCharacterID(characterID)
	if err != nil {
		return "", err
	}

	// Token 已标记为失效
	if char.TokenInvalid {
		return "", errors.New("该人物的 token 已失效，请重新授权")
	}

	// Token 有效期剩余 < 3 分钟则刷新
	if time.Until(char.TokenExpiry) < 3*time.Minute {
		if err := s.refreshCharacterToken(ctx, characterID); err != nil {
			return "", err
		}
		// 刷新后重新读取，拿到最新 access_token
		char, err = s.charRepo.GetByCharacterID(characterID)
		if err != nil {
			return "", err
		}
	}

	return char.AccessToken, nil
}

// refreshCharacterToken 刷新人物 Token 并持久化。
// 使用 per-character 互斥锁防止并发刷新：
//   - 第一个 goroutine 执行实际刷新
//   - 后续 goroutine 等锁释放后 re-read DB，发现 token 已更新则直接返回
func (s *EveSSOService) refreshCharacterToken(ctx context.Context, characterID int64) error {
	mu := getCharacterLock(characterID)
	mu.Lock()
	defer mu.Unlock()

	// 拿到锁后重新读取 — 可能别的 goroutine 已经刷新过了
	char, err := s.charRepo.GetByCharacterID(characterID)
	if err != nil {
		return err
	}
	if char.TokenInvalid {
		return errors.New("该人物的 token 已失效，请重新授权")
	}
	// 如果 token 还有 ≥ 3 分钟有效期，说明并发 goroutine 已完成刷新
	if time.Until(char.TokenExpiry) >= 3*time.Minute {
		return nil
	}

	tokenResp, err := s.eveClient.RefreshAccessToken(ctx, char.RefreshToken)
	if err != nil {
		if isTokenErrorPermanent(err) {
			// 仅对永久性错误标记 token 失效
			char.TokenInvalid = true
			if dbErr := s.charRepo.Update(char); dbErr != nil {
				global.Logger.Error("标记 token 失效写 DB 失败",
					zap.Int64("character_id", characterID),
					zap.Error(dbErr),
				)
			}
		} else {
			global.Logger.Warn("ESI token 刷新暂时失败（不标记失效）",
				zap.Int64("character_id", characterID),
				zap.Error(err),
			)
		}
		return err
	}

	// 不解析新 access_token、也不用其 claims 覆盖 scopes：
	// 刷新产生的 token 链只包含授予时请求的 scope 集合，覆盖会剥离此前
	// 显式授予的可选 scope；且解析失败时若直接返回会丢弃已轮换的
	// refresh_token（旧 token 已作废，下次刷新必然 invalid_grant）。
	char.AccessToken = tokenResp.AccessToken
	char.RefreshToken = tokenResp.RefreshToken
	char.TokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	char.TokenInvalid = false

	return s.charRepo.Update(char)
}

// GetCharactersByUserID 获取用户绑定的所有 EVE 人物（不含 Token）
func (s *EveSSOService) GetCharactersByUserID(userID uint) ([]model.EveCharacter, error) {
	return s.charRepo.ListByUserID(userID)
}

// SetPrimaryCharacter 设置用户的主人物
func (s *EveSSOService) SetPrimaryCharacter(userID uint, characterID int64) error {
	// 验证该人物确实属于当前用户
	char, err := s.charRepo.GetByCharacterID(characterID)
	if err != nil {
		return errors.New("人物不存在")
	}
	if char.UserID != userID {
		return errors.New("该人物不属于当前用户")
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	user.PrimaryCharacterID = characterID
	return s.userRepo.Update(user)
}

// UnbindCharacter 解除绑定某个 EVE 人物
func (s *EveSSOService) UnbindCharacter(userID uint, characterID int64) error {
	char, err := s.charRepo.GetByCharacterID(characterID)
	if err != nil {
		return errors.New("人物不存在")
	}
	if char.UserID != userID {
		return errors.New("该人物不属于当前用户")
	}

	// 确保至少保留一个人物
	chars, err := s.charRepo.ListByUserID(userID)
	if err != nil {
		return err
	}
	if len(chars) <= 1 {
		return errors.New("至少需要保留一个人物，无法解绑")
	}

	// 如果要解绑的是主人物，自动切换到另一个人物
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}
	if user.PrimaryCharacterID == characterID {
		for _, c := range chars {
			if c.CharacterID != characterID {
				user.PrimaryCharacterID = c.CharacterID
				break
			}
		}
		if err := s.userRepo.Update(user); err != nil {
			return err
		}
	}

	return s.charRepo.Delete(char.ID)
}

// GetRedirectURLFromState 仅读取 state 对应的前端 redirect URL（不删除 state）
// 用于错误场景下仍能跳回前端
func (s *EveSSOService) GetRedirectURLFromState(ctx context.Context, state string) string {
	if state == "" {
		return ""
	}
	var sd stateData
	if err := cache.Get(ctx, stateCachePrefix+state, &sd); err != nil {
		return ""
	}
	return sd.RedirectURL
}
