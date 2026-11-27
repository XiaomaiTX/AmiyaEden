package qqapi

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func mockReply(t *testing.T, status int, header http.Header, body string, check func(*http.Request)) *Client {
	t.Helper()
	c := New("test-app", "test-secret")
	var calls int
	c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Body != nil {
			defer req.Body.Close() //nolint:errcheck // Reader has a no-op close.
		}
		if req.URL.Scheme != "https" || req.URL.Host != "api.bot.qq.com" || req.GetBody != nil {
			t.Fatalf("unexpected URL/replay: %s replay=%v", req.URL, req.GetBody != nil)
		}
		if check != nil {
			check(req)
		}
		return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	t.Cleanup(func() {
		if calls != 1 {
			t.Errorf("got %d outbound requests, want 1", calls)
		}
	})
	return c
}

func checkCall(t *testing.T, r *http.Request, method, uri, authorization, jsonBody string) {
	t.Helper()
	if r.Method != method || r.URL.RequestURI() != uri || r.Header.Get("Authorization") != authorization {
		t.Fatalf("unexpected request: %s %s auth=%q", r.Method, r.URL.RequestURI(), r.Header.Get("Authorization"))
	}
	if jsonBody == "" {
		if r.Body != nil || r.Header.Get("Content-Type") != "" {
			t.Fatal("GET had a content body")
		}
		return
	}
	if r.Body == nil || r.Header.Get("Content-Type") != "application/json" {
		t.Fatal("expected JSON request")
	}
	body, err := io.ReadAll(r.Body)
	if err != nil || string(body) != jsonBody {
		t.Fatalf("got body %q (%v), want %q", body, err, jsonBody)
	}
}

func TestRefreshOfficialContract(t *testing.T) {
	for _, ttl := range []string{`7200`, `"7200"`} {
		t.Run(ttl, func(t *testing.T) {
			c := mockReply(t, http.StatusOK, nil, `{"access_token":"opaque-token","expires_in":`+ttl+`}`, func(req *http.Request) {
				checkCall(t, req, http.MethodPost, "/app/getAppAccessToken", "", `{"appId":"test-app","clientSecret":"test-secret"}`)
			})
			start := time.Now()
			token, err := c.Refresh(context.Background())
			if err != nil || token.AccessToken != "opaque-token" || token.ExpiresAt.Before(start.Add(2*time.Hour)) || token.ExpiresAt.After(time.Now().Add(2*time.Hour)) {
				t.Fatalf("unexpected refresh: %+v, %v", token, err)
			}
		})
	}
}

func TestRefreshFailureUnderHTTP200(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		kind ErrorKind
		code int
	}{
		{"throttle", `{"code":100001,"access_token":"not-valid"}`, Throttle, 100001},
		{"bad credentials", `{"code":100016,"message":"NEVER EXPOSE","trace_id":"qq-123"}`, AuthBlocked, 100016},
		{"bad app", `{"code":100007}`, AuthBlocked, 100007},
		{"missing app", `{"code":10004}`, AuthBlocked, 10004},
		{"unknown code", `{"code":100111,"access_token":"not-valid","expires_in":7200}`, TemporaryNotApplied, 100111},
		{"missing ttl", `{"access_token":"not-valid"}`, TemporaryNotApplied, 0},
		{"overflow ttl", `{"access_token":"not-valid","expires_in":"9999999999"}`, TemporaryNotApplied, 0},
		{"bad ttl", `{"access_token":"not-valid","expires_in":0}`, TemporaryNotApplied, 0},
		{"malformed", `{"access_token":"not-valid","expires_in":7200`, TemporaryNotApplied, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := mockReply(t, http.StatusOK, nil, tc.body, nil)
			token, err := c.Refresh(context.Background())
			if token.AccessToken != "" {
				t.Fatal("failure returned token")
			}
			op := checkError(t, err, tc.kind, http.StatusOK, tc.code)
			if tc.name == "bad credentials" && (op.TraceID != "qq-123" || strings.Contains(op.Error(), "NEVER EXPOSE")) {
				t.Fatal("error did not preserve only safe metadata")
			}
		})
	}
}

func TestExecuteOfficialContracts(t *testing.T) {
	ctx := context.Background()
	const base = "/v2/groups/Group32"
	const auth = "QQBot token"
	group := Request{GroupOpenID: "Group32"}
	t.Run("send text", func(t *testing.T) {
		c := mockReply(t, http.StatusOK, http.Header{"X-Tps-Trace-Id": {"header-trace"}}, `{"id":"ROBOT1.0_id","trace_id":"body-trace"}`, func(r *http.Request) {
			checkCall(t, r, http.MethodPost, base+"/messages", auth, `{"msg_type":0,"content":"你好"}`)
		})
		req := group
		req.Kind, req.Text = SendText, "你好"
		res, err := c.Execute(ctx, "token", req)
		if err != nil || res.MessageID != "ROBOT1.0_id" || res.TraceID != "header-trace" || res.AwaitingReview {
			t.Fatalf("unexpected send: %+v, %v", res, err)
		}
	})
	t.Run("group state", func(t *testing.T) {
		c := mockReply(t, http.StatusOK, nil, `{"member_openid":"bot","allow_proactive_msg":false,"member_role":"admin","trace_id":"state-trace"}`, func(r *http.Request) {
			checkCall(t, r, http.MethodGet, base+"/bot_state", auth, "")
		})
		req := group
		req.Kind = GroupState
		res, err := c.Execute(ctx, "token", req)
		if err != nil || res.AllowProactive || res.MemberRole != "admin" || res.TraceID != "state-trace" {
			t.Fatalf("unexpected group state: %+v, %v", res, err)
		}
	})
	t.Run("join list", func(t *testing.T) {
		c := mockReply(t, http.StatusOK, nil, `{"list":[{"join_request_id":"application-A","member_openid":"User32","username":"昵称","apply_at":"2026-08-05T14:19:09+08:00","verify_info":{"review_qa_list":[{"answer":"NEVER EXPOSE"}]}}],"next_cursor":"next+1"}`, func(r *http.Request) {
			checkCall(t, r, http.MethodGet, base+"/join_request_list?cursor=%2B%2F%3D%26%3Fx%3Dy", auth, "")
		})
		req := group
		req.Kind, req.Cursor = JoinList, "+/=&?x=y"
		res, err := c.Execute(ctx, "token", req)
		want := JoinRequest{ID: "application-A", MemberOpenID: "User32", Username: "昵称", AppliedAt: time.Date(2026, 8, 5, 6, 19, 9, 0, time.UTC)}
		if err != nil || res.NextCursor != "next+1" || len(res.JoinRequests) != 1 || res.JoinRequests[0] != want {
			t.Fatalf("unexpected join list: %+v, %v", res, err)
		}
	})
	for _, tc := range []struct {
		decision string
		wire     string
		status   int
		body     string
	}{{"reject", "decline", http.StatusOK, `{}`}, {"approve", "approve", http.StatusNoContent, ""}} {
		t.Run(tc.decision, func(t *testing.T) {
			c := mockReply(t, tc.status, http.Header{"X-Tps-Trace-Id": {"decision-trace"}}, tc.body, func(r *http.Request) {
				checkCall(t, r, http.MethodPost, base+"/approval_join_request/User32", auth, `{"op":"`+tc.wire+`","join_request_id":"application-A"}`)
			})
			req := group
			req.Kind, req.MemberOpenID, req.JoinRequestID, req.Decision = JoinDecision, "User32", "application-A", tc.decision
			res, err := c.Execute(ctx, "token", req)
			if err != nil || res.TraceID != "decision-trace" || res.AwaitingReview {
				t.Fatalf("unexpected join approval: %+v, %v", res, err)
			}
		})
	}
}

func TestWriteClassification(t *testing.T) {
	const bodyWithID = `{"id":"not-a-confirmed-id"}`
	for _, tc := range []struct {
		name   string
		status int
		body   string
		kind   ErrorKind
		code   int
	}{
		{"http throttle with non-json body", 429, `<html>NEVER EXPOSE</html>`, Throttle, 0},
		{"QQ proactive throttle", 400, `{"err_code":40034100,"message":"NEVER EXPOSE","trace_id":"qq-trace"}`, Throttle, 40034100},
		{"invalid token", 401, `{"err_code":11243}`, InvalidToken, 11243},
		{"blocked", 403, `{"err_code":11265}`, AuthBlocked, 11265},
		{"group application not authorized", 403, `{"err_code":11253}`, Forbidden, 11253},
		{"not a member", 400, `{"err_code":40034101}`, GroupNotFound, 40034101},
		{"404 says API missing, not group gone", 404, ``, Forbidden, 0},
		{"proactive rejected", 400, `{"err_code":40034105}`, PushRejected, 40034105},
		{"invalid body", 400, `{"err_code":40054007}`, InvalidContent, 40054007},
		{"pre-application privilege check failure", 500, `{"err_code":11252}`, TemporaryNotApplied, 11252},
		{"server rejected generically", 500, `{"err_code":50055001}`, Ambiguous, 50055001},
		{"gateway timeout", 504, ``, Ambiguous, 0},
		{"201 is not a delivery receipt", 201, bodyWithID, Ambiguous, 0},
		{"202 unknown code", 202, `{"err_code":304099}`, Ambiguous, 304099},
		{"unknown 200 error", 200, `{"err_code":777,"id":"fake"}`, Ambiguous, 777},
		{"200 without confirmation", 200, `{}`, Ambiguous, 0},
		{"malformed JSON", 200, bodyWithID + ` trailing`, Ambiguous, 0},
		{"malformed code", 200, `{"id":"fake","err_code":"40034100"}`, Ambiguous, 0},
		{"conflicting code fields", 200, `{"id":"fake","err_code":40034100,"code":11243}`, Ambiguous, 0},
		{"body cap", 200, bodyWithID + strings.Repeat(" ", maxResponseBodyBytes), Ambiguous, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := mockReply(t, tc.status, http.Header{"Retry-After": {"42"}}, tc.body, nil)
			r, err := c.Execute(context.Background(), "token", Request{Kind: SendText, GroupOpenID: "group", Text: "text"})
			if r.MessageID != "" || r.AwaitingReview {
				t.Fatal("unexpected successful send")
			}
			op := checkError(t, err, tc.kind, tc.status, tc.code)
			if op.RetryAfter != 42*time.Second || strings.Contains(err.Error(), "NEVER EXPOSE") {
				t.Fatalf("unexpected error metadata: %+v", op)
			}
			if tc.code == 40034100 && op.TraceID != "qq-trace" {
				t.Fatalf("lost body trace ID: %+v", op)
			}
		})
	}
}

func TestRecognizedAuditIsNotDelivery(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{
		{201, `{"code":304023,"trace_id":"audit-trace","data":{"message_audit":{"audit_id":"audit-id"}}}`},
		{202, `{"err_code":304024,"trace_id":"audit-trace"}`},
	} {
		c := mockReply(t, tc.status, nil, tc.body, nil)
		res, err := c.Execute(context.Background(), "token", Request{Kind: SendText, GroupOpenID: "group", Text: "text"})
		if err != nil || !res.AwaitingReview || res.MessageID != "" || res.TraceID != "audit-trace" {
			t.Fatalf("unexpected audit result: %+v, %v", res, err)
		}
	}
}

func TestInvalidApprovalAcknowledgementIsAmbiguous(t *testing.T) {
	for _, body := range []string{"", "not JSON", `null`, `[]`, `{"message":"not a receipt"}`, `{} {}`} {
		c := mockReply(t, http.StatusOK, nil, body, nil)
		_, err := c.Execute(context.Background(), "token", Request{Kind: JoinDecision, GroupOpenID: "group", MemberOpenID: "member", JoinRequestID: "id", Decision: "approve"})
		_ = checkError(t, err, Ambiguous, http.StatusOK, 0)
	}
}

func TestReadMissingFieldsAreNotFalseOrEmptySuccesses(t *testing.T) {
	for _, tc := range []struct{ kind, body string }{
		{GroupState, `{"member_role":"admin"}`},
		{GroupState, `{"allow_proactive_msg":false}`},
		{JoinList, `{"next_cursor":"no-list"}`},
		{JoinList, `{"list":[{"join_request_id":"valid","member_openid":"member","apply_at":"not-a-time"}]}`},
	} {
		c := mockReply(t, http.StatusOK, nil, tc.body, nil)
		_, err := c.Execute(context.Background(), "token", Request{Kind: tc.kind, GroupOpenID: "group"})
		_ = checkError(t, err, TemporaryNotApplied, http.StatusOK, 0)
	}
}

func TestInputsRejectedBeforeTransport(t *testing.T) {
	c := New("test-app", "test-secret")
	c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("made network request after validation failure")
		return nil, errors.New("unexpected request")
	})
	for _, r := range []Request{
		{Kind: "arbitrary", GroupOpenID: "group"},
		{Kind: SendText, GroupOpenID: "../etc", Text: "text"},
		{Kind: SendText, GroupOpenID: "group", Text: " "},
		{Kind: JoinDecision, GroupOpenID: "group", MemberOpenID: "member", Decision: "approve"},
		{Kind: JoinDecision, GroupOpenID: "group", MemberOpenID: "member", JoinRequestID: "id", Decision: "ban"},
	} {
		_, err := c.Execute(context.Background(), "token", r)
		_ = checkError(t, err, InvalidContent, 0, 0)
	}
	req := Request{Kind: SendText, GroupOpenID: "group", Text: "text"}
	_, err := c.Execute(context.Background(), "bad\r\nToken", req)
	_ = checkError(t, err, InvalidToken, 0, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = c.Execute(ctx, "token", req)
	_ = checkError(t, err, TemporaryNotApplied, 0, 0)
}

func TestPinnedTransportNoRedirectOrNetworkReplay(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if r.ProtoMajor != 1 || !r.Close || r.Host != "api.bot.qq.com" || r.URL.Path != "/v2/groups/group/messages" {
			t.Errorf("unexpected request protocol/endpoint: %s close=%v host=%s path=%s", r.Proto, r.Close, r.Host, r.URL.Path)
		}
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Errorf("reading local test body: %v", err)
		}
		switch n {
		case 1:
			w.Header().Set("Location", "https://api.bot.qq.com/donotfollow")
			w.WriteHeader(http.StatusTemporaryRedirect)
		case 2:
			conn, _, err := http.NewResponseController(w).Hijack()
			if err != nil {
				t.Errorf("hijacking local connection: %v", err)
				return
			}
			if err := conn.Close(); err != nil {
				t.Errorf("closing local connection: %v", err)
			}
		default:
			t.Error("redirect or network failure replayed a write")
		}
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	c := New("app", "secret")
	transport := c.http.Transport.(*http.Transport)
	// Test only: route the fixed origin to the local TLS mock without changing
	// protocol, retry, keep-alive, or redirect options from production.
	transport.TLSClientConfig = server.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	transport.TLSClientConfig.ServerName = "example.com"
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "api.bot.qq.com:443" {
			return nil, errors.New("unexpected fixated origin")
		}
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	req := Request{Kind: SendText, GroupOpenID: "group", Text: "text"}
	_, err := c.Execute(context.Background(), "token", req)
	_ = checkError(t, err, Ambiguous, http.StatusTemporaryRedirect, 0)
	_, err = c.Execute(context.Background(), "token", req)
	_ = checkError(t, err, Ambiguous, 0, 0)
	if calls.Load() != 2 {
		t.Fatalf("got %d total calls, want 2", calls.Load())
	}
}

func TestRetryAfterHTTPDateAndOverflow(t *testing.T) {
	now := time.Date(2026, 11, 26, 15, 0, 0, 0, time.UTC)
	if got := retryAfter(now.Add(time.Minute).Format(http.TimeFormat), now); got != time.Minute {
		t.Fatalf("date backoff = %s", got)
	}
	for _, raw := range []string{"-2", "9999999999", now.Add(-time.Second).Format(http.TimeFormat)} {
		if got := retryAfter(raw, now); got != 0 {
			t.Fatalf("invalid backoff %q = %s", raw, got)
		}
	}
}

func checkError(t *testing.T, err error, kind ErrorKind, status, code int) *OperationError {
	t.Helper()
	var op *OperationError
	if !errors.As(err, &op) || op.Kind != kind || op.HTTPStatus != status || op.Code != code {
		t.Fatalf("got %v, want %s HTTP %d code %d", err, kind, status, code)
	}
	return op
}
