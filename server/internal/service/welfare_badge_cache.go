package service

import (
	"amiya-eden/internal/model"
	"sync"
)

var eligibleWelfareBadgeCache = struct {
	mu     sync.RWMutex
	counts map[uint]int64
}{
	counts: make(map[uint]int64),
}

func cacheEligibleWelfareBadgeCount(userID uint, eligibleWelfares []EligibleWelfareResp) {
	count := countEligibleWelfareBadgeEntries(eligibleWelfares)

	eligibleWelfareBadgeCache.mu.Lock()
	defer eligibleWelfareBadgeCache.mu.Unlock()

	if count <= 0 {
		delete(eligibleWelfareBadgeCache.counts, userID)
		return
	}
	eligibleWelfareBadgeCache.counts[userID] = count
}

func getCachedEligibleWelfareBadgeCount(userID uint) int64 {
	eligibleWelfareBadgeCache.mu.RLock()
	defer eligibleWelfareBadgeCache.mu.RUnlock()

	return eligibleWelfareBadgeCache.counts[userID]
}

func countEligibleWelfareBadgeEntries(eligibleWelfares []EligibleWelfareResp) int64 {
	var count int64
	for _, welfare := range eligibleWelfares {
		if welfare.DistMode == model.WelfareDistModePerCharacter {
			for _, character := range welfare.EligibleCharacters {
				if character.CanApplyNow {
					count++
					break
				}
			}
			continue
		}

		if welfare.CanApplyNow {
			count++
		}
	}

	return count
}
