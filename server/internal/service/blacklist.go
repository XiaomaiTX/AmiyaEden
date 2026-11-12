package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/pkg/eve/esi"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Error messages are i18n keys; the handler returns them as they are and never
// exposes the wrapped cause.
var (
	ErrBlacklistSourceUnavailable   = errors.New("blacklist.errors.sourceUnavailable")
	ErrBlacklistSourceLayoutChanged = errors.New("blacklist.errors.sourceLayoutChanged")
	ErrBlacklistEmptyResult         = errors.New("blacklist.errors.emptyResult")
	ErrBlacklistRefreshInProgress   = errors.New("blacklist.errors.refreshInProgress")
	ErrBlacklistRefreshCooldown     = errors.New("blacklist.errors.refreshCooldown")
	ErrBlacklistInternal            = errors.New("blacklist.errors.internal")
	ErrBlacklistInvalidQuery        = errors.New("blacklist.errors.invalidQuery")
)

const (
	blacklistAuditCategory   = "blacklist"
	blacklistAuditAction     = "blacklist_refresh"
	blacklistRefreshInterval = 24 * time.Hour
)

// blacklistRefreshMu is process-wide so that every service instance shares it.
// The deployment runs a single instance; two concurrent refreshes from
// different instances would still converge on the same sheet snapshot.
var blacklistRefreshMu sync.Mutex

type BlacklistService struct {
	repo       *repository.BlacklistRepository
	userRepo   *repository.UserRepository
	auditSvc   *AuditService
	esiClient  *esi.Client
	httpClient *http.Client
	sheetURL   string
	// ESI retry pacing and lookup budget; tests shorten them.
	esiRetryDelays  []time.Duration
	esiLookupBudget time.Duration
	now             func() time.Time
}

func NewBlacklistService() *BlacklistService {
	return &BlacklistService{
		repo:       repository.NewBlacklistRepository(),
		userRepo:   repository.NewUserRepository(),
		auditSvc:   NewAuditService(),
		esiClient:  newConfiguredESIClient(),
		httpClient: &http.Client{Timeout: blacklistSheetTimeout},
		sheetURL:   blacklistSheetURL,

		esiRetryDelays:  blacklistESIRetryDelays,
		esiLookupBudget: blacklistESILookupBudget,
		now:             time.Now,
	}
}

type BlacklistRefreshSummary struct {
	FinishedAt        time.Time  `json:"finished_at"`
	OperatorUserID    uint       `json:"operator_user_id"`
	OperatorName      string     `json:"operator_name"`
	Characters        int        `json:"characters"`
	QQs               int        `json:"qqs"`
	Discords          int        `json:"discords"`
	Added             int        `json:"added"`
	Updated           int        `json:"updated"`
	Removed           int        `json:"removed"`
	NewUnresolved     int        `json:"new_unresolved"`
	SkippedUnresolved int        `json:"skipped_unresolved"`
	LookupFailed      int        `json:"lookup_failed"`
	NextRefreshAt     *time.Time `json:"next_refresh_at"`
}

// Refresh mirrors "The List" into the local check sets. Download and ESI
// lookups happen before the transaction; any failure before commit leaves
// every table untouched. ESI lookup failures do not fail the refresh: the
// affected names are deferred to the next refresh (see resolveBlacklistCharacterNames).
func (s *BlacklistService) Refresh(ctx context.Context, operatorUserID uint) (*BlacklistRefreshSummary, error) {
	if !blacklistRefreshMu.TryLock() {
		return nil, ErrBlacklistRefreshInProgress
	}
	defer blacklistRefreshMu.Unlock()

	latest, err := s.repo.LatestRefreshRun()
	if err != nil {
		return nil, fmt.Errorf("%w: latest refresh run: %v", ErrBlacklistInternal, err)
	}
	if latest != nil {
		if next := blacklistNextRefreshAt(*latest); next != nil && s.now().Before(*next) {
			return nil, ErrBlacklistRefreshCooldown
		}
	}

	summary, esiLookups, err := s.refresh(ctx, operatorUserID)
	if err != nil {
		if global.Logger != nil {
			global.Logger.Warn("联盟黑名单刷新失败", zap.Uint("operator_user_id", operatorUserID), zap.Error(err))
		}
		return nil, err
	}
	if global.Logger != nil {
		global.Logger.Info("联盟黑名单刷新完成",
			zap.Uint("operator_user_id", operatorUserID),
			zap.Int("esi_lookups", esiLookups),
			zap.Int("characters", summary.Characters),
			zap.Int("added", summary.Added),
			zap.Int("updated", summary.Updated),
			zap.Int("removed", summary.Removed),
			zap.Int("lookup_failed", summary.LookupFailed))
	}
	return summary, nil
}

func (s *BlacklistService) refresh(ctx context.Context, operatorUserID uint) (*BlacklistRefreshSummary, int, error) {
	data, err := downloadBlacklistSheet(ctx, s.httpClient, s.sheetURL)
	if err != nil {
		return nil, 0, err
	}
	rows, err := parseBlacklistSheet(data)
	if err != nil {
		return nil, 0, err
	}
	plan, err := s.resolveNames(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	drafts := buildBlacklistDrafts(rows, plan.resolved)
	run := model.BlacklistRefreshRun{OperatorUserID: operatorUserID}
	for _, draft := range drafts {
		switch draft.kind {
		case model.BlacklistKindCharacter:
			run.Characters++
		case model.BlacklistKindQQ:
			run.QQs++
		case model.BlacklistKindDiscord:
			run.Discords++
		}
	}
	if run.Characters == 0 {
		return nil, 0, ErrBlacklistEmptyResult
	}
	run.SkippedUnresolved = plan.skipped
	run.LookupFailed = plan.lookupFailed

	err = global.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.applyEntriesTx(tx, drafts, &run); err != nil {
			return err
		}
		if err := s.applyUnresolvedTx(tx, plan, &run); err != nil {
			return err
		}
		run.FinishedAt = s.now()
		if err := s.repo.CreateRefreshRunTx(tx, &run); err != nil {
			return err
		}
		return s.auditSvc.RecordRequiredEventTx(tx, AuditRecordInput{
			Category:     blacklistAuditCategory,
			Action:       blacklistAuditAction,
			ActorUserID:  operatorUserID,
			ResourceType: "blacklist_refresh_run",
			ResourceID:   strconv.FormatUint(uint64(run.ID), 10),
			Result:       model.AuditResultSuccess,
			// Counts only: entries carry personal contact details.
			Details: map[string]any{
				"characters":         run.Characters,
				"qqs":                run.QQs,
				"discords":           run.Discords,
				"added":              run.Added,
				"updated":            run.Updated,
				"removed":            run.Removed,
				"new_unresolved":     run.NewUnresolved,
				"skipped_unresolved": run.SkippedUnresolved,
				"lookup_failed":      run.LookupFailed,
			},
		})
	})
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrBlacklistInternal, err)
	}
	operatorName, err := s.operatorName(operatorUserID)
	if err != nil && global.Logger != nil {
		global.Logger.Warn("读取黑名单刷新操作人失败", zap.Uint("operator_user_id", operatorUserID), zap.Error(err))
	}
	return blacklistSummaryFromRun(run, operatorName), plan.esiLookups, nil
}

type blacklistCharacterRef struct {
	Value   string // character_id as a decimal string
	Display string // canonical name
}

type blacklistNamePlan struct {
	resolved   map[string]blacklistCharacterRef // lowercased sheet name -> character
	candidates map[string]struct{}              // every lowercased name on the sheet
	unresolved []model.BlacklistUnresolvedName  // names that failed this time and are not cached yet
	skipped    int
	esiLookups int
	// Names whose ESI lookup failed or ran out of budget. They get no entry
	// and no cache row, so the next refresh looks them up again.
	lookupFailed int
}

// resolveNames applies D4: reuse IDs of characters already on the blacklist,
// skip cached unresolved names, drop malformed names, and ask ESI for the rest.
func (s *BlacklistService) resolveNames(ctx context.Context, rows []blacklistSheetRow) (*blacklistNamePlan, error) {
	entries, err := s.repo.ListAllEntriesTx(global.DB)
	if err != nil {
		return nil, fmt.Errorf("%w: list entries: %v", ErrBlacklistInternal, err)
	}
	cached, err := s.repo.ListUnresolvedNamesTx(global.DB)
	if err != nil {
		return nil, fmt.Errorf("%w: list unresolved names: %v", ErrBlacklistInternal, err)
	}
	known := make(map[string]blacklistCharacterRef, len(entries))
	for _, entry := range entries {
		if entry.Kind == model.BlacklistKindCharacter {
			known[strings.ToLower(entry.Display)] = blacklistCharacterRef{Value: entry.Value, Display: entry.Display}
		}
	}
	cachedKeys := make(map[string]struct{}, len(cached))
	for _, row := range cached {
		cachedKeys[row.NameKey] = struct{}{}
	}

	plan := &blacklistNamePlan{resolved: map[string]blacklistCharacterRef{}, candidates: map[string]struct{}{}}
	var lookup []string
	for _, row := range rows {
		for _, name := range row.Names {
			key := strings.ToLower(name)
			if _, seen := plan.candidates[key]; seen {
				continue
			}
			plan.candidates[key] = struct{}{}
			if ref, ok := known[key]; ok {
				plan.resolved[key] = ref
				continue
			}
			if _, ok := cachedKeys[key]; ok {
				plan.skipped++
				continue
			}
			if !validBlacklistCharacterName(name) {
				plan.unresolved = append(plan.unresolved, model.BlacklistUnresolvedName{NameKey: key, Name: name, Kind: model.BlacklistUnresolvedInvalidFormat})
				continue
			}
			lookup = append(lookup, name)
		}
	}

	lookups := resolveBlacklistCharacterNames(ctx, s.esiClient, lookup, s.esiRetryDelays, s.esiLookupBudget)
	plan.esiLookups = len(lookup)
	plan.lookupFailed = len(lookups.deferred)
	for key, character := range lookups.resolved {
		plan.resolved[key] = blacklistCharacterRef{Value: strconv.FormatInt(character.ID, 10), Display: character.Name}
	}
	for _, name := range lookups.notFound {
		plan.unresolved = append(plan.unresolved, model.BlacklistUnresolvedName{NameKey: strings.ToLower(name), Name: name, Kind: model.BlacklistUnresolvedNotFound})
	}
	return plan, nil
}

type blacklistDraft struct {
	kind    string
	value   string
	display string
	reasons []string
}

func blacklistKey(kind, value string) string { return kind + "\x00" + value }

// buildBlacklistDrafts produces one draft per (kind, value) in sheet order.
// The first spelling seen wins the display; reasons are deduplicated in order
// and empty reasons are ignored.
func buildBlacklistDrafts(rows []blacklistSheetRow, resolved map[string]blacklistCharacterRef) []*blacklistDraft {
	var drafts []*blacklistDraft
	index := map[string]*blacklistDraft{}
	add := func(kind, value, display, reason string) {
		key := blacklistKey(kind, value)
		draft, ok := index[key]
		if !ok {
			draft = &blacklistDraft{kind: kind, value: value, display: display}
			index[key] = draft
			drafts = append(drafts, draft)
		}
		if reason != "" && !slices.Contains(draft.reasons, reason) {
			draft.reasons = append(draft.reasons, reason)
		}
	}
	for _, row := range rows {
		for _, name := range row.Names {
			if ref, ok := resolved[strings.ToLower(name)]; ok {
				add(model.BlacklistKindCharacter, ref.Value, ref.Display, row.Reason)
			}
		}
		for _, qq := range row.QQs {
			add(model.BlacklistKindQQ, qq, qq, row.Reason)
		}
		for _, discord := range row.Discords {
			add(model.BlacklistKindDiscord, discord.Value, discord.Display, row.Reason)
		}
	}
	return drafts
}

func (s *BlacklistService) applyEntriesTx(tx *gorm.DB, drafts []*blacklistDraft, run *model.BlacklistRefreshRun) error {
	existing, err := s.repo.ListAllEntriesTx(tx)
	if err != nil {
		return err
	}
	byKey := make(map[string]model.BlacklistEntry, len(existing))
	for _, entry := range existing {
		byKey[blacklistKey(entry.Kind, entry.Value)] = entry
	}
	kept := make(map[uint]struct{}, len(drafts))
	var created []model.BlacklistEntry
	for _, draft := range drafts {
		reason := strings.Join(draft.reasons, "\n")
		old, ok := byKey[blacklistKey(draft.kind, draft.value)]
		if !ok {
			created = append(created, model.BlacklistEntry{Kind: draft.kind, Value: draft.value, Display: draft.display, Reason: reason})
			continue
		}
		kept[old.ID] = struct{}{}
		if old.Display != draft.display || old.Reason != reason {
			if err := s.repo.UpdateEntryTx(tx, old.ID, draft.display, reason); err != nil {
				return err
			}
			run.Updated++
		}
	}
	var removed []uint
	for _, entry := range existing {
		if _, ok := kept[entry.ID]; !ok {
			removed = append(removed, entry.ID)
		}
	}
	if err := s.repo.CreateEntriesTx(tx, created); err != nil {
		return err
	}
	if err := s.repo.DeleteEntriesTx(tx, removed); err != nil {
		return err
	}
	run.Added = len(created)
	run.Removed = len(removed)
	return nil
}

// applyUnresolvedTx caches this run's new failures and forgets names that are
// no longer on the sheet, so a corrected spelling is looked up again.
func (s *BlacklistService) applyUnresolvedTx(tx *gorm.DB, plan *blacklistNamePlan, run *model.BlacklistRefreshRun) error {
	existing, err := s.repo.ListUnresolvedNamesTx(tx)
	if err != nil {
		return err
	}
	present := make(map[string]struct{}, len(existing))
	var stale []uint
	for _, row := range existing {
		if _, onSheet := plan.candidates[row.NameKey]; onSheet {
			present[row.NameKey] = struct{}{}
		} else {
			stale = append(stale, row.ID)
		}
	}
	var fresh []model.BlacklistUnresolvedName
	for _, row := range plan.unresolved {
		if _, ok := present[row.NameKey]; !ok {
			fresh = append(fresh, row)
		}
	}
	if err := s.repo.CreateUnresolvedNamesTx(tx, fresh); err != nil {
		return err
	}
	if err := s.repo.DeleteUnresolvedNamesTx(tx, stale); err != nil {
		return err
	}
	run.NewUnresolved = len(fresh)
	return nil
}

// ListEntries pages through the check entries. An empty kind lists all three
// sets; any other value must name one of them.
func (s *BlacklistService) ListEntries(page, size int, kind, keyword string) ([]model.BlacklistEntry, int64, error) {
	switch kind {
	case "", model.BlacklistKindCharacter, model.BlacklistKindQQ, model.BlacklistKindDiscord:
	default:
		return nil, 0, ErrBlacklistInvalidQuery
	}
	rows, total, err := s.repo.List(page, size, repository.BlacklistEntryFilter{Kind: kind, Keyword: keyword})
	if err != nil {
		return nil, 0, fmt.Errorf("%w: list entries: %v", ErrBlacklistInternal, err)
	}
	return rows, total, nil
}

// LatestSummary returns the most recent successful refresh, or nil when the
// blacklist has never been refreshed.
func (s *BlacklistService) LatestSummary() (*BlacklistRefreshSummary, error) {
	run, err := s.repo.LatestRefreshRun()
	if err != nil {
		return nil, fmt.Errorf("%w: latest refresh run: %v", ErrBlacklistInternal, err)
	}
	if run == nil {
		return nil, nil
	}
	operatorName, err := s.operatorName(run.OperatorUserID)
	if err != nil {
		return nil, fmt.Errorf("%w: refresh operator: %v", ErrBlacklistInternal, err)
	}
	return blacklistSummaryFromRun(*run, operatorName), nil
}

func (s *BlacklistService) operatorName(userID uint) (string, error) {
	user, err := s.userRepo.GetByID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return user.Nickname, nil
}

func blacklistSummaryFromRun(run model.BlacklistRefreshRun, operatorName string) *BlacklistRefreshSummary {
	return &BlacklistRefreshSummary{
		FinishedAt:        run.FinishedAt,
		OperatorUserID:    run.OperatorUserID,
		OperatorName:      operatorName,
		Characters:        run.Characters,
		QQs:               run.QQs,
		Discords:          run.Discords,
		Added:             run.Added,
		Updated:           run.Updated,
		Removed:           run.Removed,
		NewUnresolved:     run.NewUnresolved,
		SkippedUnresolved: run.SkippedUnresolved,
		LookupFailed:      run.LookupFailed,
		NextRefreshAt:     blacklistNextRefreshAt(run),
	}
}

// A partial refresh must stay retryable so that first-time imports and ESI
// outages can make progress. Completed refreshes are limited across operators
// and process restarts by the persisted latest run, before any outbound call.
func blacklistNextRefreshAt(run model.BlacklistRefreshRun) *time.Time {
	if run.LookupFailed > 0 {
		return nil
	}
	next := run.FinishedAt.Add(blacklistRefreshInterval)
	return &next
}
