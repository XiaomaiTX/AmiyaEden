package service

import (
	"testing"
	"time"
)

func TestCorporationTickerCache(t *testing.T) {
	resetCorporationTickerCache()
	t.Cleanup(resetCorporationTickerCache)

	if _, ok := getCachedCorpTicker(123); ok {
		t.Fatal("expected miss on empty cache")
	}

	setCachedCorpTicker(123, "FUXI")
	ticker, ok := getCachedCorpTicker(123)
	if !ok || ticker != "FUXI" {
		t.Fatalf("cache hit = %v, ticker = %q; want hit FUXI", ok, ticker)
	}

	// Negative caching: empty ticker is still a cache hit so the caller does
	// not re-issue the ESI request when the upstream resolution previously
	// produced no ticker.
	setCachedCorpTicker(456, "")
	ticker, ok = getCachedCorpTicker(456)
	if !ok || ticker != "" {
		t.Fatalf("negative cache: hit = %v, ticker = %q; want hit empty", ok, ticker)
	}
}

func TestCorporationTickerCacheExpiry(t *testing.T) {
	resetCorporationTickerCache()
	t.Cleanup(resetCorporationTickerCache)

	corporationTickerCacheMu.Lock()
	corporationTickerCache[42] = corporationTickerCacheEntry{
		ticker:    "OLD",
		fetchedAt: time.Now().Add(-corporationTickerCacheTTL - time.Minute),
	}
	corporationTickerCacheMu.Unlock()

	if _, ok := getCachedCorpTicker(42); ok {
		t.Fatal("expected expired entry to miss")
	}
}
