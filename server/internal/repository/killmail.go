package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"time"

	"gorm.io/gorm"
)

// KillmailRepository 击杀邮件数据访问层
type KillmailRepository struct{}

func NewKillmailRepository() *KillmailRepository {
	return &KillmailRepository{}
}

// GetCharacterKillmailLink 查询人物-KM 关联记录
func (r *KillmailRepository) GetCharacterKillmailLink(characterID, killmailID int64) (*model.EveCharacterKillmail, error) {
	var ckm model.EveCharacterKillmail
	err := global.DB.Where("character_id = ? AND killmail_id = ?", characterID, killmailID).First(&ckm).Error
	return &ckm, err
}

// GetKillmailByID 按 kill_mail_id 查询 KM 主记录
func (r *KillmailRepository) GetKillmailByID(killmailID int64) (*model.EveKillmailList, error) {
	var km model.EveKillmailList
	err := global.DB.Where("kill_mail_id = ?", killmailID).First(&km).Error
	return &km, err
}

type VictimKillmailListFilter struct {
	CharacterIDs             []int64
	Since                    *time.Time
	StartAt                  *time.Time
	EndAt                    *time.Time
	ExcludeSubmittedByUserID *uint
	Limit                    int
}

func buildVictimKillmailListQuery(db *gorm.DB, filter VictimKillmailListFilter) *gorm.DB {
	query := db.Model(&model.EveKillmailList{}).
		Where("character_id IN ?", filter.CharacterIDs).
		Order("kill_mail_time DESC")

	if filter.Since != nil {
		query = query.Where("kill_mail_time >= ?", *filter.Since)
	}
	if filter.StartAt != nil && filter.EndAt != nil {
		query = query.Where("kill_mail_time BETWEEN ? AND ?", *filter.StartAt, *filter.EndAt)
	}
	if filter.ExcludeSubmittedByUserID != nil {
		query = query.Where(`NOT EXISTS (
			SELECT 1
			FROM srp_application
			WHERE srp_application.user_id = ?
			  AND srp_application.killmail_id = eve_killmail_list.kill_mail_id
		)`, *filter.ExcludeSubmittedByUserID)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	return query
}

// ListVictimKillmailLists 查询人物作为受害者的 KM 主记录，支持时间范围、排除已提交 SRP 与条数限制
func (r *KillmailRepository) ListVictimKillmailLists(filter VictimKillmailListFilter) ([]model.EveKillmailList, error) {
	if len(filter.CharacterIDs) == 0 {
		return []model.EveKillmailList{}, nil
	}

	var list []model.EveKillmailList
	err := buildVictimKillmailListQuery(global.DB, filter).Find(&list).Error
	return list, err
}

// ListKillmailItemsByKillmailID 查询 KM 的所有物品
func (r *KillmailRepository) ListKillmailItemsByKillmailID(killmailID int64) ([]model.EveKillmailItem, error) {
	var list []model.EveKillmailItem
	err := global.DB.Where("kill_mail_id = ?", killmailID).Find(&list).Error
	return list, err
}
