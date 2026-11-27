package qqapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	mockGatewayEndpoint = "wss://api.bot.qq.com/websocket/"
	mockGatewayToken    = "gateway-test-token"
	readyFrame          = `{"op":0,"s":10,"t":"READY","d":{"session_id":"gateway-session","user":{"username":"not retained"},"shard":[0,0]}}`
	helloFrame          = `{"op":10,"d":{"heartbeat_interval":45000}}`
)

func TestDiscoverGateway(t *testing.T) {
	c := mockReply(t, http.StatusOK, nil, `{"url":"`+mockGatewayEndpoint+`"}`, func(req *http.Request) {
		checkCall(t, req, http.MethodGet, "/gateway", "QQBot "+mockGatewayToken, "")
	})
	got, err := c.DiscoverGateway(t.Context(), mockGatewayToken)
	if err != nil || got != mockGatewayEndpoint {
		t.Fatalf("DiscoverGateway = %q, %v", got, err)
	}
}

func TestDiscoverGatewayErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		status int
		kind   ErrorKind
		code   int
		badTok bool
	}{
		{"untrusted URL", `{"url":"wss://not.qq.example/private"}`, 200, TemporaryNotApplied, 0, false},
		{"unauthorized", `{"message":"private"}`, 401, InvalidToken, 0, true},
		{"wrong token", `{"code":11243,"message":"private"}`, 200, InvalidToken, 11243, true},
		{"expired token", `{"code":11244,"message":"private"}`, 200, InvalidToken, 11244, true},
		{"blocked bot", `{"code":11265,"message":"private"}`, 403, AuthBlocked, 11265, false},
		{"HTTP throttle", `{"code":11244,"message":"private"}`, 429, Throttle, 11244, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := mockReply(t, tc.status, nil, tc.body, nil)
			endpoint, err := c.DiscoverGateway(t.Context(), mockGatewayToken)
			if endpoint != "" || errors.Is(err, ErrGatewayInvalidToken) != tc.badTok {
				t.Fatalf("DiscoverGateway = %q, %v", endpoint, err)
			}
			_ = checkError(t, err, tc.kind, tc.status, tc.code)
			if strings.Contains(err.Error(), "private") {
				t.Fatalf("provider data in error: %v", err)
			}
		})
	}
}

func TestGatewayDestinationAndRedirect(t *testing.T) {
	for _, endpoint := range []string{
		"ws://api.bot.qq.com/websocket/", "https://api.bot.qq.com/websocket/", "wss://api.bot.qq.com.evil.test/",
		"wss://api.bot.qq.com@localhost/", "wss://user@api.bot.qq.com/", "wss://127.0.0.1/",
		"wss://api.bot.qq.com:8443/", "wss://api.bot.qq.com./", "wss://api.bot.qq.com/#secret", "wss:api.bot.qq.com/",
	} {
		c := New("app", "secret")
		c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatalf("invalid gateway endpoint dialed: %q", endpoint)
			return nil, errors.New("unexpected dial")
		})
		var session GatewaySession
		if err := c.RunGateway(t.Context(), endpoint, mockGatewayToken, false, &session, noopReady, noopEvent); err != errGatewayEndpoint {
			t.Errorf("RunGateway(%q) = %v", endpoint, err)
		}
	}
	c := New("app", "secret")
	transport, ok := c.http.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil {
		t.Fatal("gateway transport must not use a proxy")
	}
	hits := 0
	c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		hits++
		if req.URL.Host != "api.bot.qq.com" || req.Header.Get("Authorization") != "" {
			t.Errorf("gateway upgrade sent credentials or followed a redirect: %v", req.URL)
		}
		return &http.Response{
			StatusCode: http.StatusFound, Header: http.Header{"Location": {"wss://redirect.invalid/private"}}, Body: http.NoBody,
		}, nil
	})
	var session GatewaySession
	if err := c.RunGateway(t.Context(), mockGatewayEndpoint, mockGatewayToken, false, &session, noopReady, noopEvent); err != errGatewayConnect || hits != 1 {
		t.Fatalf("redirect result: err=%v, hits=%d", err, hits)
	}
}

func TestGatewayIdentifyHeartbeatAndDispatch(t *testing.T) {
	c := mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
		if err := sendFrames(ctx, conn, `{"op":10,"d":{"heartbeat_interval":150}}`); err != nil {
			return err
		}
		if err := readIdentify(ctx, conn); err != nil {
			return err
		}
		if err := readHeartbeat(ctx, conn, nil); err != nil {
			return err
		}
		if err := sendFrames(ctx, conn, `{"op":11}`, readyFrame,
			`{"op":0,"s":11,"t":"GROUP_AT_MESSAGE_CREATE","id":{"ignored":"wrong ID type"},"d":{"content":"private chat"}}`,
			`{"op":0,"s":12,"t":"GROUP_ADD_ROBOT","$id":"event-v2","id":"event-v1","d":{"group_openid":"group","timestamp":1784570534}}`,
			`{"op":0,"s":13,"t":"GROUP_JOIN_REQUEST","d":{"group_openid":"group","join_request_id":"request","member_openid":"member","username":"昵称","apply_at":"2026-07-20T01:02:03+08:00","verify_info":{"verify_message":"private answer"},"auto_approved":{}}}`,
			`{"op":0,"s":14,"t":"UNKNOWN","d":["private content"]}`, `{"op":1}`); err != nil {
			return err
		}
		var reply gatewayPacket[json.RawMessage]
		if err := wsjson.Read(ctx, conn, &reply); err != nil {
			return err
		}
		if reply.Op != 11 || reply.Data != nil {
			return fmt.Errorf("got server-heartbeat reply %+v", reply)
		}
		latest := int64(14)
		if err := readHeartbeat(ctx, conn, &latest); err != nil {
			return err
		}
		return sendFrames(ctx, conn, `{"op":11}`, `{"op":7}`)
	})
	var session GatewaySession
	var order []string
	var events []Event
	ctx := testGatewayContext(t)
	err := c.RunGateway(ctx, mockGatewayEndpoint, mockGatewayToken, false, &session, func(context.Context) error {
		order = append(order, "ready")
		return nil
	}, func(_ context.Context, e Event) error {
		order = append(order, e.Type)
		events = append(events, e)
		return nil
	})
	if err != errGatewayReconnect || session != (GatewaySession{id: "gateway-session", seq: 14}) {
		t.Fatalf("RunGateway = %v, session = %+v", err, session)
	}
	if want := []string{"ready", GroupAddRobot, GroupJoinRequest}; !reflect.DeepEqual(order, want) {
		t.Fatalf("callbacks = %v, want %v", order, want)
	}
	if want := (Event{ID: "event-v2", Type: GroupAddRobot, GroupOpenID: "group", At: time.Unix(1784570534, 0).UTC()}); events[0] != want {
		t.Errorf("group event = %+v, want %+v", events[0], want)
	}
	wantTime := time.Date(2026, 7, 19, 17, 2, 3, 0, time.UTC)
	if e, want := events[1], (JoinRequest{ID: "request", MemberOpenID: "member", Username: "昵称", AppliedAt: wantTime, AutoApproved: true}); !strings.HasPrefix(e.ID, "sha256:") || e.GroupOpenID != "group" || e.At != wantTime || e.JoinRequest == nil || *e.JoinRequest != want {
		t.Errorf("join event = %+v, want request %+v", e, want)
	}
}

func TestGatewayResumeOnlyCommittedAndReadyAfterReplay(t *testing.T) {
	const replay = `{"op":0,"s":12,"t":"GROUP_MSG_REJECT","id":"replayed","d":{"group_openid":"group","timestamp":1784570534}}`
	const ignored = `{"op":0,"s":11,"t":"C2C_MESSAGE_CREATE","d":{"content":"not retained"}}`
	first := mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
		if err := openAndReadIdentify(ctx, conn); err != nil {
			return err
		}
		return sendFrames(ctx, conn, readyFrame, ignored, replay)
	})
	saveErr := errors.New("event persistence failed")
	var session GatewaySession
	ctx := testGatewayContext(t)
	err := first.RunGateway(ctx, mockGatewayEndpoint, mockGatewayToken, false, &session, noopReady, func(context.Context, Event) error {
		return saveErr
	})
	if err != saveErr || session != (GatewaySession{id: "gateway-session", seq: 11}) {
		t.Fatalf("RunGateway = %v, committed session = %+v", err, session)
	}
	second := mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
		if err := openAndReadResume(ctx, conn, "gateway-session", 11); err != nil {
			return err
		}
		return sendFrames(ctx, conn, ignored, replay, replay,
			`{"op":0,"s":13,"t":"RESUMED","d":""}`,
			`{"op":0,"s":14,"t":"GROUP_MSG_RECEIVE","id":"live","d":{"group_openid":"group","timestamp":1784570535}}`, `{"op":7}`)
	})
	var order []string
	err = second.RunGateway(ctx, mockGatewayEndpoint, mockGatewayToken, false, &session, func(context.Context) error {
		order = append(order, "ready")
		return nil
	}, func(_ context.Context, e Event) error {
		order = append(order, e.ID)
		return nil
	})
	if want := []string{"replayed", "ready", "live"}; err != errGatewayReconnect || session.seq != 14 || !reflect.DeepEqual(order, want) {
		t.Errorf("RunGateway = %v, session = %+v, callbacks = %v, want %v", err, session, order, want)
	}
}

func TestGatewayHeartbeatContinuesDuringPersistence(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	c := mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
		if err := sendFrames(ctx, conn, `{"op":10,"d":{"heartbeat_interval":150}}`); err != nil {
			return err
		}
		if err := readIdentify(ctx, conn); err != nil {
			return err
		}
		if err := sendFrames(ctx, conn, readyFrame, `{"op":0,"s":11,"t":"GROUP_ADD_ROBOT","d":{"group_openid":"group","timestamp":1784570534}}`); err != nil {
			return err
		}
		select {
		case <-started:
		case <-ctx.Done():
			return ctx.Err()
		}
		latest := int64(11)
		for range 2 {
			if err := readHeartbeat(ctx, conn, &latest); err != nil {
				return err
			}
			if err := sendFrames(ctx, conn, `{"op":11}`); err != nil {
				return err
			}
		}
		close(release)
		return sendFrames(ctx, conn, `{"op":7}`)
	})
	var session GatewaySession
	err := c.RunGateway(testGatewayContext(t), mockGatewayEndpoint, mockGatewayToken, false, &session, noopReady, func(ctx context.Context, _ Event) error {
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	if err != errGatewayReconnect || session.seq != 11 {
		t.Fatalf("RunGateway = %v, session = %+v", err, session)
	}
}

func TestGatewayHeartbeatACKTimeout(t *testing.T) {
	c := mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
		if err := sendFrames(ctx, conn, `{"op":10,"d":{"heartbeat_interval":100}}`); err != nil {
			return err
		}
		if err := readIdentify(ctx, conn); err != nil {
			return err
		}
		if err := sendFrames(ctx, conn, readyFrame); err != nil {
			return err
		}
		latest := int64(10)
		if err := readHeartbeat(ctx, conn, &latest); err != nil {
			return err
		}
		// No ACK: the client must disconnect before it sends another heartbeat.
		if _, _, err := conn.Read(ctx); err == nil {
			return errors.New("unexpected write after missing heartbeat ACK")
		}
		return nil
	})
	var session GatewaySession
	err := c.RunGateway(testGatewayContext(t), mockGatewayEndpoint, mockGatewayToken, false, &session, noopReady, noopEvent)
	if err != errGatewayHeartbeat || session != (GatewaySession{id: "gateway-session", seq: 10}) {
		t.Fatalf("RunGateway = %v, session = %+v", err, session)
	}
}

func TestGatewayCloseCodes(t *testing.T) {
	for _, tc := range []struct {
		code  websocket.StatusCode
		cause error
		reset bool
	}{
		{4001, ErrGatewayBlocked, true}, {4002, ErrGatewayBlocked, true}, {4004, ErrGatewayInvalidToken, true},
		{4006, nil, true}, {4007, nil, true}, {4008, nil, false}, {4009, nil, false},
		{4010, ErrGatewayBlocked, true}, {4011, ErrGatewayBlocked, true}, {4012, ErrGatewayBlocked, true},
		{4013, ErrGatewayBlocked, true}, {4014, ErrGatewayIntentForbidden, true},
		{4900, nil, true}, {4913, nil, true}, {4914, ErrGatewayBlocked, true}, {4915, ErrGatewayBlocked, true},
		{websocket.StatusGoingAway, nil, false},
	} {
		t.Run(fmt.Sprint(tc.code), func(t *testing.T) {
			original := GatewaySession{id: "gateway-session", seq: 12}
			c := mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
				if err := openAndReadResume(ctx, conn, original.id, original.seq); err != nil {
					return err
				}
				return conn.Close(tc.code, "provider private reason")
			})
			session := original
			err := c.RunGateway(testGatewayContext(t), mockGatewayEndpoint, mockGatewayToken, false, &session, noopReady, noopEvent)
			if err == nil || (tc.cause != nil && !errors.Is(err, tc.cause)) || (tc.cause == nil && (errors.Is(err, ErrGatewayBlocked) || errors.Is(err, ErrGatewayInvalidToken))) || strings.Contains(err.Error(), "private reason") {
				t.Errorf("unclassified or unsanitized close: %v", err)
			}
			want := original
			if tc.reset {
				want = GatewaySession{}
			}
			if session != want {
				t.Errorf("session after close = %+v, want %+v", session, want)
			}
		})
	}
}

func TestGatewayInvalidSessionForcesIdentify(t *testing.T) {
	c := mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
		if err := openAndReadResume(ctx, conn, "gateway-session", 12); err != nil {
			return err
		}
		return sendFrames(ctx, conn, `{"op":9,"d":true}`)
	})
	session := GatewaySession{id: "gateway-session", seq: 12}
	ctx := testGatewayContext(t)
	if err := c.RunGateway(ctx, mockGatewayEndpoint, mockGatewayToken, false, &session, noopReady, noopEvent); err != errGatewayInvalidSession || session != (GatewaySession{}) {
		t.Fatalf("RunGateway = %v, session = %+v", err, session)
	}
	c = mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
		if err := openAndReadIdentify(ctx, conn); err != nil {
			return err
		}
		return sendFrames(ctx, conn, readyFrame, `{"op":7}`)
	})
	if err := c.RunGateway(ctx, mockGatewayEndpoint, mockGatewayToken, false, &session, noopReady, noopEvent); err != errGatewayReconnect || session != (GatewaySession{id: "gateway-session", seq: 10}) {
		t.Fatalf("RunGateway after invalid session = %v, session = %+v", err, session)
	}
}

func TestGatewayOptionalGroupMemberIntent(t *testing.T) {
	ctx := testGatewayContext(t)
	// Changing intents must use Identify, not resume the old subscription.
	session := GatewaySession{id: "old-session", seq: 20}
	first := mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
		if err := sendFrames(ctx, conn, helloFrame); err != nil {
			return err
		}
		if err := readIdentifyIntents(ctx, conn, 1<<25|1<<24); err != nil {
			return err
		}
		return sendFrames(ctx, conn, readyFrame, `{"op":7}`)
	})
	if err := first.RunGateway(ctx, mockGatewayEndpoint, mockGatewayToken, true, &session, noopReady, noopEvent); err != errGatewayReconnect || session != (GatewaySession{id: "gateway-session", seq: 10, joinRequests: true}) {
		t.Fatalf("optional subscription RunGateway = %v, session = %+v", err, session)
	}
	second := mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
		if err := openAndReadResume(ctx, conn, "gateway-session", 10); err != nil {
			return err
		}
		return conn.Close(4014, "private provider reason")
	})
	err := second.RunGateway(ctx, mockGatewayEndpoint, mockGatewayToken, true, &session, noopReady, noopEvent)
	if !errors.Is(err, ErrGatewayIntentForbidden) || !errors.Is(err, ErrGatewayBlocked) || session != (GatewaySession{}) {
		t.Fatalf("intent close RunGateway = %v, session = %+v", err, session)
	}
}

func TestGatewayCancellationAndMalformedSupportedEvent(t *testing.T) {
	t.Run("cancel while waiting for hello", func(t *testing.T) {
		connected := make(chan struct{})
		c := mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
			close(connected)
			_, _, _ = conn.Read(ctx)
			return nil
		})
		ctx, cancel := context.WithCancel(testGatewayContext(t))
		defer cancel()
		go func() {
			select {
			case <-connected:
				cancel()
			case <-ctx.Done():
			}
		}()
		var session GatewaySession
		if err := c.RunGateway(ctx, mockGatewayEndpoint, mockGatewayToken, false, &session, noopReady, noopEvent); err != context.Canceled {
			t.Fatalf("RunGateway on cancellation = %v", err)
		}
	})
	t.Run("malformed replay forces identify instead of replaying again", func(t *testing.T) {
		c := mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
			if err := openAndReadResume(ctx, conn, "gateway-session", 10); err != nil {
				return err
			}
			return sendFrames(ctx, conn, `{"op":0,"s":11,"t":"GROUP_ADD_ROBOT","d":{"group_openid":"group","timestamp":"1784570534"}}`)
		})
		session := GatewaySession{id: "gateway-session", seq: 10}
		ctx := testGatewayContext(t)
		err := c.RunGateway(ctx, mockGatewayEndpoint, mockGatewayToken, false, &session, func(context.Context) error {
			t.Error("malformed replay became ready")
			return nil
		}, func(context.Context, Event) error {
			t.Error("malformed supported dispatch reached callback")
			return nil
		})
		if err != errGatewayMalformedEvent || session != (GatewaySession{}) {
			t.Fatalf("RunGateway = %v, session = %+v", err, session)
		}
		c = mockGateway(t, func(ctx context.Context, conn *websocket.Conn) error {
			if err := openAndReadIdentify(ctx, conn); err != nil {
				return err
			}
			return sendFrames(ctx, conn, readyFrame, `{"op":7}`)
		})
		if err := c.RunGateway(ctx, mockGatewayEndpoint, mockGatewayToken, false, &session, noopReady, noopEvent); err != errGatewayReconnect {
			t.Fatalf("RunGateway after malformed replay = %v", err)
		}
	})
}

func testGatewayContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func mockGateway(t *testing.T, serve func(context.Context, *websocket.Conn) error) *Client {
	t.Helper()
	done := make(chan struct{})
	started := make(chan struct{})
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		defer close(done)
		if r.ProtoMajor != 1 || r.Host != gatewayHost || r.URL.Path != "/websocket/" || r.Header.Get("Authorization") != "" {
			t.Errorf("bad upgrade request: %s %s %s, auth present=%v", r.Proto, r.Host, r.URL, r.Header.Get("Authorization") != "")
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("mock WebSocket upgrade: %v", err)
			return
		}
		defer func() { _ = conn.CloseNow() }()
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancel()
		if err := serve(ctx, conn); err != nil {
			t.Errorf("mock gateway: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		select {
		case <-started:
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Error("mock gateway did not stop")
			}
		default:
		}
	})
	c := New("app", "secret")
	transport := c.http.Transport.(*http.Transport)
	// Keep production upgrade and redirect options; only route TLS to the mock.
	transport.TLSClientConfig = srv.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	transport.TLSClientConfig.ServerName = "example.com"
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != gatewayHost+":443" {
			return nil, errors.New("mock gateway expected official endpoint")
		}
		return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
	}
	return c
}

func sendFrames(ctx context.Context, conn *websocket.Conn, frames ...string) error {
	for _, frame := range frames {
		if err := conn.Write(ctx, websocket.MessageText, []byte(frame)); err != nil {
			return err
		}
	}
	return nil
}

func openAndReadIdentify(ctx context.Context, conn *websocket.Conn) error {
	if err := sendFrames(ctx, conn, helloFrame); err != nil {
		return err
	}
	return readIdentify(ctx, conn)
}

func readIdentify(ctx context.Context, conn *websocket.Conn) error {
	return readIdentifyIntents(ctx, conn, 1<<25)
}

func readIdentifyIntents(ctx context.Context, conn *websocket.Conn, intents int) error {
	var got gatewayPacket[gatewayIdentify]
	if err := wsjson.Read(ctx, conn, &got); err != nil {
		return err
	}
	want := gatewayPacket[gatewayIdentify]{Op: 2, Data: gatewayIdentify{Token: "QQBot " + mockGatewayToken, Intents: intents, Shard: [2]int{0, 1}}}
	if got != want {
		return fmt.Errorf("Identify = %+v, want %+v", got, want)
	}
	return nil
}

func openAndReadResume(ctx context.Context, conn *websocket.Conn, sessionID string, seq int64) error {
	if err := sendFrames(ctx, conn, helloFrame); err != nil {
		return err
	}
	var got gatewayPacket[gatewayResume]
	if err := wsjson.Read(ctx, conn, &got); err != nil {
		return err
	}
	want := gatewayPacket[gatewayResume]{Op: 6, Data: gatewayResume{Token: mockGatewayToken, SessionID: sessionID, Seq: seq}}
	if got != want {
		return fmt.Errorf("Resume = %+v, want %+v", got, want)
	}
	return nil
}

func readHeartbeat(ctx context.Context, conn *websocket.Conn, seq *int64) error {
	var got gatewayPacket[json.RawMessage]
	want := "null"
	if seq != nil {
		want = fmt.Sprint(*seq)
	}
	if err := wsjson.Read(ctx, conn, &got); err != nil {
		return err
	}
	if got.Op != 1 || string(got.Data) != want {
		return fmt.Errorf("Heartbeat = %d %s, want 1 %s", got.Op, got.Data, want)
	}
	return nil
}

func noopReady(context.Context) error        { return nil }
func noopEvent(context.Context, Event) error { return nil }
