package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type blacklistTestEnv struct {
	t   *testing.T
	svc *BlacklistService
	esi *fakeBlacklistESI

	mu          sync.Mutex
	sheet       string
	sheetStatus int
	onServe     func()
	clock       time.Time
}

func newBlacklistTestEnv(t *testing.T, withAudit bool) *blacklistTestEnv {
	t.Helper()
	models := []any{&model.BlacklistEntry{}, &model.BlacklistUnresolvedName{}, &model.BlacklistRefreshRun{}, &model.User{}}
	if withAudit {
		models = append(models, &model.AuditEvent{})
	}
	db := newServiceTestDB(t, "blacklist", models...)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	env := &blacklistTestEnv{t: t, esi: &fakeBlacklistESI{characters: map[string]int64{}}, clock: time.Now()}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		env.mu.Lock()
		body, status, onServe := env.sheet, env.sheetStatus, env.onServe
		env.mu.Unlock()
		if onServe != nil {
			onServe()
		}
		if status != 0 {
			http.Error(w, "unavailable", status)
			return
		}
		_, _ = fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)
	env.svc = &BlacklistService{
		repo:       repository.NewBlacklistRepository(),
		userRepo:   repository.NewUserRepository(),
		auditSvc:   NewAuditService(),
		esiClient:  newFakeBlacklistESIClient(t, env.esi),
		httpClient: server.Client(),
		sheetURL:   server.URL,

		esiRetryDelays:  testBlacklistRetryDelays,
		esiLookupBudget: time.Minute,
		now:             func() time.Time { return env.clock },
	}
	return env
}

func (e *blacklistTestEnv) setSheet(rows ...[]string) {
	e.t.Helper()
	var buf bytes.Buffer
	buf.WriteString(blacklistTestHeader)
	writer := csv.NewWriter(&buf)
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			e.t.Fatal(err)
		}
	}
	writer.Flush()
	e.mu.Lock()
	e.sheet = buf.String()
	e.mu.Unlock()
}

func (e *blacklistTestEnv) setRawSheet(body string) {
	e.mu.Lock()
	e.sheet = body
	e.mu.Unlock()
}

func (e *blacklistTestEnv) refresh(operatorUserID uint) *BlacklistRefreshSummary {
	e.t.Helper()
	summary, err := e.svc.Refresh(context.Background(), operatorUserID)
	if err != nil {
		e.t.Fatalf("refresh: %v", err)
	}
	// Existing sheet-alignment tests model successive daily snapshots.
	e.clock = e.clock.Add(blacklistRefreshInterval)
	return summary
}

func TestBlacklistRefreshCooldown(t *testing.T) {
	env := newBlacklistTestEnv(t, true)
	env.esi.characters["pilot one"] = 90000001
	env.setSheet(blacklistRow("Pilot One", "Reason", "", "", ""))
	first, err := env.svc.Refresh(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	wantNext := first.FinishedAt.Add(24 * time.Hour)
	if first.NextRefreshAt == nil || !first.NextRefreshAt.Equal(wantNext) {
		t.Fatalf("next refresh: %+v", first)
	}
	before := blacklistSnapshot(t)
	env.onServe = func() { t.Error("cooldown must prevent the sheet download") }
	env.clock = wantNext.Add(-time.Nanosecond)
	if _, err := env.svc.Refresh(context.Background(), 2); !errors.Is(err, ErrBlacklistRefreshCooldown) {
		t.Fatalf("another operator bypassed cooldown: %v", err)
	}
	// Reconstruct the service to prove the limit survives a process restart.
	restarted := NewBlacklistService()
	restarted.now = env.svc.now
	if _, err := restarted.Refresh(context.Background(), 1); !errors.Is(err, ErrBlacklistRefreshCooldown) {
		t.Fatalf("restart bypassed cooldown: %v", err)
	}
	if got := blacklistSnapshot(t); !reflect.DeepEqual(got, before) || blacklistCount(t, &model.BlacklistRefreshRun{}) != 1 || blacklistCount(t, &model.AuditEvent{}) != 1 {
		t.Fatal("cooldown changed local data")
	}
	if len(env.esi.batchSizes()) != 1 {
		t.Fatal("cooldown sent ESI requests")
	}
	env.onServe = nil
	env.clock = wantNext
	env.setRawSheet("broken header")
	if _, err := env.svc.Refresh(context.Background(), 2); !errors.Is(err, ErrBlacklistSourceLayoutChanged) {
		t.Fatalf("failed refresh at cooldown boundary: %v", err)
	}
	env.setSheet(blacklistRow("Pilot One", "Reason", "", "", ""))
	if _, err := env.svc.Refresh(context.Background(), 2); err != nil {
		t.Fatalf("refresh at exactly 24 hours: %v", err)
	}
}

func TestBlacklistPartialRefreshCanRetryImmediately(t *testing.T) {
	env := newBlacklistTestEnv(t, true)
	env.esi.characters["pilot one"] = 90000001
	env.esi.characters["late pilot"] = 90000002
	env.esi.failNames = map[string]bool{"pilot 000": true}
	var names []string
	for i := 0; i < 50; i++ {
		names = append(names, fmt.Sprintf("Pilot %03d", i))
	}
	names = append(names, "Pilot One", "Late Pilot")
	env.setSheet(blacklistRow(strings.Join(names, ","), "Reason", "", "", ""))
	// First batch is deferred; the last batch resolves enough to commit.
	partial, err := env.svc.Refresh(context.Background(), 1)
	if err != nil || partial.LookupFailed != 50 || partial.NextRefreshAt != nil {
		t.Fatalf("partial refresh: %+v %v", partial, err)
	}
	env.esi.failNames = nil
	complete, err := env.svc.Refresh(context.Background(), 2)
	if err != nil || complete.LookupFailed != 0 || complete.NextRefreshAt == nil {
		t.Fatalf("immediate retry: %+v %v", complete, err)
	}
}

// blacklistRow lays out one sheet row: main name, reason, alt1, Discord, QQ.
func blacklistRow(main, reason, alt, discord, qq string) []string {
	return []string{main, "2024-01-01", "tester", reason, "", alt, "", "", discord, qq, ""}
}

// blacklistSnapshot maps "kind:value" to "display|reason".
func blacklistSnapshot(t *testing.T) map[string]string {
	t.Helper()
	var entries []model.BlacklistEntry
	if err := global.DB.Find(&entries).Error; err != nil {
		t.Fatal(err)
	}
	snapshot := make(map[string]string, len(entries))
	for _, entry := range entries {
		snapshot[entry.Kind+":"+entry.Value] = entry.Display + "|" + entry.Reason
	}
	return snapshot
}

func blacklistUnresolvedSnapshot(t *testing.T) map[string]string {
	t.Helper()
	var rows []model.BlacklistUnresolvedName
	if err := global.DB.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	snapshot := make(map[string]string, len(rows))
	for _, row := range rows {
		snapshot[row.NameKey] = row.Kind
	}
	return snapshot
}

func blacklistCount(t *testing.T, value any) int64 {
	t.Helper()
	var count int64
	if err := global.DB.Model(value).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func assertBlacklistSummaryCounts(t *testing.T, got *BlacklistRefreshSummary, want BlacklistRefreshSummary) {
	t.Helper()
	got = &BlacklistRefreshSummary{
		Characters: got.Characters, QQs: got.QQs, Discords: got.Discords,
		Added: got.Added, Updated: got.Updated, Removed: got.Removed,
		NewUnresolved: got.NewUnresolved, SkippedUnresolved: got.SkippedUnresolved,
		LookupFailed: got.LookupFailed,
	}
	if *got != want {
		t.Fatalf("summary counts\n got: %+v\nwant: %+v", *got, want)
	}
}

func TestBlacklistRefreshMirrorsSheet(t *testing.T) {
	env := newBlacklistTestEnv(t, true)
	operator := model.User{Nickname: "Ops Lead"}
	if err := global.DB.Create(&operator).Error; err != nil {
		t.Fatal(err)
	}
	env.esi.characters["pilot one"] = 90000001
	env.esi.characters["alt one"] = 90000002
	env.esi.characters["pilot two"] = 90000003
	env.setSheet(
		blacklistRow("Pilot One", "Scam", "Alt One", "@Spy_One", "10000001"),
		blacklistRow("Pilot Two", "Awox", "", "", "10000001\n10000002"),
		blacklistRow("pilot one", "Theft", "", "", ""),
		blacklistRow("", "Contact", "", "Contact Only", "30000001"),
		blacklistRow("Ghost Pilot", "Ghost", "", "ghost#0001", "40000001"),
		blacklistRow("Bad/Name", "Bad", "", "", ""),
	)

	first := env.refresh(operator.ID)
	assertBlacklistSummaryCounts(t, first, BlacklistRefreshSummary{
		Characters: 3, QQs: 4, Discords: 3, Added: 10, NewUnresolved: 2,
	})
	if first.OperatorUserID != operator.ID || first.OperatorName != "Ops Lead" || first.FinishedAt.IsZero() {
		t.Fatalf("summary operator fields = %+v", first)
	}
	if got := env.esi.batchSizes(); !reflect.DeepEqual(got, []int{4}) {
		t.Fatalf("ESI batches = %v, want one batch of 4 (malformed name never sent)", got)
	}
	wantFirst := map[string]string{
		"character:90000001":   "Pilot One|Scam\nTheft",
		"character:90000002":   "Alt One|Scam",
		"character:90000003":   "Pilot Two|Awox",
		"qq:10000001":          "10000001|Scam\nAwox",
		"qq:10000002":          "10000002|Awox",
		"qq:30000001":          "30000001|Contact",
		"qq:40000001":          "40000001|Ghost",
		"discord:spy_one":      "Spy_One|Scam",
		"discord:contact only": "Contact Only|Contact",
		"discord:ghost#0001":   "ghost#0001|Ghost",
	}
	if got := blacklistSnapshot(t); !reflect.DeepEqual(got, wantFirst) {
		t.Fatalf("entries after first refresh\n got: %v\nwant: %v", got, wantFirst)
	}
	wantUnresolved := map[string]string{
		"ghost pilot": model.BlacklistUnresolvedNotFound,
		"bad/name":    model.BlacklistUnresolvedInvalidFormat,
	}
	if got := blacklistUnresolvedSnapshot(t); !reflect.DeepEqual(got, wantUnresolved) {
		t.Fatalf("unresolved names = %v, want %v", got, wantUnresolved)
	}

	var audit model.AuditEvent
	if err := global.DB.Where("category = ? AND action = ?", "blacklist", "blacklist_refresh").First(&audit).Error; err != nil {
		t.Fatalf("audit event: %v", err)
	}
	if audit.ActorUserID != operator.ID || audit.Result != model.AuditResultSuccess {
		t.Fatalf("audit event = %+v", audit)
	}
	for _, private := range []string{"10000001", "spy_one", "Pilot One", "Scam"} {
		if strings.Contains(strings.ToLower(audit.DetailsJSON), strings.ToLower(private)) {
			t.Fatalf("audit details leak %q: %s", private, audit.DetailsJSON)
		}
	}
	if !strings.Contains(audit.DetailsJSON, `"added":10`) {
		t.Fatalf("audit details missing counts: %s", audit.DetailsJSON)
	}

	env.esi.characters["new pilot"] = 90000004
	env.setSheet(
		blacklistRow("Pilot One", "Scam", "Alt One", "@Spy_One", "10000001"),
		blacklistRow("Pilot Two", "Awox and theft", "", "", "10000001"),
		blacklistRow("Ghost Pilot", "Ghost", "", "ghost#0001", "40000001"),
		blacklistRow("Bad/Name", "Bad", "", "", ""),
		blacklistRow("New Pilot", "New", "", "@SPY_ONE", ""),
	)
	second := env.refresh(operator.ID)
	assertBlacklistSummaryCounts(t, second, BlacklistRefreshSummary{
		Characters: 4, QQs: 2, Discords: 2, Added: 1, Updated: 4, Removed: 3, SkippedUnresolved: 2,
	})
	if got := env.esi.batchSizes(); !reflect.DeepEqual(got, []int{4, 1}) {
		t.Fatalf("ESI batches = %v, want only the new name looked up", got)
	}
	wantSecond := map[string]string{
		"character:90000001": "Pilot One|Scam",
		"character:90000002": "Alt One|Scam",
		"character:90000003": "Pilot Two|Awox and theft",
		"character:90000004": "New Pilot|New",
		"qq:10000001":        "10000001|Scam\nAwox and theft",
		"qq:40000001":        "40000001|Ghost",
		"discord:spy_one":    "Spy_One|Scam\nNew",
		"discord:ghost#0001": "ghost#0001|Ghost",
	}
	if got := blacklistSnapshot(t); !reflect.DeepEqual(got, wantSecond) {
		t.Fatalf("entries after second refresh\n got: %v\nwant: %v", got, wantSecond)
	}
	if got := blacklistUnresolvedSnapshot(t); !reflect.DeepEqual(got, wantUnresolved) {
		t.Fatalf("unresolved names = %v, want %v", got, wantUnresolved)
	}
	if got := blacklistCount(t, &model.BlacklistRefreshRun{}); got != 2 {
		t.Fatalf("refresh runs = %d, want 2", got)
	}
}

func TestBlacklistRefreshSteadyStateSkipsESI(t *testing.T) {
	env := newBlacklistTestEnv(t, true)
	env.esi.characters["pilot one"] = 90000001
	env.setSheet(
		blacklistRow("Pilot One", "Scam", "", "", "10000001"),
		blacklistRow("Ghost Pilot", "Ghost", "", "", ""),
	)
	env.refresh(1)
	before := blacklistSnapshot(t)

	summary := env.refresh(1)
	assertBlacklistSummaryCounts(t, summary, BlacklistRefreshSummary{Characters: 1, QQs: 1, SkippedUnresolved: 1})
	if got := env.esi.batchSizes(); !reflect.DeepEqual(got, []int{2}) {
		t.Fatalf("ESI batches = %v, want no lookup on the second refresh", got)
	}
	if got := blacklistSnapshot(t); !reflect.DeepEqual(got, before) {
		t.Fatalf("steady-state refresh changed entries: %v -> %v", before, got)
	}
}

func TestBlacklistRefreshRetriesCorrectedSpelling(t *testing.T) {
	env := newBlacklistTestEnv(t, true)
	env.esi.characters["anchor pilot"] = 90000001
	env.esi.characters["misspelled pilot"] = 90000002
	env.setSheet(
		blacklistRow("Anchor Pilot", "Scam", "", "", ""),
		blacklistRow("Mispeled Pilot", "Spy", "", "", ""),
	)
	env.refresh(1)
	if got := blacklistUnresolvedSnapshot(t); !reflect.DeepEqual(got, map[string]string{"mispeled pilot": model.BlacklistUnresolvedNotFound}) {
		t.Fatalf("unresolved after typo = %v", got)
	}

	env.setSheet(
		blacklistRow("Anchor Pilot", "Scam", "", "", ""),
		blacklistRow("Misspelled Pilot", "Spy", "", "", ""),
	)
	summary := env.refresh(1)
	assertBlacklistSummaryCounts(t, summary, BlacklistRefreshSummary{Characters: 2, Added: 1})
	if got := env.esi.batchSizes(); !reflect.DeepEqual(got, []int{2, 1}) {
		t.Fatalf("ESI batches = %v, want the corrected name looked up", got)
	}
	if got := blacklistUnresolvedSnapshot(t); len(got) != 0 {
		t.Fatalf("stale typo not forgotten: %v", got)
	}
	if got := blacklistSnapshot(t)["character:90000002"]; got != "Misspelled Pilot|Spy" {
		t.Fatalf("corrected character entry = %q", got)
	}
}

func TestBlacklistRefreshFailuresLeaveDataUnchanged(t *testing.T) {
	env := newBlacklistTestEnv(t, true)
	env.esi.characters["pilot one"] = 90000001
	env.setSheet(
		blacklistRow("Pilot One", "Scam", "", "", "10000001"),
		blacklistRow("Ghost Pilot", "Ghost", "", "", ""),
	)
	env.refresh(1)
	entries := blacklistSnapshot(t)
	unresolved := blacklistUnresolvedSnapshot(t)

	cases := []struct {
		name    string
		arrange func()
		reset   func()
		want    error
	}{
		{
			name: "no character resolves",
			arrange: func() {
				env.setSheet(blacklistRow("Ghost Pilot", "Ghost", "", "", "20000001"), blacklistRow("", "Contact", "", "contact", "30000001"))
			},
			want: ErrBlacklistEmptyResult,
		},
		{
			name: "all new character lookups fail",
			arrange: func() {
				env.setSheet(blacklistRow("New Pilot", "New", "", "new-contact", "20000001"))
				env.esi.failStatus = http.StatusGatewayTimeout
			},
			reset: func() { env.esi.failStatus = 0 },
			want:  ErrBlacklistEmptyResult,
		},
		{
			name:    "header changed",
			arrange: func() { env.setRawSheet("Name,Reason\nPilot One,Scam\n") },
			want:    ErrBlacklistSourceLayoutChanged,
		},
		{
			name: "source down",
			arrange: func() {
				env.mu.Lock()
				env.sheetStatus = http.StatusServiceUnavailable
				env.mu.Unlock()
			},
			reset: func() {
				env.mu.Lock()
				env.sheetStatus = 0
				env.mu.Unlock()
			},
			want: ErrBlacklistSourceUnavailable,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.arrange()
			if tc.reset != nil {
				defer tc.reset()
			}
			_, err := env.svc.Refresh(context.Background(), 1)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if got := blacklistSnapshot(t); !reflect.DeepEqual(got, entries) {
				t.Fatalf("entries changed: %v", got)
			}
			if got := blacklistUnresolvedSnapshot(t); !reflect.DeepEqual(got, unresolved) {
				t.Fatalf("unresolved names changed: %v", got)
			}
			if got := blacklistCount(t, &model.BlacklistRefreshRun{}); got != 1 {
				t.Fatalf("refresh runs = %d, want 1", got)
			}
			if got := blacklistCount(t, &model.AuditEvent{}); got != 1 {
				t.Fatalf("audit events = %d, want 1", got)
			}
		})
	}
}

func TestBlacklistRefreshDefersFailedLookups(t *testing.T) {
	env := newBlacklistTestEnv(t, true)
	env.esi.characters["pilot one"] = 90000001
	env.esi.characters["late pilot"] = 90000002
	env.esi.characters["gone pilot"] = 90000003
	env.setSheet(
		blacklistRow("Pilot One", "Scam", "", "", "10000001"),
		blacklistRow("Gone Pilot", "Old", "", "", "50000001"),
	)
	env.refresh(1)

	env.setSheet(
		blacklistRow("Pilot One", "Scam and theft", "", "", "10000001"),
		blacklistRow("Late Pilot", "Spy", "", "late#0001", "20000001"),
	)
	env.esi.mu.Lock()
	env.esi.failStatus = http.StatusGatewayTimeout
	env.esi.mu.Unlock()
	failed := env.refresh(1)
	assertBlacklistSummaryCounts(t, failed, BlacklistRefreshSummary{
		Characters: 1, QQs: 2, Discords: 1, Added: 2, Updated: 2, Removed: 2, LookupFailed: 1,
	})
	wantFailed := map[string]string{
		"character:90000001": "Pilot One|Scam and theft",
		"qq:10000001":        "10000001|Scam and theft",
		"qq:20000001":        "20000001|Spy",
		"discord:late#0001":  "late#0001|Spy",
	}
	if got := blacklistSnapshot(t); !reflect.DeepEqual(got, wantFailed) {
		t.Fatalf("entries after failed lookup\n got: %v\nwant: %v", got, wantFailed)
	}
	if got := blacklistUnresolvedSnapshot(t); len(got) != 0 {
		t.Fatalf("failed lookup cached as unresolved: %v", got)
	}
	var audit model.AuditEvent
	if err := global.DB.Order("id DESC").First(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(audit.DetailsJSON, `"lookup_failed":1`) {
		t.Fatalf("audit details missing lookup_failed: %s", audit.DetailsJSON)
	}
	latest, err := env.svc.LatestSummary()
	if err != nil || latest == nil || latest.LookupFailed != 1 {
		t.Fatalf("latest summary = %+v, err = %v", latest, err)
	}

	env.esi.mu.Lock()
	env.esi.failStatus = 0
	env.esi.mu.Unlock()
	retried := env.refresh(1)
	assertBlacklistSummaryCounts(t, retried, BlacklistRefreshSummary{Characters: 2, QQs: 2, Discords: 1, Added: 1})
	if got := blacklistSnapshot(t)["character:90000002"]; got != "Late Pilot|Spy" {
		t.Fatalf("deferred name not resolved on retry: %q", got)
	}
}

func TestBlacklistRefreshCommitsAfterLookupBudgetExpires(t *testing.T) {
	env := newBlacklistTestEnv(t, true)
	env.esi.characters["pilot one"] = 90000001
	env.setSheet(blacklistRow("Pilot One", "Old", "", "", "10000001"))
	env.refresh(1)
	env.setSheet(
		blacklistRow("Pilot One", "Updated", "", "", ""),
		blacklistRow("Late Pilot", "New", "", "contact", "20000001"),
	)
	env.svc.esiLookupBudget = time.Millisecond
	env.esi.delay = 50 * time.Millisecond
	summary := env.refresh(1)
	assertBlacklistSummaryCounts(t, summary, BlacklistRefreshSummary{
		Characters: 1, QQs: 1, Discords: 1, Added: 2, Updated: 1, Removed: 1, LookupFailed: 1,
	})
	if got := blacklistUnresolvedSnapshot(t); len(got) != 0 {
		t.Fatalf("budget expiry cached as unresolved: %v", got)
	}
}

func TestBlacklistRefreshRejectsConcurrentRun(t *testing.T) {
	env := newBlacklistTestEnv(t, true)
	env.esi.characters["pilot one"] = 90000001
	env.setSheet(blacklistRow("Pilot One", "Scam", "", "", ""))
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	env.mu.Lock()
	env.onServe = func() {
		once.Do(func() { close(started) })
		<-release
	}
	env.mu.Unlock()

	done := make(chan error, 1)
	go func() {
		_, err := env.svc.Refresh(context.Background(), 1)
		done <- err
	}()
	<-started
	if _, err := NewBlacklistService().Refresh(context.Background(), 1); !errors.Is(err, ErrBlacklistRefreshInProgress) {
		t.Fatalf("concurrent refresh err = %v, want refresh in progress", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("first refresh: %v", err)
	}
}

func TestBlacklistLatestSummary(t *testing.T) {
	env := newBlacklistTestEnv(t, true)
	summary, err := env.svc.LatestSummary()
	if err != nil || summary != nil {
		t.Fatalf("never refreshed: summary=%+v err=%v", summary, err)
	}

	operator := model.User{Nickname: "Ops Lead"}
	if err := global.DB.Create(&operator).Error; err != nil {
		t.Fatal(err)
	}
	env.esi.characters["pilot one"] = 90000001
	env.esi.characters["pilot two"] = 90000002
	env.setSheet(blacklistRow("Pilot One", "Scam", "", "Spy", "10000001"), blacklistRow("Ghost Pilot", "Ghost", "", "", ""))
	env.refresh(operator.ID)
	env.setSheet(blacklistRow("Pilot One", "Scam", "", "Spy", "10000001"), blacklistRow("Pilot Two", "Awox", "", "", ""), blacklistRow("Ghost Pilot", "Ghost", "", "", ""))
	latest := env.refresh(operator.ID)

	summary, err = env.svc.LatestSummary()
	if err != nil {
		t.Fatal(err)
	}
	if summary == nil {
		t.Fatal("summary missing after refresh")
	}
	if !summary.FinishedAt.Equal(latest.FinishedAt) || summary.OperatorUserID != operator.ID || summary.OperatorName != "Ops Lead" {
		t.Fatalf("summary identity fields = %+v, want latest run by Ops Lead", summary)
	}
	assertBlacklistSummaryCounts(t, summary, BlacklistRefreshSummary{
		Characters: 2, QQs: 1, Discords: 1, Added: 1, SkippedUnresolved: 1,
	})

	rows, total, err := env.svc.ListEntries(1, 10, model.BlacklistKindCharacter, "")
	if err != nil || total != 2 || len(rows) != 2 || rows[0].Display != "Pilot One" || rows[1].Display != "Pilot Two" {
		t.Fatalf("character list = %+v total=%d err=%v", rows, total, err)
	}
	rows, total, err = env.svc.ListEntries(1, 10, "", "10000001")
	if err != nil || total != 1 || rows[0].Kind != model.BlacklistKindQQ {
		t.Fatalf("QQ keyword list = %+v total=%d err=%v", rows, total, err)
	}
	if _, _, err := env.svc.ListEntries(1, 10, "alliance", ""); !errors.Is(err, ErrBlacklistInvalidQuery) {
		t.Fatalf("unknown kind err = %v, want invalid query", err)
	}
}

func TestBlacklistLatestSummaryWithoutOperatorRecord(t *testing.T) {
	env := newBlacklistTestEnv(t, true)
	env.esi.characters["pilot one"] = 90000001
	env.setSheet(blacklistRow("Pilot One", "Scam", "", "", ""))
	env.refresh(404)
	summary, err := env.svc.LatestSummary()
	if err != nil || summary == nil || summary.OperatorUserID != 404 || summary.OperatorName != "" {
		t.Fatalf("summary = %+v err=%v, want empty operator name", summary, err)
	}
}

func TestBlacklistRefreshRollsBackWhenAuditFails(t *testing.T) {
	env := newBlacklistTestEnv(t, false) // no audit table, so the required audit write fails
	env.esi.characters["pilot one"] = 90000001
	env.setSheet(
		blacklistRow("Pilot One", "Scam", "", "", "10000001"),
		blacklistRow("Ghost Pilot", "Ghost", "", "", ""),
	)
	_, err := env.svc.Refresh(context.Background(), 1)
	if !errors.Is(err, ErrBlacklistInternal) {
		t.Fatalf("err = %v, want internal", err)
	}
	if got := blacklistCount(t, &model.BlacklistEntry{}); got != 0 {
		t.Fatalf("entries = %d, want rollback", got)
	}
	if got := blacklistCount(t, &model.BlacklistUnresolvedName{}); got != 0 {
		t.Fatalf("unresolved names = %d, want rollback", got)
	}
	if got := blacklistCount(t, &model.BlacklistRefreshRun{}); got != 0 {
		t.Fatalf("refresh runs = %d, want rollback", got)
	}
}
