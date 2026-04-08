package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────
//  Corporation Killmails 军团击杀邮件（管理员）
//  GET /corporations/{corporation_id}/killmails/recent/
//  GET /killmails/{killmail_id}/{killmail_hash}  (详情)
//  默认刷新间隔: 60 Minutes / 不活跃: 1 Day
// ─────────────────────────────────────────────

func init() {
	Register(&CorpKillmailsTask{})
}

// CorpKillmailsTask 军团击杀邮件刷新任务
type CorpKillmailsTask struct{}

func (t *CorpKillmailsTask) Name() string        { return "corporation_killmails" }
func (t *CorpKillmailsTask) Description() string { return "军团击杀/损失邮件（管理员）" }
func (t *CorpKillmailsTask) Priority() Priority  { return PriorityNormal }

func (t *CorpKillmailsTask) Interval() RefreshInterval {
	return RefreshInterval{
		Active:   60 * time.Minute,
		Inactive: 24 * time.Hour,
	}
}

func (t *CorpKillmailsTask) RequiredScopes() []TaskScope {
	return []TaskScope{
		{Scope: "esi-killmails.read_corporation_killmails.v1", Description: "读取军团击杀邮件", Optional: true},
	}
}

func (t *CorpKillmailsTask) Execute(ctx *TaskContext) error {
	bgCtx := context.Background()
	if !hasCorpKillmailDirectorRole(ctx.CharacterID) {
		global.Logger.Debug("[ESI] 军团击杀邮件：人物缺少 Director 职权，跳过刷新",
			zap.Int64("character_id", ctx.CharacterID),
		)
		return ErrTaskSkipped
	}

	// 1. 查找执行人物的 CorporationID
	var char model.EveCharacter
	if err := global.DB.Where("character_id = ?", ctx.CharacterID).First(&char).Error; err != nil {
		return fmt.Errorf("lookup character corporation: %w", err)
	}
	if char.CorporationID == 0 {
		global.Logger.Warn("[ESI] 军团击杀邮件：人物无军团信息",
			zap.Int64("character_id", ctx.CharacterID),
		)
		return nil
	}
	corpID := char.CorporationID

	// 2. 预加载本军团已知人物 ID 集合（用于关联判断）
	var charIDs []int64
	global.DB.Model(&model.EveCharacter{}).
		Where("corporation_id = ?", corpID).
		Pluck("character_id", &charIDs)
	knownChars := make(map[int64]bool, len(charIDs))
	for _, id := range charIDs {
		knownChars[id] = true
	}

	// 3. 获取最近的 killmail 列表（自动分页）
	recentPath := fmt.Sprintf("/corporations/%d/killmails/recent/", corpID)
	var refs []KillmailRef
	if _, err := ctx.Client.GetPaginated(bgCtx, recentPath, ctx.AccessToken, &refs); err != nil {
		return fmt.Errorf("fetch corporation killmails: %w", err)
	}

	global.Logger.Debug("[ESI] 军团击杀邮件引用获取完成",
		zap.Int64("character_id", ctx.CharacterID),
		zap.Int64("corporation_id", corpID),
		zap.Int("count", len(refs)),
	)

	// 4. 逐个获取 killmail 详情并入库
	for _, ref := range refs {
		// 先检查数据库中是否已存在该 killmail
		var count int64
		global.DB.Model(&model.EveKillmailList{}).Where("kill_mail_id = ?", ref.KillmailID).Count(&count)
		if count > 0 {
			// 已存在，确保受害者关联
			t.ensureVictimLink(ref.KillmailID, knownChars)
			continue
		}

		detailPath := fmt.Sprintf("/killmails/%d/%s/", ref.KillmailID, ref.KillmailHash)
		var detail KillmailDetail
		if err := ctx.Client.Get(bgCtx, detailPath, "", &detail); err != nil {
			global.Logger.Warn("[ESI] 获取 killmail 详情失败",
				zap.Int64("killmail_id", ref.KillmailID),
				zap.Error(err),
			)
			continue
		}

		// 提取 victim 信息
		var victimCharID, victimCorpID, victimAllianceID int64
		if detail.Victim.CharacterID != nil {
			victimCharID = *detail.Victim.CharacterID
		}
		if detail.Victim.CorporationID != nil {
			victimCorpID = *detail.Victim.CorporationID
		}
		if detail.Victim.AllianceID != nil {
			victimAllianceID = *detail.Victim.AllianceID
		}

		// 在事务中写入 killmail 主记录 + items
		err := global.DB.Transaction(func(tx *gorm.DB) error {
			km := model.EveKillmailList{
				KillmailID:    ref.KillmailID,
				KillmailHash:  ref.KillmailHash,
				KillmailTime:  detail.KillmailTime,
				SolarSystemID: detail.SolarSystemID,
				ShipTypeID:    int64(detail.Victim.ShipTypeID),
				CharacterID:   victimCharID,
				CorporationID: victimCorpID,
				AllianceID:    victimAllianceID,
			}
			if err := tx.Create(&km).Error; err != nil {
				return err
			}

			// 将 victim items 写入 eve_killmail_item 表
			if len(detail.Victim.Items) > 0 {
				var items []model.EveKillmailItem
				for _, it := range detail.Victim.Items {
					if it.QuantityDestroyed != nil && *it.QuantityDestroyed > 0 {
						dropType := false
						items = append(items, model.EveKillmailItem{
							KillmailID: ref.KillmailID,
							ItemID:     it.ItemTypeID,
							ItemNum:    int64(*it.QuantityDestroyed),
							DropType:   &dropType,
							Flag:       it.Flag,
						})
					}
					if it.QuantityDropped != nil && *it.QuantityDropped > 0 {
						dropType := true
						items = append(items, model.EveKillmailItem{
							KillmailID: ref.KillmailID,
							ItemID:     it.ItemTypeID,
							ItemNum:    int64(*it.QuantityDropped),
							DropType:   &dropType,
							Flag:       it.Flag,
						})
					}
				}
				if len(items) > 0 {
					if err := tx.Create(&items).Error; err != nil {
						return err
					}
				}
			}

			return nil
		})

		if err != nil {
			global.Logger.Warn("[ESI] 军团 killmail 入库失败",
				zap.Int64("killmail_id", ref.KillmailID),
				zap.Error(err),
			)
			continue
		}

		// 事务成功后，为已知人物创建关联
		if victimCharID != 0 && knownChars[victimCharID] {
			global.DB.Create(&model.EveCharacterKillmail{
				CharacterID: victimCharID,
				KillmailID:  ref.KillmailID,
				Victim:      true,
			})
		}

		global.Logger.Debug("[ESI] 军团 killmail 入库成功",
			zap.Int64("killmail_id", ref.KillmailID),
			zap.Int("items", len(detail.Victim.Items)),
			zap.Time("killmail_time", detail.KillmailTime),
		)
	}

	return nil
}

func hasCorpKillmailDirectorRole(characterID int64) bool {
	var roles []string
	if err := global.DB.Model(&model.EveCharacterCorpRole{}).
		Where("character_id = ?", characterID).
		Pluck("corp_role", &roles).Error; err != nil {
		global.Logger.Warn("[ESI] 军团击杀邮件：查询人物军团职权失败",
			zap.Int64("character_id", characterID),
			zap.Error(err),
		)
		return false
	}
	for _, role := range roles {
		if role == "Director" {
			return true
		}
	}
	return false
}

// ensureVictimLink 检查已有 killmail 的受害者是否为已知人物，若缺少关联则创建
func (t *CorpKillmailsTask) ensureVictimLink(killmailID int64, knownChars map[int64]bool) {
	var km model.EveKillmailList
	if err := global.DB.Where("kill_mail_id = ?", killmailID).First(&km).Error; err != nil {
		return
	}
	if km.CharacterID == 0 || !knownChars[km.CharacterID] {
		return
	}

	var linkCount int64
	global.DB.Model(&model.EveCharacterKillmail{}).
		Where("character_id = ? AND killmail_id = ?", km.CharacterID, killmailID).
		Count(&linkCount)
	if linkCount == 0 {
		global.DB.Create(&model.EveCharacterKillmail{
			CharacterID: km.CharacterID,
			KillmailID:  killmailID,
			Victim:      true,
		})
	}
}
