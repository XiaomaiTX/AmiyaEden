package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func init() {
	Register(&CorporationHistoryTask{})
}

// CorporationHistoryTask 人物军团任职历史刷新任务
type CorporationHistoryTask struct{}

func (t *CorporationHistoryTask) Name() string        { return "character_corporation_history" }
func (t *CorporationHistoryTask) Description() string { return "人物军团任职历史" }
func (t *CorporationHistoryTask) Priority() Priority  { return PriorityNormal }

func (t *CorporationHistoryTask) Interval() RefreshInterval {
	return RefreshInterval{
		Active:   7 * 24 * time.Hour,
		Inactive: 7 * 24 * time.Hour,
	}
}

func (t *CorporationHistoryTask) RequiredScopes() []TaskScope {
	return nil
}

type corporationHistoryResponse struct {
	RecordID      int64  `json:"record_id"`
	CorporationID int64  `json:"corporation_id"`
	IsDeleted     bool   `json:"is_deleted"`
	StartDate     string `json:"start_date"`
}

func (t *CorporationHistoryTask) Execute(ctx *TaskContext) error {
	bgCtx := ctx.ContextOrBackground()
	path := fmt.Sprintf("/characters/%d/corporationhistory/", ctx.CharacterID)

	var historyResp []corporationHistoryResponse
	if err := ctx.Client.Get(bgCtx, path, ctx.AccessToken, &historyResp); err != nil {
		return fmt.Errorf("fetch corporation history: %w", err)
	}

	historyRows := normalizeCorporationHistoryRows(ctx.CharacterID, historyResp)
	tenureDays := computeFuxiLegionTenureDays(historyRows, time.Now().UTC())
	tenureCopy := tenureDays

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("character_id = ?", ctx.CharacterID).
			Delete(&model.CharacterCorporationHistory{}).Error; err != nil {
			return fmt.Errorf("delete old corporation history: %w", err)
		}

		if len(historyRows) > 0 {
			if err := tx.Create(&historyRows).Error; err != nil {
				return fmt.Errorf("insert corporation history: %w", err)
			}
		}

		if err := tx.Model(&model.EveCharacter{}).
			Where("character_id = ?", ctx.CharacterID).
			Updates(map[string]interface{}{"fuxi_legion_tenure_days": &tenureCopy}).Error; err != nil {
			return fmt.Errorf("update tenure days: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if global.Logger != nil {
		global.Logger.Debug("[ESI] 人物军团任职历史刷新完成",
			zap.Int64("character_id", ctx.CharacterID),
			zap.Int("count", len(historyRows)),
			zap.Int("fuxi_legion_tenure_days", tenureDays),
		)
	}

	return nil
}

func normalizeCorporationHistoryRows(characterID int64, historyResp []corporationHistoryResponse) []model.CharacterCorporationHistory {
	rows := make([]model.CharacterCorporationHistory, 0, len(historyResp))
	for _, item := range historyResp {
		if strings.TrimSpace(item.StartDate) == "" {
			continue
		}
		startDate, err := time.Parse(time.RFC3339, item.StartDate)
		if err != nil {
			continue
		}
		rows = append(rows, model.CharacterCorporationHistory{
			CharacterID:   characterID,
			RecordID:      item.RecordID,
			CorporationID: item.CorporationID,
			IsDeleted:     item.IsDeleted,
			StartDate:     startDate.UTC(),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].StartDate.Equal(rows[j].StartDate) {
			return rows[i].RecordID < rows[j].RecordID
		}
		return rows[i].StartDate.Before(rows[j].StartDate)
	})

	return rows
}

func computeFuxiLegionTenureDays(historyRows []model.CharacterCorporationHistory, now time.Time) int {
	totalDays := 0
	for index, row := range historyRows {
		if row.CorporationID != model.SystemCorporationID {
			continue
		}

		endDate := now
		if index+1 < len(historyRows) {
			endDate = historyRows[index+1].StartDate
		}
		if endDate.Before(row.StartDate) {
			continue
		}

		durationDays := int(endDate.Sub(row.StartDate).Hours() / 24)
		if durationDays > 0 {
			totalDays += durationDays
		}
	}

	return totalDays
}
