// Package qqapi implements the small subset of the official QQ Bot OpenAPI
// used by the service. It does not cache tokens or schedule/retry operations.
package qqapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	SendText     = "send_text"
	GroupState   = "group_state"
	JoinList     = "join_list"
	JoinDecision = "join_decision"

	apiOrigin            = "https://api.bot.qq.com"
	maxResponseBodyBytes = 1 << 20
	maxOpenIDBytes       = 128
	maxJoinRequestBytes  = 384
	maxMessageIDBytes    = 256
	maxCursorBytes       = 1024
	maxTokenBytes        = 8192
)

type Request struct {
	Kind          string
	GroupOpenID   string
	Text          string
	Cursor        string
	MemberOpenID  string
	JoinRequestID string
	Decision      string
}

type Result struct {
	MessageID      string
	NextCursor     string
	JoinRequests   []JoinRequest
	AllowProactive bool
	MemberRole     string
	TraceID        string
	AwaitingReview bool
}

type JoinRequest struct {
	ID           string
	MemberOpenID string
	Username     string
	AppliedAt    time.Time
	AutoApproved bool
}

type Token struct {
	AccessToken string
	ExpiresAt   time.Time
}

type Client struct {
	appID  string
	secret string
	http   *http.Client
}

// New creates a client pinned to the official QQ HTTPS origin with no proxy,
// redirects, HTTP/2, or connection reuse (net/http can retry reused connections).
func New(appID, secret string) *Client {
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	return &Client{
		appID:  appID,
		secret: secret,
		http: &http.Client{
			Transport: &http.Transport{
				DialContext:            (&net.Dialer{Timeout: 4 * time.Second}).DialContext,
				DisableKeepAlives:      true,
				DisableCompression:     true,
				TLSHandshakeTimeout:    4 * time.Second,
				MaxResponseHeaderBytes: 64 << 10,
				Protocols:              &protocols,
			},
			CheckRedirect: rejectRedirect,
			Timeout:       10 * time.Second,
		},
	}
}

func rejectRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

type refreshBody struct {
	AppID        string `json:"appId"`
	ClientSecret string `json:"clientSecret"`
}

type proactiveTextBody struct {
	MsgType int    `json:"msg_type"`
	Content string `json:"content"`
}

type joinDecisionBody struct {
	Op            string `json:"op"`
	JoinRequestID string `json:"join_request_id"`
}

// Refresh performs one credential exchange. The caller owns credential caching
// and refresh scheduling. QQ's documentation specifies numeric TTL seconds, but
// its sample response uses a decimal string, so both are supported.
func (c *Client) Refresh(ctx context.Context) (Token, error) {
	if c.appID == "" || c.secret == "" {
		return Token{}, &OperationError{Kind: AuthBlocked}
	}
	started := time.Now()
	r, err := postJSON(ctx, c, "/app/getAppAccessToken", "", refreshOp, refreshBody{AppID: c.appID, ClientSecret: c.secret})
	if err != nil {
		return Token{}, err
	}
	var out struct {
		AccessToken string          `json:"access_token"`
		ExpiresIn   json.RawMessage `json:"expires_in"`
	}
	if r.status != http.StatusOK || json.Unmarshal(r.body, &out) != nil || !visibleASCII(out.AccessToken, maxTokenBytes) {
		return Token{}, r.failure(TemporaryNotApplied)
	}
	seconds, ok := decimalInt(bytes.Trim(out.ExpiresIn, `"`))
	if !ok || seconds <= 0 || seconds > maxDurationSeconds {
		return Token{}, r.failure(TemporaryNotApplied)
	}
	return Token{AccessToken: out.AccessToken, ExpiresAt: started.Add(time.Duration(seconds) * time.Second)}, nil
}

// Execute makes at most one OpenAPI request for the given typed operation.
func (c *Client) Execute(ctx context.Context, token string, req Request) (Result, error) {
	if !visibleASCII(token, maxTokenBytes) {
		return Result{}, &OperationError{Kind: InvalidToken}
	}
	if !urlSafeID(req.GroupOpenID, maxOpenIDBytes) {
		return Result{}, &OperationError{Kind: InvalidContent}
	}
	base := "/v2/groups/" + req.GroupOpenID
	switch req.Kind {
	case SendText:
		return c.sendText(ctx, base, token, req.Text)
	case GroupState:
		return c.groupState(ctx, base, token)
	case JoinList:
		return c.joinList(ctx, base, token, req.Cursor)
	case JoinDecision:
		return c.joinDecision(ctx, base, token, req)
	default:
		return Result{}, &OperationError{Kind: InvalidContent}
	}
}

func (c *Client) sendText(ctx context.Context, base, token, text string) (Result, error) {
	if len(text) > maxResponseBodyBytes || !utf8.ValidString(text) || strings.TrimSpace(text) == "" {
		return Result{}, &OperationError{Kind: InvalidContent}
	}
	r, err := postJSON(ctx, c, base+"/messages", token, sendOp, proactiveTextBody{MsgType: 0, Content: text})
	if err != nil {
		return Result{}, err
	}
	if r.status == http.StatusCreated || r.status == http.StatusAccepted {
		// QQ's official 304023/304024 codes are acknowledgements of review,
		// not delivery. The operation never returns a message ID for those.
		return Result{TraceID: r.traceID, AwaitingReview: true}, nil
	}
	var out struct {
		ID string `json:"id"`
	}
	if r.status != http.StatusOK || json.Unmarshal(r.body, &out) != nil || !visibleASCII(out.ID, maxMessageIDBytes) {
		return Result{}, r.failure(Ambiguous)
	}
	return Result{MessageID: out.ID, TraceID: r.traceID}, nil
}

func (c *Client) groupState(ctx context.Context, base, token string) (Result, error) {
	r, err := c.call(ctx, http.MethodGet, base+"/bot_state", token, nil, readOp)
	if err != nil {
		return Result{}, err
	}
	var out struct {
		AllowProactive *bool  `json:"allow_proactive_msg"`
		MemberRole     string `json:"member_role"`
	}
	if r.status != http.StatusOK || json.Unmarshal(r.body, &out) != nil || out.AllowProactive == nil || !urlSafeID(out.MemberRole, 32) {
		return Result{}, r.failure(TemporaryNotApplied)
	}
	return Result{AllowProactive: *out.AllowProactive, MemberRole: out.MemberRole, TraceID: r.traceID}, nil
}

func (c *Client) joinList(ctx context.Context, base, token, cursor string) (Result, error) {
	path := base + "/join_request_list"
	if cursor != "" {
		if !visibleASCII(cursor, maxCursorBytes) {
			return Result{}, &OperationError{Kind: InvalidContent}
		}
		path += "?" + (url.Values{"cursor": {cursor}}).Encode()
	}
	r, err := c.call(ctx, http.MethodGet, path, token, nil, readOp)
	if err != nil {
		return Result{}, err
	}
	var out struct {
		List       []wireJoinRequest `json:"list"`
		NextCursor string            `json:"next_cursor"`
	}
	if r.status != http.StatusOK || json.Unmarshal(r.body, &out) != nil || out.List == nil ||
		(out.NextCursor != "" && !visibleASCII(out.NextCursor, maxCursorBytes)) {
		return Result{}, r.failure(TemporaryNotApplied)
	}
	items := make([]JoinRequest, 0, len(out.List))
	for _, row := range out.List {
		item, ok := row.joinRequest()
		if !ok {
			return Result{}, r.failure(TemporaryNotApplied)
		}
		items = append(items, item)
	}
	return Result{NextCursor: out.NextCursor, JoinRequests: items, TraceID: r.traceID}, nil
}

func (c *Client) joinDecision(ctx context.Context, base, token string, req Request) (Result, error) {
	// The official ID is optional, but require it here to avoid applying a
	// stale decision to a newer request by the same group member.
	if !urlSafeID(req.MemberOpenID, maxOpenIDBytes) || !visibleASCII(req.JoinRequestID, maxJoinRequestBytes) {
		return Result{}, &OperationError{Kind: InvalidContent}
	}
	op := req.Decision
	switch op {
	case "approve", "decline":
	case "reject":
		op = "decline"
	default:
		return Result{}, &OperationError{Kind: InvalidContent}
	}
	r, err := postJSON(ctx, c, base+"/approval_join_request/"+req.MemberOpenID, token, decisionOp, joinDecisionBody{Op: op, JoinRequestID: req.JoinRequestID})
	if err != nil {
		return Result{}, err
	}
	if !r.decisionAck {
		return Result{}, r.failure(Ambiguous)
	}
	return Result{TraceID: r.traceID}, nil
}

// Keep only the fields used by the service; verification messages and Q&A
// aren't decoded into application types.
type wireJoinRequest struct {
	ID           string    `json:"join_request_id"`
	MemberOpenID string    `json:"member_openid"`
	Username     string    `json:"username"`
	AppliedAt    time.Time `json:"apply_at"`
	AutoApproved *struct{} `json:"auto_approved"`
}

func (w wireJoinRequest) joinRequest() (JoinRequest, bool) {
	if !visibleASCII(w.ID, maxJoinRequestBytes) || !urlSafeID(w.MemberOpenID, maxOpenIDBytes) || w.AppliedAt.IsZero() {
		return JoinRequest{}, false
	}
	return JoinRequest{ID: w.ID, MemberOpenID: w.MemberOpenID, Username: w.Username, AppliedAt: w.AppliedAt.UTC(), AutoApproved: w.AutoApproved != nil}, true
}

func postJSON[B refreshBody | proactiveTextBody | joinDecisionBody](ctx context.Context, c *Client, path, token string, op operation, body B) (apiResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return apiResponse{}, &OperationError{Kind: InvalidContent}
	}
	return c.call(ctx, http.MethodPost, path, token, b, op)
}

func (c *Client) call(ctx context.Context, method, path, token string, body []byte, op operation) (apiResponse, error) {
	if ctx.Err() != nil {
		return apiResponse{}, &OperationError{Kind: TemporaryNotApplied}
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, apiOrigin+path, reader)
	if err != nil {
		return apiResponse{}, &OperationError{Kind: InvalidContent}
	}
	req.GetBody = nil // Don't provide a mechanism to replay the body.
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "QQBot "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return apiResponse{}, &OperationError{Kind: op.unconfirmed()}
	}
	defer resp.Body.Close() //nolint:errcheck // No useful action on response close failure.
	r := apiResponse{
		status:     resp.StatusCode,
		traceID:    validTraceID(resp.Header.Get("X-Tps-Trace-ID")),
		retryAfter: retryAfter(resp.Header.Get("Retry-After"), time.Now()),
	}
	r.body, err = io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
	if err == nil && len(r.body) <= maxResponseBodyBytes {
		r.parse(op)
	}
	if kind := r.classify(op); kind != "" {
		return apiResponse{}, r.failure(kind)
	}
	return r, nil
}

func visibleASCII(value string, maxBytes int) bool {
	if len(value) == 0 || len(value) > maxBytes {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '!' || value[i] > '~' {
			return false
		}
	}
	return true
}

func urlSafeID(value string, maxBytes int) bool {
	if len(value) == 0 || len(value) > maxBytes {
		return false
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '-' || b == '_' {
			continue
		}
		return false
	}
	return true
}
