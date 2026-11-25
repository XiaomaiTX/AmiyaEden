package esi

import (
	"amiya-eden/global"
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/zap"
)

func TestNewAuthorizedRequestSetsExpectedHeaders(t *testing.T) {
	client := NewClientWithConfig("https://esi.evetech.net", "/latest")

	req, err := client.newAuthorizedRequest(http.MethodPost, "/characters/1/", "token-123", bytes.NewBufferString(`{}`), "application/json")
	if err != nil {
		t.Fatalf("expected request to be built, got error: %v", err)
	}

	if req.URL.String() != "https://esi.evetech.net/latest/characters/1/" {
		t.Fatalf("unexpected request URL: %s", req.URL.String())
	}
	if got := req.Header.Get("Authorization"); got != "Bearer token-123" {
		t.Fatalf("unexpected authorization header: %s", got)
	}
	if got := req.Header.Get("Accept"); got != "application/json" {
		t.Fatalf("unexpected accept header: %s", got)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content-type header: %s", got)
	}
}

type requestTestTransport func(*http.Request) (*http.Response, error)

func (f requestTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGetWithMetaStopsWaitingAfterLastRateLimitResponse(t *testing.T) {
	logger := global.Logger
	global.Logger = zap.NewNop()
	t.Cleanup(func() { global.Logger = logger })

	synctest.Test(t, func(t *testing.T) {
		requests := 0
		client := NewClientWithConfig("https://esi.example", "")
		client.httpClient = &http.Client{Transport: requestTestTransport(func(req *http.Request) (*http.Response, error) {
			requests++
			return &http.Response{
				StatusCode: 420,
				Header:     http.Header{"X-Esi-Request-Id": {"rate-limit-response"}},
				Body:       io.NopCloser(strings.NewReader(`{"error":"rate limited"}`)),
				Request:    req,
			}, nil
		})}

		start := time.Now()
		meta, err := client.GetWithMeta(context.Background(), "/test", "", nil)
		if requests != 4 || time.Since(start) != 14*time.Second {
			t.Fatalf("requests = %d, elapsed = %s; want four requests and three backoffs (14s)", requests, time.Since(start))
		}
		if !IsHTTPStatus(err, 420) || meta == nil || meta.RequestID != "rate-limit-response" {
			t.Fatalf("last response = %+v, error = %v; want typed 420 with response metadata", meta, err)
		}
	})
}

func TestReadResponseBodyRejectsOversizedPayload(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(bytes.NewBufferString("abcdef")),
	}

	body, err := readResponseBody(resp, 5)
	if err == nil {
		t.Fatal("expected oversized payload error")
	}
	if body != nil {
		t.Fatalf("expected no body on oversize error, got %q", string(body))
	}
}
