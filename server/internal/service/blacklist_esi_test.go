package service

import (
	"amiya-eden/pkg/eve/esi"
	"context"
	"encoding/json"
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

// fakeBlacklistESI answers POST /universe/ids/ from a fixed directory of
// characters and corporations (lowercased name -> id) and records batch sizes.
// failStatus fails every request; failFirst fails only that many requests;
// failNames fails every batch containing one of those lowercased names; delay
// slows every answer.
type fakeBlacklistESI struct {
	mu              sync.Mutex
	characters      map[string]int64
	corporations    map[string]int64
	failStatus      int
	failFirst       int
	failNames       map[string]bool
	disconnectFirst int
	delay           time.Duration
	batches         []int
}

func (f *fakeBlacklistESI) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/universe/ids/") {
			t.Errorf("unexpected ESI request %s %s", r.Method, r.URL.Path)
		}
		var names []string
		if err := json.NewDecoder(r.Body).Decode(&names); err != nil {
			t.Errorf("decode names: %v", err)
		}
		f.mu.Lock()
		f.batches = append(f.batches, len(names))
		failStatus, delay := f.failStatus, f.delay
		if f.failFirst > 0 {
			f.failFirst--
			failStatus = http.StatusGatewayTimeout
		}
		for _, name := range names {
			if f.failNames[strings.ToLower(name)] && failStatus == 0 {
				failStatus = http.StatusGatewayTimeout
			}
		}
		disconnect := f.disconnectFirst > 0
		if disconnect {
			f.disconnectFirst--
		}
		f.mu.Unlock()
		if disconnect {
			connection, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("disconnect ESI request: %v", err)
				return
			}
			_ = connection.Close()
			return
		}
		if delay > 0 {
			time.Sleep(delay)
		}
		if failStatus != 0 {
			http.Error(w, `{"error":"boom"}`, failStatus)
			return
		}
		type item struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		}
		payload := map[string][]item{}
		for _, name := range names {
			key := strings.ToLower(name)
			if id, ok := f.characters[key]; ok {
				payload["characters"] = append(payload["characters"], item{ID: id, Name: canonicalBlacklistTestName(key)})
			}
			if id, ok := f.corporations[key]; ok {
				payload["corporations"] = append(payload["corporations"], item{ID: id, Name: name})
			}
		}
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func (f *fakeBlacklistESI) batchSizes() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int(nil), f.batches...)
}

// canonicalBlacklistTestName title-cases each word, standing in for ESI's
// canonical spelling.
func canonicalBlacklistTestName(lower string) string {
	words := strings.Fields(lower)
	for i, word := range words {
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func newFakeBlacklistESIClient(t *testing.T, fake *fakeBlacklistESI) *esi.Client {
	t.Helper()
	server := httptest.NewServer(fake.handler(t))
	t.Cleanup(server.Close)
	return esi.NewClientWithConfig(server.URL, "")
}

// testBlacklistRetryDelays keeps retries instant in tests.
var testBlacklistRetryDelays = []time.Duration{time.Millisecond, time.Millisecond}

func resolveBlacklistTestNames(client *esi.Client, names []string) blacklistLookupResult {
	return resolveBlacklistCharacterNames(context.Background(), client, names, testBlacklistRetryDelays, time.Minute)
}

func TestResolveBlacklistCharacterNamesBatchesAndMapsCase(t *testing.T) {
	fake := &fakeBlacklistESI{
		characters:   map[string]int64{},
		corporations: map[string]int64{"placeholder corp": 98000001},
	}
	names := make([]string, 0, blacklistESIBatchSize+1)
	for i := range blacklistESIBatchSize - 1 {
		name := fmt.Sprintf("pilot %03d", i)
		fake.characters[name] = int64(90000000 + i)
		names = append(names, strings.ToUpper(name))
	}
	fake.characters["last pilot"] = 91000000
	names = append(names, "Placeholder Corp", "lAsT pIlOt")
	client := newFakeBlacklistESIClient(t, fake)

	got := resolveBlacklistTestNames(client, names)
	if sizes := fake.batchSizes(); !reflect.DeepEqual(sizes, []int{50, 1}) {
		t.Fatalf("batch sizes = %v, want [50 1]", sizes)
	}
	if len(got.resolved) != blacklistESIBatchSize {
		t.Fatalf("resolved %d names, want %d", len(got.resolved), blacklistESIBatchSize)
	}
	if character := got.resolved["last pilot"]; character != (blacklistResolvedCharacter{ID: 91000000, Name: "Last Pilot"}) {
		t.Fatalf("case-insensitive mapping = %+v", character)
	}
	if character := got.resolved["pilot 007"]; character.ID != 90000007 || character.Name != "Pilot 007" {
		t.Fatalf("first batch mapping = %+v", character)
	}
	if !reflect.DeepEqual(got.notFound, []string{"Placeholder Corp"}) {
		t.Fatalf("notFound = %v, want the corporation name only", got.notFound)
	}
	if len(got.deferred) != 0 {
		t.Fatalf("deferred = %v, want none", got.deferred)
	}
}

func TestResolveBlacklistCharacterNamesExactBatch(t *testing.T) {
	fake := &fakeBlacklistESI{characters: map[string]int64{}}
	names := make([]string, 0, blacklistESIBatchSize)
	for i := range blacklistESIBatchSize {
		names = append(names, fmt.Sprintf("Unknown %03d", i))
	}
	client := newFakeBlacklistESIClient(t, fake)
	got := resolveBlacklistTestNames(client, names)
	if sizes := fake.batchSizes(); !reflect.DeepEqual(sizes, []int{50}) {
		t.Fatalf("batch sizes = %v, want [50]", sizes)
	}
	if len(got.resolved) != 0 || len(got.notFound) != blacklistESIBatchSize || len(got.deferred) != 0 {
		t.Fatalf("resolved=%d notFound=%d deferred=%d", len(got.resolved), len(got.notFound), len(got.deferred))
	}
}

func TestResolveBlacklistCharacterNamesRetriesTransientError(t *testing.T) {
	fake := &fakeBlacklistESI{characters: map[string]int64{"known pilot": 90000001}, failFirst: 2}
	client := newFakeBlacklistESIClient(t, fake)
	got := resolveBlacklistTestNames(client, []string{"Known Pilot", "Ghost Pilot"})
	if sizes := fake.batchSizes(); len(sizes) != 3 {
		t.Fatalf("requests = %v, want 3 attempts", sizes)
	}
	if got.resolved["known pilot"].ID != 90000001 || !reflect.DeepEqual(got.notFound, []string{"Ghost Pilot"}) || len(got.deferred) != 0 {
		t.Fatalf("after retry: %+v", got)
	}
}

func TestResolveBlacklistCharacterNamesDefersFailedBatch(t *testing.T) {
	fake := &fakeBlacklistESI{characters: map[string]int64{}, failNames: map[string]bool{"pilot 000": true}}
	names := make([]string, 0, blacklistESIBatchSize+1)
	for i := range blacklistESIBatchSize {
		names = append(names, fmt.Sprintf("Pilot %03d", i))
	}
	fake.characters["late pilot"] = 91000000
	names = append(names, "Late Pilot")
	client := newFakeBlacklistESIClient(t, fake)

	got := resolveBlacklistTestNames(client, names)
	if sizes := fake.batchSizes(); !reflect.DeepEqual(sizes, []int{50, 50, 50, 1}) {
		t.Fatalf("batch sizes = %v, want three attempts at the failing batch, then the next batch", sizes)
	}
	if !reflect.DeepEqual(got.deferred, names[:blacklistESIBatchSize]) {
		t.Fatalf("deferred = %v, want the failing batch", got.deferred)
	}
	if len(got.notFound) != 0 {
		t.Fatalf("failed names reported as not found: %v", got.notFound)
	}
	if got.resolved["late pilot"].ID != 91000000 {
		t.Fatalf("later batch not resolved: %+v", got.resolved)
	}
}

func TestResolveBlacklistCharacterNamesDoesNotRetryClientError(t *testing.T) {
	fake := &fakeBlacklistESI{failStatus: http.StatusBadRequest}
	client := newFakeBlacklistESIClient(t, fake)
	got := resolveBlacklistTestNames(client, []string{"Ghost Pilot"})
	if sizes := fake.batchSizes(); len(sizes) != 1 {
		t.Fatalf("requests = %v, want a single attempt", sizes)
	}
	if !reflect.DeepEqual(got.deferred, []string{"Ghost Pilot"}) || len(got.notFound) != 0 {
		t.Fatalf("client error: %+v", got)
	}
}

func TestResolveBlacklistCharacterNamesRetriesRateLimitAndNetworkErrors(t *testing.T) {
	for _, tc := range []struct {
		name         string
		fake         *fakeBlacklistESI
		wantRequests int
		wantDeferred int
	}{
		{"rate limit", &fakeBlacklistESI{failStatus: http.StatusTooManyRequests}, 3, 1},
		{"connection closed", &fakeBlacklistESI{characters: map[string]int64{"known pilot": 90000001}, disconnectFirst: 2}, 3, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveBlacklistTestNames(newFakeBlacklistESIClient(t, tc.fake), []string{"Known Pilot"})
			if len(tc.fake.batchSizes()) != tc.wantRequests || len(got.deferred) != tc.wantDeferred || len(got.notFound) != 0 {
				t.Fatalf("requests=%v result=%+v", tc.fake.batchSizes(), got)
			}
			if tc.wantDeferred == 0 && got.resolved["known pilot"].ID != 90000001 {
				t.Fatalf("name not resolved after retry: %+v", got)
			}
		})
	}
}

func TestBlacklistESIRequestTimeoutIsRetryable(t *testing.T) {
	if !retryableBlacklistESIError(fmt.Errorf("HTTP client timeout: %w", context.DeadlineExceeded)) {
		t.Fatal("per-request timeout should be retried while the lookup budget remains")
	}
	if retryableBlacklistESIError(fmt.Errorf("request canceled: %w", context.Canceled)) {
		t.Fatal("canceled request should not be retried")
	}
}

func TestResolveBlacklistCharacterNamesBudgetIncludesRetryBackoff(t *testing.T) {
	fake := &fakeBlacklistESI{failStatus: http.StatusGatewayTimeout}
	client := newFakeBlacklistESIClient(t, fake)
	names := []string{"Known Pilot"}
	got := resolveBlacklistCharacterNames(context.Background(), client, names, []time.Duration{time.Hour}, 100*time.Millisecond)
	if len(fake.batchSizes()) != 1 || !reflect.DeepEqual(got.deferred, names) || len(got.notFound) != 0 {
		t.Fatalf("budget expired during backoff: batches=%v result=%+v", fake.batchSizes(), got)
	}
}

func TestResolveBlacklistCharacterNamesCanceledContextSendsNoRequests(t *testing.T) {
	fake := &fakeBlacklistESI{}
	client := newFakeBlacklistESIClient(t, fake)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	names := []string{"Known Pilot"}
	got := resolveBlacklistCharacterNames(ctx, client, names, testBlacklistRetryDelays, time.Minute)
	if !errors.Is(ctx.Err(), context.Canceled) || len(fake.batchSizes()) != 0 || !reflect.DeepEqual(got.deferred, names) || len(got.notFound) != 0 {
		t.Fatalf("canceled lookup: batches=%v result=%+v", fake.batchSizes(), got)
	}
}

func TestResolveBlacklistCharacterNamesDefersAfterBudget(t *testing.T) {
	fake := &fakeBlacklistESI{characters: map[string]int64{}, delay: 80 * time.Millisecond}
	names := make([]string, 0, 2*blacklistESIBatchSize+1)
	for i := range 2*blacklistESIBatchSize + 1 {
		names = append(names, fmt.Sprintf("Pilot %03d", i))
	}
	client := newFakeBlacklistESIClient(t, fake)

	got := resolveBlacklistCharacterNames(context.Background(), client, names, nil, 120*time.Millisecond)
	if sizes := fake.batchSizes(); !reflect.DeepEqual(sizes, []int{50, 50}) {
		t.Fatalf("batch sizes = %v, want the third batch never sent", sizes)
	}
	if len(got.notFound) != blacklistESIBatchSize {
		t.Fatalf("notFound = %d, want the first batch", len(got.notFound))
	}
	if !reflect.DeepEqual(got.deferred, names[blacklistESIBatchSize:]) {
		t.Fatalf("deferred %d names, want the %d names after the first batch", len(got.deferred), len(names)-blacklistESIBatchSize)
	}
}

func TestResolveBlacklistCharacterNamesNoNames(t *testing.T) {
	fake := &fakeBlacklistESI{}
	client := newFakeBlacklistESIClient(t, fake)
	got := resolveBlacklistTestNames(client, nil)
	if len(got.resolved) != 0 || len(got.notFound) != 0 || len(got.deferred) != 0 || len(fake.batchSizes()) != 0 {
		t.Fatalf("empty input: %+v batches=%v", got, fake.batchSizes())
	}
}
