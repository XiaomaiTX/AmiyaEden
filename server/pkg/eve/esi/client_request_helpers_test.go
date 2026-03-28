package esi

import (
	"bytes"
	"io"
	"net/http"
	"testing"
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
