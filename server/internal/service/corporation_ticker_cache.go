package service

import (
	"sync"
	"time"
)

const corporationTickerCacheTTL = 6 * time.Hour

type corporationTickerCacheEntry struct {
	ticker    string
	fetchedAt time.Time
}

var (
	corporationTickerCacheMu sync.RWMutex
	corporationTickerCache   = make(map[int64]corporationTickerCacheEntry)
)

// getCachedCorpTicker returns a cached ticker for corpID if present and fresh.
// The second return value reports whether a usable cache entry exists, regardless
// of whether the cached ticker is empty (negative caching for unresolved corps).
func getCachedCorpTicker(corpID int64) (string, bool) {
	corporationTickerCacheMu.RLock()
	entry, ok := corporationTickerCache[corpID]
	corporationTickerCacheMu.RUnlock()
	if !ok {
		return "", false
	}
	if time.Since(entry.fetchedAt) > corporationTickerCacheTTL {
		return "", false
	}
	return entry.ticker, true
}

func setCachedCorpTicker(corpID int64, ticker string) {
	corporationTickerCacheMu.Lock()
	corporationTickerCache[corpID] = corporationTickerCacheEntry{
		ticker:    ticker,
		fetchedAt: time.Now(),
	}
	corporationTickerCacheMu.Unlock()
}

// resetCorporationTickerCache clears cached tickers. Intended for tests.
func resetCorporationTickerCache() {
	corporationTickerCacheMu.Lock()
	corporationTickerCache = make(map[int64]corporationTickerCacheEntry)
	corporationTickerCacheMu.Unlock()
}
