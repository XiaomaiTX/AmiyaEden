package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/utils"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ─────────────────────────────────────────────
//  Character Corporation Roles 军团职权权限
//  GET /characters/{character_id}/roles/
//  默认刷新间隔: 24 Hours / 不活跃: 24 Hours
//  需要 scope: esi-characters.read_corporation_roles.v1
// ─────────────────────────────────────────────

func init() {
	Register(&CorpRolesTask{})
}

// CorpRolesTask 军团职权权限刷新任务
type CorpRolesTask struct{}

func (t *CorpRolesTask) Name() string        { return "character_corp_roles" }
func (t *CorpRolesTask) Description() string { return "人物军团职权" }
func (t *CorpRolesTask) Priority() Priority  { return PriorityHigh }

func (t *CorpRolesTask) Interval() RefreshInterval {
	return RefreshInterval{
		Active:   24 * time.Hour,
		Inactive: 24 * time.Hour,
	}
}

func (t *CorpRolesTask) RequiredScopes() []TaskScope {
	return []TaskScope{
		{Scope: "esi-characters.read_corporation_roles.v1", Description: "读取人物军团职权"},
	}
}

// corpRolesResponse ESI 返回的军团职权数据
type corpRolesResponse struct {
	Roles        []string `json:"roles"`
	RolesAtBase  []string `json:"roles_at_base"`
	RolesAtHQ    []string `json:"roles_at_hq"`
	RolesAtOther []string `json:"roles_at_other"`
}

func (t *CorpRolesTask) Execute(ctx *TaskContext) error {
	bgCtx := ctx.ContextOrBackground()
	path := fmt.Sprintf("/characters/%d/roles/", ctx.CharacterID)
	character, err := repository.NewEveCharacterRepository().GetByCharacterID(ctx.CharacterID)
	if err != nil {
		return fmt.Errorf("query corporation before roles: %w", err)
	}
	corpID := character.CorporationID

	var rolesResp corpRolesResponse
	if err := ctx.Client.Get(bgCtx, path, ctx.AccessToken, &rolesResp); err != nil {
		return fmt.Errorf("fetch corporation roles: %w", err)
	}

	// 合并四个职权列表并去重
	roleSet := make(map[string]struct{})
	for _, r := range rolesResp.Roles {
		roleSet[r] = struct{}{}
	}
	for _, r := range rolesResp.RolesAtBase {
		roleSet[r] = struct{}{}
	}
	for _, r := range rolesResp.RolesAtHQ {
		roleSet[r] = struct{}{}
	}
	for _, r := range rolesResp.RolesAtOther {
		roleSet[r] = struct{}{}
	}

	roles := make([]string, 0, len(roleSet))
	for r := range roleSet {
		roles = append(roles, r)
	}

	global.Logger.Debug("[ESI] 人物军团职权刷新完成",
		zap.Int64("character_id", ctx.CharacterID),
		zap.Int64("corporation_id", corpID),
		zap.Int("count", len(roles)),
		zap.Strings("roles", roles),
	)

	// 入库：同步人物的军团职权
	autoRoleRepo := repository.NewAutoRoleRepository()
	if corpID != model.AdmissionFMACorporationID && !isCorporationAllowed(corpID, utils.GetAllowCorporations()) {
		if err := autoRoleRepo.SyncCharacterCorpRoles(ctx.CharacterID, corpID, nil); err != nil {
			return fmt.Errorf("clear corp roles for disallowed corporation: %w", err)
		}
		global.Logger.Debug("[ESI] 人物所在军团不在 allow_corporations，已忽略军团职权信号",
			zap.Int64("character_id", ctx.CharacterID),
			zap.Int64("corporation_id", corpID))
		notifyCorporationAuthorityChanged(bgCtx, []int64{ctx.CharacterID})
		return nil
	}

	if err := autoRoleRepo.SyncCharacterCorpRoles(ctx.CharacterID, corpID, roles); err != nil {
		return fmt.Errorf("sync corp roles: %w", err)
	}
	notifyCorporationAuthorityChanged(bgCtx, []int64{ctx.CharacterID})

	return nil
}
