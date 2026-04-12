package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func base62Encode(n uint) string {
	if n == 0 {
		return "0"
	}
	var b strings.Builder
	for n > 0 {
		b.WriteByte(base62Chars[n%62])
		n /= 62
	}
	// reverse
	s := []byte(b.String())
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return string(s)
}

// ─── Types ────────────────────────────────────────────────────────────────────

// RecruitEntryRow is an entry with display fields for the API response.
type RecruitEntryRow struct {
	ID            uint       `json:"id"`
	QQ            string     `json:"qq"`
	EnteredAt     time.Time  `json:"entered_at"`
	Source        string     `json:"source"`
	Status        string     `json:"status"`
	MatchedUserID uint       `json:"matched_user_id"`
	RewardedAt    *time.Time `json:"rewarded_at"`
}

// RecruitLinkRow is a link with its entries for the user's view.
type RecruitLinkRow struct {
	ID          uint              `json:"id"`
	Code        string            `json:"code"`
	Source      string            `json:"source"`
	GeneratedAt time.Time         `json:"generated_at"`
	Entries     []RecruitEntryRow `json:"entries"`
}

// AdminRecruitLinkRow adds the owner user_id for the admin view.
type AdminRecruitLinkRow struct {
	RecruitLinkRow
	UserID uint `json:"user_id"`
}

// ─── Service ─────────────────────────────────────────────────────────────────

type RecruitmentLinkService struct {
	repo        *repository.NewbroRecruitmentRepository
	userRepo    *repository.UserRepository
	settingsSvc *NewbroSettingsService
}

func NewRecruitmentLinkService() *RecruitmentLinkService {
	return &RecruitmentLinkService{
		repo:        repository.NewNewbroRecruitmentRepository(),
		userRepo:    repository.NewUserRepository(),
		settingsSvc: NewNewbroSettingsService(),
	}
}

// GenerateLink creates a new recruitment link for the user, enforcing the cooldown period.
func (s *RecruitmentLinkService) GenerateLink(userID uint, now time.Time) (*model.NewbroRecruitment, bool, error) {
	settings := s.settingsSvc.GetSettings()
	cooldown := time.Duration(settings.RecruitCooldownDays) * 24 * time.Hour

	rec := &model.NewbroRecruitment{UserID: userID, Source: model.RecruitmentSourceLink, GeneratedAt: now}
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := s.userRepo.GetByIDForUpdateTx(tx, userID); err != nil {
			return err
		}

		latest, err := s.repo.GetLatestGeneratedLinkByUserIDTx(tx, userID)
		if err != nil {
			return err
		}
		if latest != nil && now.Sub(latest.GeneratedAt) < cooldown {
			remaining := cooldown - now.Sub(latest.GeneratedAt)
			return fmt.Errorf("冷却中，还需等待 %d 天后才能重新生成",
				int(remaining.Hours()/24)+1)
		}

		if err := s.repo.CreateTx(tx, rec); err != nil {
			return err
		}
		rec.Code = base62Encode(rec.ID)
		return s.repo.UpdateCodeTx(tx, rec.ID, rec.Code)
	})
	if err != nil {
		return nil, false, err
	}
	return rec, true, nil
}

// GetMyLinks returns all recruitment links for a user with their entries.
func (s *RecruitmentLinkService) GetMyLinks(userID uint) ([]RecruitLinkRow, error) {
	recs, err := s.repo.ListByUserID(userID)
	if err != nil {
		return nil, err
	}
	return s.buildLinkRows(recs)
}

// ListAllLinks returns paginated recruitment links across all users (admin view).
func (s *RecruitmentLinkService) ListAllLinks(page, pageSize int) ([]AdminRecruitLinkRow, int64, error) {
	recs, total, err := s.repo.ListAllPaged(page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.buildLinkRows(recs)
	if err != nil {
		return nil, 0, err
	}
	result := make([]AdminRecruitLinkRow, len(rows))
	for i, row := range rows {
		result[i] = AdminRecruitLinkRow{
			RecruitLinkRow: row,
			UserID:         recs[i].UserID,
		}
	}
	return result, total, nil
}

func (s *RecruitmentLinkService) buildLinkRows(recs []model.NewbroRecruitment) ([]RecruitLinkRow, error) {
	if len(recs) == 0 {
		return nil, nil
	}
	ids := make([]uint, len(recs))
	indexByID := make(map[uint]int, len(recs))
	for i, r := range recs {
		ids[i] = r.ID
		indexByID[r.ID] = i
	}
	entries, err := s.repo.ListEntriesByRecruitmentIDs(ids)
	if err != nil {
		return nil, err
	}

	rows := make([]RecruitLinkRow, len(recs))
	for i, r := range recs {
		source := model.NormalizeRecruitmentSource(r.Source)
		code := r.Code
		if source == model.RecruitmentSourceDirectReferral {
			code = ""
		}
		rows[i] = RecruitLinkRow{
			ID:          r.ID,
			Code:        code,
			Source:      source,
			GeneratedAt: r.GeneratedAt,
			Entries:     []RecruitEntryRow{},
		}
	}
	for _, e := range entries {
		idx, ok := indexByID[e.RecruitmentID]
		if !ok {
			continue
		}
		rows[idx].Entries = append(rows[idx].Entries, RecruitEntryRow{
			ID:            e.ID,
			QQ:            e.QQ,
			EnteredAt:     e.EnteredAt,
			Source:        model.NormalizeRecruitEntrySource(e.Source),
			Status:        e.Status,
			MatchedUserID: e.MatchedUserID,
			RewardedAt:    e.RewardedAt,
		})
	}
	return rows, nil
}
