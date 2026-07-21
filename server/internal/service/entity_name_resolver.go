package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/pkg/eve/esi"
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	entityNameCacheTTL      = 7 * 24 * time.Hour
	entityNameResolveBatch  = 500
	entityNameSingleflightT = 30 * time.Second
)

type EntityNameResolveResult struct {
	Names map[int64]string
	Miss  []int64
}

type entityNameEntry struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	// ESI /universe/names 返回的 category 可用于识别类型，未知时保持 unknown。
	Category string `json:"category"`
}

type entityNameInFlight struct {
	done   chan struct{}
	result map[int64]string
	err    error
	start  time.Time
}

type EntityNameResolver struct {
	repo   *repository.EntityNameCacheRepository
	client *esi.Client

	mu       sync.Mutex
	inFlight map[string]*entityNameInFlight
}

func NewEntityNameResolver() *EntityNameResolver {
	client := esi.NewClient()
	if global.Config != nil {
		client = esi.NewClientWithConfig(global.Config.EveSSO.ESIBaseURL, global.Config.EveSSO.ESIAPIPrefix)
	}
	return &EntityNameResolver{
		repo:     repository.NewEntityNameCacheRepository(),
		client:   client,
		inFlight: make(map[string]*entityNameInFlight),
	}
}

func (r *EntityNameResolver) Resolve(ctx context.Context, ids []int64) EntityNameResolveResult {
	result := EntityNameResolveResult{
		Names: make(map[int64]string),
		Miss:  make([]int64, 0),
	}
	if len(ids) == 0 {
		return result
	}

	normalized := uniquePositiveInt64(ids)
	if len(normalized) == 0 {
		return result
	}

	now := time.Now()
	cachedRows, err := r.repo.ListByEntityIDs(normalized)
	if err != nil {
		if global.Logger != nil {
			global.Logger.Warn("[EntityNameResolver] load cache failed", zap.Error(err))
		}
	}

	cachedFresh := make(map[int64]model.EveEntityNameCache, len(cachedRows))
	for _, row := range cachedRows {
		if !row.ExpiresAt.Before(now) && row.Name != "" {
			result.Names[row.EntityID] = row.Name
			cachedFresh[row.EntityID] = row
		}
	}

	toResolve := make([]int64, 0, len(normalized))
	for _, id := range normalized {
		if _, ok := result.Names[id]; !ok {
			toResolve = append(toResolve, id)
		}
	}

	if len(toResolve) > 0 {
		resolved, resolveErr := r.resolveMissingWithSingleflight(ctx, toResolve)
		if resolveErr != nil {
			if global.Logger != nil {
				global.Logger.Warn("[EntityNameResolver] resolve via ESI failed", zap.Error(resolveErr))
			}
		}
		for id, name := range resolved {
			result.Names[id] = name
		}
	}

	for _, id := range normalized {
		if _, ok := result.Names[id]; !ok {
			result.Miss = append(result.Miss, id)
		}
	}
	sort.Slice(result.Miss, func(i, j int) bool { return result.Miss[i] < result.Miss[j] })

	if global.Logger != nil {
		global.Logger.Debug("[EntityNameResolver] resolve finished",
			zap.Int("requested", len(normalized)),
			zap.Int("hit", len(result.Names)),
			zap.Int("miss", len(result.Miss)),
		)
	}
	return result
}

func uniquePositiveInt64(ids []int64) []int64 {
	set := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := set[id]; exists {
			continue
		}
		set[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func buildResolveKey(ids []int64) string {
	return fmt.Sprintf("%v", ids)
}

func (r *EntityNameResolver) resolveMissingWithSingleflight(ctx context.Context, ids []int64) (map[int64]string, error) {
	key := buildResolveKey(ids)
	r.mu.Lock()
	if existing, ok := r.inFlight[key]; ok {
		done := existing.done
		r.mu.Unlock()
		select {
		case <-done:
			return existing.result, existing.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	entry := &entityNameInFlight{
		done:   make(chan struct{}),
		result: make(map[int64]string),
		start:  time.Now(),
	}
	r.inFlight[key] = entry
	r.mu.Unlock()

	entry.result, entry.err = r.resolveMissing(ctx, ids)

	r.mu.Lock()
	delete(r.inFlight, key)
	close(entry.done)
	r.mu.Unlock()

	if elapsed := time.Since(entry.start); elapsed > entityNameSingleflightT && global.Logger != nil {
		global.Logger.Warn("[EntityNameResolver] singleflight resolve took too long",
			zap.Duration("elapsed", elapsed),
			zap.Int("ids", len(ids)),
		)
	}

	return entry.result, entry.err
}

func (r *EntityNameResolver) resolveMissing(ctx context.Context, ids []int64) (map[int64]string, error) {
	resolved := make(map[int64]string, len(ids))
	toUpsert := make([]model.EveEntityNameCache, 0, len(ids))
	var firstErr error
	batchCount := 0

	for start := 0; start < len(ids); start += entityNameResolveBatch {
		end := start + entityNameResolveBatch
		if end > len(ids) {
			end = len(ids)
		}
		batchCount++
		chunk := ids[start:end]

		var entries []entityNameEntry
		err := r.client.PostJSON(ctx, "/universe/names?datasource=tranquility", "", chunk, &entries)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if global.Logger != nil {
				global.Logger.Warn("[EntityNameResolver] ESI batch resolve failed",
					zap.Error(err),
					zap.Int("batch", batchCount),
					zap.Int("size", len(chunk)),
				)
			}
			continue
		}

		now := time.Now()
		for _, item := range entries {
			if item.ID <= 0 || item.Name == "" {
				continue
			}
			resolved[item.ID] = item.Name
			toUpsert = append(toUpsert, model.EveEntityNameCache{
				EntityID:       item.ID,
				EntityType:     normalizeEntityType(item.Category),
				Name:           item.Name,
				Source:         model.EntityNameSourceESI,
				LastResolvedAt: now,
				ExpiresAt:      now.Add(entityNameCacheTTL),
			})
		}
	}

	if err := r.repo.Upsert(toUpsert); err != nil {
		if global.Logger != nil {
			global.Logger.Warn("[EntityNameResolver] upsert cache failed", zap.Error(err))
		}
	}

	if global.Logger != nil {
		global.Logger.Debug("[EntityNameResolver] ESI resolve summary",
			zap.Int("resolved", len(resolved)),
			zap.Int("requested", len(ids)),
			zap.Int("batches", batchCount),
		)
	}
	return resolved, firstErr
}

func normalizeEntityType(category string) string {
	switch category {
	case model.EntityNameTypeCharacter, model.EntityNameTypeCorporation, model.EntityNameTypeAlliance:
		return category
	default:
		return model.EntityNameTypeUnknown
	}
}

// PrimeCorporationNames upserts corporation id→name pairs that were resolved
// from another authoritative ESI source (e.g. POST /universe/ids/), so later
// Resolve calls can hit the cache instead of re-querying ESI.
func (r *EntityNameResolver) PrimeCorporationNames(items []CorporationDisplay) {
	if r == nil || len(items) == 0 {
		return
	}
	toUpsert := make([]model.EveEntityNameCache, 0, len(items))
	now := time.Now()
	for _, item := range items {
		if item.CorporationID <= 0 || item.CorporationName == "" {
			continue
		}
		toUpsert = append(toUpsert, model.EveEntityNameCache{
			EntityID:       item.CorporationID,
			EntityType:     model.EntityNameTypeCorporation,
			Name:           item.CorporationName,
			Source:         model.EntityNameSourceESI,
			LastResolvedAt: now,
			ExpiresAt:      now.Add(entityNameCacheTTL),
		})
	}
	if len(toUpsert) == 0 {
		return
	}
	repo := r.repo
	if repo == nil {
		repo = repository.NewEntityNameCacheRepository()
	}
	if err := repo.Upsert(toUpsert); err != nil {
		if global.Logger != nil {
			global.Logger.Warn("[EntityNameResolver] prime cache failed", zap.Error(err))
		}
	}
}
