package service

import (
	"amiya-eden/internal/model"
	"time"
)

const newbroRecentAffiliationLimit = 10

func normalizeRecentAffiliations(rows []model.NewbroCaptainAffiliation) []model.NewbroCaptainAffiliation {
	if len(rows) <= newbroRecentAffiliationLimit {
		return rows
	}
	return rows[:newbroRecentAffiliationLimit]
}

func shouldReuseCurrentAffiliation(current *model.NewbroCaptainAffiliation, captainUserID uint) bool {
	if current == nil || current.EndedAt != nil {
		return false
	}
	return current.CaptainUserID == captainUserID
}

func shouldBlockSelfAffiliation(playerUserID, captainUserID uint) bool {
	return playerUserID != 0 && playerUserID == captainUserID
}

func filterCaptainCandidateUsers(currentUserID uint, users []model.User) []model.User {
	if len(users) == 0 {
		return users
	}

	filtered := make([]model.User, 0, len(users))
	for _, user := range users {
		if user.ID == currentUserID {
			continue
		}
		filtered = append(filtered, user)
	}
	return filtered
}

func buildNewbroCaptainAffiliation(
	playerUserID uint,
	playerPrimaryCharacterIDAtStart int64,
	captainUserID uint,
	createdBy uint,
	startedAt time.Time,
) model.NewbroCaptainAffiliation {
	return model.NewbroCaptainAffiliation{
		PlayerUserID:                    playerUserID,
		PlayerPrimaryCharacterIDAtStart: playerPrimaryCharacterIDAtStart,
		CaptainUserID:                   captainUserID,
		CreatedBy:                       createdBy,
		StartedAt:                       startedAt,
	}
}
