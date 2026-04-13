package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"fmt"
)

func uniqueNonZeroUserIDs(userIDs []uint) []uint {
	if len(userIDs) == 0 {
		return nil
	}

	seen := make(map[uint]struct{}, len(userIDs))
	result := make([]uint, 0, len(userIDs))
	for _, userID := range userIDs {
		if userID == 0 {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		result = append(result, userID)
	}
	return result
}

func loadCaptainProfiles(
	userRepo *repository.UserRepository,
	charRepo *repository.EveCharacterRepository,
	userIDs []uint,
) (map[uint]captainProfile, error) {
	result := make(map[uint]captainProfile)
	uniqueUserIDs := uniqueNonZeroUserIDs(userIDs)
	if len(uniqueUserIDs) == 0 {
		return result, nil
	}

	users, err := userRepo.ListByIDs(uniqueUserIDs)
	if err != nil {
		return nil, fmt.Errorf("load users by IDs: %w", err)
	}

	primaryCharacterIDs := make([]int64, 0, len(users))
	userByPrimaryID := make(map[int64]model.User, len(users))
	for _, user := range users {
		result[user.ID] = captainProfile{Nickname: user.Nickname}
		if user.PrimaryCharacterID == 0 {
			continue
		}
		primaryCharacterIDs = append(primaryCharacterIDs, user.PrimaryCharacterID)
		userByPrimaryID[user.PrimaryCharacterID] = user
	}

	chars, err := charRepo.ListByCharacterIDs(primaryCharacterIDs)
	if err != nil {
		return nil, fmt.Errorf("load primary characters: %w", err)
	}
	for _, char := range chars {
		user := userByPrimaryID[char.CharacterID]
		result[user.ID] = captainProfile{
			Nickname:             user.Nickname,
			PrimaryCharacterID:   char.CharacterID,
			PrimaryCharacterName: char.CharacterName,
		}
	}
	return result, nil
}
