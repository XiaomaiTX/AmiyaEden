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
	ID                   uint       `json:"id"`
	QQ                   string     `json:"qq"`
	EnteredAt            time.Time  `json:"entered_at"`
	Source               string     `json:"source"`
	Status               string     `json:"status"`
	MatchedUserID        uint       `json:"matched_user_id"`
	MatchedCharacterName string     `json:"matched_character_name,omitempty"`
	RewardedAt           *time.Time `json:"rewarded_at"`
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
	UserID            uint   `json:"user_id"`
	UserCharacterName string `json:"user_character_name"`
}

// ─── Service ─────────────────────────────────────────────────────────────────

type RecruitmentLinkService struct {
	repo        *repository.NewbroRecruitmentRepository
	userRepo    *repository.UserRepository
	charRepo    *repository.EveCharacterRepository
	settingsSvc *NewbroSettingsService
}

func NewRecruitmentLinkService() *RecruitmentLinkService {
	return &RecruitmentLinkService{
		repo:        repository.NewNewbroRecruitmentRepository(),
		userRepo:    repository.NewUserRepository(),
		charRepo:    repository.NewEveCharacterRepository(),
		settingsSvc: NewNewbroSettingsService(),
	}
}

// GenerateLink creates a new recruitment link for the user, enforcing the cooldown period.
func (s *RecruitmentLinkService) GenerateLink(userID uint, now time.Time) (*model.NewbroRecruitment, bool, error) {
	settings := s.settingsSvc.GetSettings()
	cooldown := time.Duration(settings.RecruitCooldownDays) * 24 * time.Hour

	// Assign a unique placeholder before insert to avoid a unique-constraint
	// collision on Code when multiple users generate links concurrently.
	// The user row is locked via SELECT FOR UPDATE inside the transaction, so
	// ~<userID> is unique across in-flight transactions for different users.
	// The placeholder is overwritten with base62(ID) within the same transaction.
	rec := &model.NewbroRecruitment{UserID: userID, Source: model.RecruitmentSourceLink, GeneratedAt: now, Code: fmt.Sprintf("~%d", userID)}
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
	recruiterUserIDs := make([]uint, 0, len(recs))
	for _, rec := range recs {
		recruiterUserIDs = append(recruiterUserIDs, rec.UserID)
	}
	recruiterNames, err := s.loadUserDisplayNames(recruiterUserIDs)
	if err != nil {
		return nil, 0, err
	}
	result := make([]AdminRecruitLinkRow, len(rows))
	for i, row := range rows {
		result[i] = AdminRecruitLinkRow{
			RecruitLinkRow:    row,
			UserID:            recs[i].UserID,
			UserCharacterName: recruiterNames[recs[i].UserID],
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
	matchedUserIDs := make([]uint, 0, len(entries))
	for _, e := range entries {
		if e.Status != model.RecruitEntryStatusValid || e.MatchedUserID == 0 {
			continue
		}
		matchedUserIDs = append(matchedUserIDs, e.MatchedUserID)
	}
	matchedDisplayNames, err := s.loadUserDisplayNames(matchedUserIDs)
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
			ID:                   e.ID,
			QQ:                   e.QQ,
			EnteredAt:            e.EnteredAt,
			Source:               model.NormalizeRecruitEntrySource(e.Source),
			Status:               e.Status,
			MatchedUserID:        e.MatchedUserID,
			MatchedCharacterName: matchedDisplayNames[e.MatchedUserID],
			RewardedAt:           e.RewardedAt,
		})
	}
	return rows, nil
}

func (s *RecruitmentLinkService) loadUserDisplayNames(userIDs []uint) (map[uint]string, error) {
	result := make(map[uint]string)
	uniqueUserIDs := make([]uint, 0, len(userIDs))
	seenUserIDs := make(map[uint]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if userID == 0 {
			continue
		}
		if _, exists := seenUserIDs[userID]; exists {
			continue
		}
		seenUserIDs[userID] = struct{}{}
		uniqueUserIDs = append(uniqueUserIDs, userID)
	}
	if len(uniqueUserIDs) == 0 {
		return result, nil
	}

	users, err := s.userRepo.ListByIDs(uniqueUserIDs)
	if err != nil {
		return nil, err
	}
	userByID := make(map[uint]model.User, len(users))
	characterIDs := make([]int64, 0, len(users))
	for _, user := range users {
		userByID[user.ID] = user
		if user.PrimaryCharacterID != 0 {
			characterIDs = append(characterIDs, user.PrimaryCharacterID)
		}
	}

	charByID := map[int64]model.EveCharacter{}
	if len(characterIDs) > 0 {
		chars, err := s.charRepo.ListByCharacterIDs(characterIDs)
		if err != nil {
			return nil, err
		}
		charByID = make(map[int64]model.EveCharacter, len(chars))
		for _, char := range chars {
			charByID[char.CharacterID] = char
		}
	}

	for _, userID := range uniqueUserIDs {
		user, ok := userByID[userID]
		if !ok {
			result[userID] = fmt.Sprintf("%d", userID)
			continue
		}
		if user.PrimaryCharacterID != 0 {
			if char, exists := charByID[user.PrimaryCharacterID]; exists && strings.TrimSpace(char.CharacterName) != "" {
				result[userID] = char.CharacterName
				continue
			}
		}
		if strings.TrimSpace(user.Nickname) != "" {
			result[userID] = user.Nickname
			continue
		}
		result[userID] = fmt.Sprintf("%d", userID)
	}

	return result, nil
}
