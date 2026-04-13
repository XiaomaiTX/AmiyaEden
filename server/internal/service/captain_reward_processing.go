package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"fmt"
	"math"
	"sort"
	"time"

	"gorm.io/gorm"
)

const captainRewardProcessingFetchLimit = 500

type captainRewardProcessingBatch struct {
	CaptainUserID      uint
	AttributionIDs     []uint
	AttributionCount   int64
	AttributedISKTotal float64
	BonusRate          float64
	CreditedValue      float64
	ProcessedAt        time.Time
}

type CaptainRewardProcessingService struct {
	attrRepo       *repository.CaptainBountyAttributionRepository
	settlementRepo *repository.CaptainRewardSettlementRepository
	walletSvc      *SysWalletService
	settingsSvc    *NewbroSettingsService
	runGuard       exclusiveRunGuard
}

func NewCaptainRewardProcessingService() *CaptainRewardProcessingService {
	return &CaptainRewardProcessingService{
		attrRepo:       repository.NewCaptainBountyAttributionRepository(),
		settlementRepo: repository.NewCaptainRewardSettlementRepository(),
		walletSvc:      NewSysWalletService(),
		settingsSvc:    NewNewbroSettingsService(),
	}
}

func calculateCaptainRewardCredit(attributedISKTotal, bonusRate float64) float64 {
	raw := (attributedISKTotal / 1_000_000) * (bonusRate / 100)
	return math.Round(raw*100) / 100
}

func buildCaptainRewardProcessingBatches(
	rows []model.CaptainBountyAttribution,
	bonusRate float64,
	processedAt time.Time,
) []captainRewardProcessingBatch {
	grouped := make(map[uint]*captainRewardProcessingBatch)
	for _, row := range rows {
		batch, ok := grouped[row.CaptainUserID]
		if !ok {
			batch = &captainRewardProcessingBatch{
				CaptainUserID: row.CaptainUserID,
				BonusRate:     bonusRate,
				ProcessedAt:   processedAt,
			}
			grouped[row.CaptainUserID] = batch
		}
		batch.AttributionIDs = append(batch.AttributionIDs, row.ID)
		batch.AttributionCount++
		batch.AttributedISKTotal += row.Amount
	}

	captainUserIDs := make([]uint, 0, len(grouped))
	for captainUserID := range grouped {
		captainUserIDs = append(captainUserIDs, captainUserID)
	}
	sort.Slice(captainUserIDs, func(i, j int) bool {
		return captainUserIDs[i] < captainUserIDs[j]
	})

	batches := make([]captainRewardProcessingBatch, 0, len(captainUserIDs))
	for _, captainUserID := range captainUserIDs {
		batch := grouped[captainUserID]
		batch.CreditedValue = calculateCaptainRewardCredit(batch.AttributedISKTotal, batch.BonusRate)
		batches = append(batches, *batch)
	}
	return batches
}

func (s *CaptainRewardProcessingService) Run(now time.Time) (*CaptainRewardProcessResult, error) {
	if err := s.runGuard.Start("队长奖励结算"); err != nil {
		return nil, err
	}
	defer s.runGuard.Finish()

	bonusRate := s.settingsSvc.GetSettings().BonusRate
	result := &CaptainRewardProcessResult{ProcessedAt: now}

	for {
		rows, err := s.attrRepo.ListUnprocessed(captainRewardProcessingFetchLimit)
		if err != nil {
			return nil, fmt.Errorf("list unprocessed attributions: %w", err)
		}
		if len(rows) == 0 {
			break
		}
		batches := buildCaptainRewardProcessingBatches(rows, bonusRate, now)

		for _, batch := range batches {
			refID := fmt.Sprintf("newbro_captain_reward:%d:%d", batch.CaptainUserID, batch.ProcessedAt.UnixNano())
			settlement := &model.CaptainRewardSettlement{
				CaptainUserID:      batch.CaptainUserID,
				AttributionCount:   batch.AttributionCount,
				AttributedISKTotal: batch.AttributedISKTotal,
				BonusRate:          batch.BonusRate,
				CreditedValue:      batch.CreditedValue,
				ProcessedAt:        batch.ProcessedAt,
				WalletRefID:        refID,
			}

			err := global.DB.Transaction(func(tx *gorm.DB) error {
				if err := s.settlementRepo.CreateTx(tx, settlement); err != nil {
					return err
				}
				reason := fmt.Sprintf("队长帮扶奖励结算（%.2f%%，%.2f ISK）", batch.BonusRate, batch.AttributedISKTotal)
				if err := s.walletSvc.ApplyWalletDeltaTx(
					tx,
					batch.CaptainUserID,
					batch.CreditedValue,
					reason,
					model.WalletRefNewbroCaptainReward,
					refID,
				); err != nil {
					return err
				}
				rowsAffected, err := s.attrRepo.MarkProcessedTx(tx, batch.AttributionIDs, batch.ProcessedAt)
				if err != nil {
					return err
				}
				if rowsAffected != int64(len(batch.AttributionIDs)) {
					return fmt.Errorf("队长 %d 的奖励处理期间归因记录发生变化", batch.CaptainUserID)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("process captain %d batch: %w", batch.CaptainUserID, err)
			}

			result.ProcessedCaptainCount++
			result.ProcessedAttributionCount += len(batch.AttributionIDs)
			result.SettlementCount++
			result.TotalCreditedValue += batch.CreditedValue
		}
	}

	result.TotalCreditedValue = math.Round(result.TotalCreditedValue*100) / 100
	return result, nil
}
