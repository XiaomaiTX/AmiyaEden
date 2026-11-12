package service

import (
	"amiya-eden/global"
	"amiya-eden/pkg/eve/esi"
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ESI accepts up to 500 names per POST /universe/ids/, but large batches
// regularly time out at ESI's gateway (504) on a cold refresh.
const blacklistESIBatchSize = 50

const blacklistESILookupBudget = 60 * time.Second

var blacklistESIRetryDelays = []time.Duration{time.Second, 2 * time.Second}

type blacklistResolvedCharacter struct {
	ID   int64
	Name string // canonical spelling from ESI
}

type blacklistLookupResult struct {
	resolved map[string]blacklistResolvedCharacter // lowercased name -> character
	notFound []string                              // ESI answered and listed no such character
	deferred []string                              // lookup failed or ran out of budget; retry next refresh
}

// resolveBlacklistCharacterNames looks names up through ESI's public
// name-to-ID endpoint. A name counts as not found only when ESI answered its
// batch without listing it under "characters" with exactly that name (ignoring
// case); the endpoint also returns loose matches for corporations and
// alliances, which never count.
//
// A batch that still fails after its retries, or that is not answered before
// the budget runs out, is deferred rather than treated as not found, so an ESI
// outage never poisons the unresolved-name cache.
func resolveBlacklistCharacterNames(ctx context.Context, client *esi.Client, names []string, retryDelays []time.Duration, budget time.Duration) blacklistLookupResult {
	result := blacklistLookupResult{resolved: make(map[string]blacklistResolvedCharacter, len(names))}
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	for start := 0; start < len(names); start += blacklistESIBatchSize {
		batch := names[start:min(start+blacklistESIBatchSize, len(names))]
		if ctx.Err() != nil {
			result.deferred = append(result.deferred, names[start:]...)
			break
		}
		characters, err := lookupBlacklistBatch(ctx, client, batch, retryDelays)
		if err != nil {
			if global.Logger != nil {
				global.Logger.Warn("黑名单人名批次查询失败，下次刷新重试", zap.Int("names", len(batch)), zap.Error(err))
			}
			result.deferred = append(result.deferred, batch...)
			continue
		}
		for key, character := range characters {
			result.resolved[key] = character
		}
		for _, name := range batch {
			if _, ok := characters[strings.ToLower(name)]; !ok {
				result.notFound = append(result.notFound, name)
			}
		}
	}
	return result
}

// lookupBlacklistBatch posts one batch, retrying server errors, rate limits
// and transport errors after each delay in retryDelays.
func lookupBlacklistBatch(ctx context.Context, client *esi.Client, batch []string, retryDelays []time.Duration) (map[string]blacklistResolvedCharacter, error) {
	for attempt := 0; ; attempt++ {
		var payload struct {
			Characters []struct {
				ID   int64  `json:"id"`
				Name string `json:"name"`
			} `json:"characters"`
		}
		err := client.PostJSON(ctx, "/universe/ids/?datasource=tranquility", "", batch, &payload)
		if err == nil {
			characters := make(map[string]blacklistResolvedCharacter, len(payload.Characters))
			for _, character := range payload.Characters {
				if character.ID > 0 && strings.TrimSpace(character.Name) != "" {
					characters[strings.ToLower(character.Name)] = blacklistResolvedCharacter{ID: character.ID, Name: character.Name}
				}
			}
			return characters, nil
		}
		if attempt >= len(retryDelays) || ctx.Err() != nil || !retryableBlacklistESIError(err) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, err
		case <-time.After(retryDelays[attempt]):
		}
	}
}

func retryableBlacklistESIError(err error) bool {
	var httpErr *esi.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= 500 || httpErr.StatusCode == http.StatusTooManyRequests
	}
	// The HTTP client's per-request timeout also wraps DeadlineExceeded. It
	// can be retried while the shared lookup context still has time left.
	return !errors.Is(err, context.Canceled)
}
