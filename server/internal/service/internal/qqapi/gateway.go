package qqapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	gatewayHost             = "api.bot.qq.com"
	groupAndC2CIntent       = 1 << 25
	groupMemberIntent       = 1 << 24
	gatewayDispatchQueue    = 32
	gatewayHandshakeTimeout = 10 * time.Second
	gatewayReadyTimeout     = 60 * time.Second
	gatewayWriteTimeout     = 5 * time.Second
	maxHeartbeatInterval    = 5 * time.Minute
	maxGatewayURLBytes      = 2048
	maxSessionIDBytes       = 256
	noGatewaySequence       = -1
)

const (
	opDispatch       = 0
	opHeartbeat      = 1
	opIdentify       = 2
	opResume         = 6
	opReconnect      = 7
	opInvalidSession = 9
	opHello          = 10
	opHeartbeatACK   = 11
)

var (
	// ErrGatewayBlocked means QQ disallows both resuming and identifying again.
	ErrGatewayBlocked = errors.New("qqapi: gateway blocked")
	// ErrGatewayIntentForbidden is a block specific to the requested intents.
	ErrGatewayIntentForbidden = fmt.Errorf("%w: intent forbidden", ErrGatewayBlocked)
	// ErrGatewayInvalidToken means QQ rejected the supplied access token.
	ErrGatewayInvalidToken = errors.New("qqapi: gateway rejected access token")

	errGatewayEndpoint       = errors.New("qqapi: invalid gateway endpoint")
	errGatewayArguments      = errors.New("qqapi: invalid gateway arguments")
	errGatewayConnect        = errors.New("qqapi: gateway connection failed")
	errGatewayDisconnected   = errors.New("qqapi: gateway disconnected")
	errGatewayProtocol       = errors.New("qqapi: invalid gateway frame")
	errGatewayMalformedEvent = errors.New("qqapi: malformed gateway event")
	errGatewayWrite          = errors.New("qqapi: gateway write failed")
	errGatewayHandshake      = errors.New("qqapi: gateway hello timeout")
	errGatewayReady          = errors.New("qqapi: gateway readiness timeout")
	errGatewayHeartbeat      = errors.New("qqapi: gateway heartbeat ACK timeout")
	errGatewayBacklog        = errors.New("qqapi: gateway dispatch queue full")
	errGatewayReconnect      = errors.New("qqapi: gateway requested reconnect")
	errGatewayInvalidSession = errors.New("qqapi: gateway invalid session")
)

// GatewaySession holds the in-memory resume checkpoint for one shard. Use a
// zero value for a new session; keep it with the same bot and access scope and
// pass it to subsequent RunGateway calls serially.
type GatewaySession struct {
	id           string
	seq          int64
	joinRequests bool
}

// DiscoverGateway retrieves the production, unsharded endpoint from official
// QQ OpenAPI. The result is validated again before dialing it.
func (c *Client) DiscoverGateway(ctx context.Context, token string) (string, error) {
	if !visibleASCII(token, maxTokenBytes) {
		return "", &OperationError{Kind: InvalidToken}
	}
	r, err := c.call(ctx, http.MethodGet, "/gateway", token, nil, readOp)
	if err != nil {
		var op *OperationError
		// Tencent's SDK additionally identifies 11244 as expired/absent token.
		if errors.As(err, &op) && (op.Kind == InvalidToken || (op.Kind != Throttle && op.Code == 11244)) {
			classified := *op
			classified.Kind = InvalidToken
			return "", errors.Join(ErrGatewayInvalidToken, &classified)
		}
		return "", err
	}
	var out struct {
		URL string `json:"url"`
	}
	if r.status != http.StatusOK || json.Unmarshal(r.body, &out) != nil || !validGatewayEndpoint(out.URL) {
		return "", r.failure(TemporaryNotApplied)
	}
	return out.URL, nil
}

func validGatewayEndpoint(endpoint string) bool {
	if !visibleASCII(endpoint, maxGatewayURLBytes) || strings.Contains(endpoint, "#") {
		return false
	}
	u, err := url.Parse(endpoint)
	return err == nil && u.Scheme == "wss" && u.User == nil && u.Opaque == "" &&
		(strings.EqualFold(u.Host, gatewayHost) || strings.EqualFold(u.Host, gatewayHost+":443"))
}

// RunGateway runs one connection; the caller owns retries and token refresh.
// If joinRequests is true, it also requests QQ's group member intent. Both
// callbacks run serially and must observe the context. On resume, replay is
// persisted before onReady, which is called only after QQ sends RESUMED.
func (c *Client) RunGateway(ctx context.Context, endpoint, token string, joinRequests bool, session *GatewaySession, onReady func(context.Context) error, onEvent func(context.Context, Event) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validGatewayEndpoint(endpoint) {
		return errGatewayEndpoint
	}
	if !visibleASCII(token, maxTokenBytes) || session == nil || onReady == nil || onEvent == nil {
		return errGatewayArguments
	}
	// New pins the TLS transport to direct connections and rejects redirects.
	// No credential is sent in the upgrade; Identify/Resume carry it in WSS.
	conn, _, err := websocket.Dial(ctx, endpoint, &websocket.DialOptions{HTTPClient: c.http})
	if err != nil {
		return gatewayOutcome(ctx, session, errGatewayConnect)
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(maxResponseBodyBytes)
	if session.id != "" && session.joinRequests != joinRequests {
		*session = GatewaySession{} // Resume cannot renegotiate the intent set.
	}

	interval, err := gatewayHello(ctx, conn)
	if err != nil {
		return gatewayOutcome(ctx, session, err)
	}
	resuming := session.id != ""
	seq := int64(noGatewaySequence)
	if resuming {
		seq = session.seq
		err = writeGateway(ctx, conn, gatewayPacket[gatewayResume]{Op: opResume, Data: gatewayResume{Token: token, SessionID: session.id, Seq: seq}})
	} else {
		intents := groupAndC2CIntent
		if joinRequests {
			intents |= groupMemberIntent
		}
		err = writeGateway(ctx, conn, gatewayPacket[gatewayIdentify]{Op: opIdentify, Data: gatewayIdentify{Token: "QQBot " + token, Intents: intents, Shard: [2]int{0, 1}}})
	}
	if err != nil {
		return gatewayOutcome(ctx, session, err)
	}
	return runGateway(ctx, conn, interval, seq, resuming, joinRequests, session, onReady, onEvent)
}

type gatewayPacket[T any] struct {
	Op   int `json:"op"`
	Data T   `json:"d"`
}

type gatewayIdentify struct {
	Token   string `json:"token"`
	Intents int    `json:"intents"`
	Shard   [2]int `json:"shard"`
}

type gatewayResume struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Seq       int64  `json:"seq"`
}

type gatewayFrame struct {
	Op   *int   `json:"op"`
	Seq  *int64 `json:"s"`
	Type string `json:"t"`
}

func readGatewayFrame(ctx context.Context, conn *websocket.Conn) (gatewayFrame, []byte, error) {
	kind, raw, err := conn.Read(ctx)
	if err != nil {
		if code := websocket.CloseStatus(err); code >= 0 {
			return gatewayFrame{}, nil, gatewayClose(code)
		}
		return gatewayFrame{}, nil, errGatewayDisconnected
	}
	var frame gatewayFrame
	if kind != websocket.MessageText || json.Unmarshal(raw, &frame) != nil || frame.Op == nil || *frame.Op < 0 {
		return gatewayFrame{}, nil, errGatewayProtocol
	}
	return frame, raw, nil
}

func writeGateway[T any](ctx context.Context, conn *websocket.Conn, packet T) error {
	writeCtx, cancel := context.WithTimeout(ctx, gatewayWriteTimeout)
	defer cancel()
	if wsjson.Write(writeCtx, conn, packet) != nil {
		return errGatewayWrite
	}
	return nil
}

func gatewayHello(ctx context.Context, conn *websocket.Conn) (time.Duration, error) {
	helloCtx, cancel := context.WithTimeout(ctx, gatewayHandshakeTimeout)
	defer cancel()
	frame, raw, err := readGatewayFrame(helloCtx, conn)
	if err != nil {
		if errors.Is(helloCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
			return 0, errGatewayHandshake
		}
		return 0, err
	}
	switch *frame.Op {
	case opReconnect:
		return 0, errGatewayReconnect
	case opInvalidSession:
		return 0, errGatewayInvalidSession
	case opHello:
		var hello gatewayPacket[struct {
			HeartbeatMillis int64 `json:"heartbeat_interval"`
		}]
		if json.Unmarshal(raw, &hello) == nil && hello.Data.HeartbeatMillis > 0 && hello.Data.HeartbeatMillis <= maxHeartbeatInterval.Milliseconds() {
			return time.Duration(hello.Data.HeartbeatMillis) * time.Millisecond, nil
		}
	}
	return 0, errGatewayProtocol
}

type gatewayDispatch struct {
	seq       int64
	sessionID string // Only set for READY.
	ready     bool   // READY or RESUMED, never regular dispatches.
	event     Event  // Zero for ignored dispatches.
}

type gatewayConnection struct {
	*websocket.Conn
	receivedSeq atomic.Int64 // Heartbeats use received, not committed, sequence numbers.
	awaitingACK atomic.Bool
}

func runGateway(ctx context.Context, conn *websocket.Conn, interval time.Duration, seq int64, resuming, joinRequests bool, session *GatewaySession, onReady func(context.Context) error, onEvent func(context.Context, Event) error) (err error) {
	live, stop := context.WithCancelCause(ctx)
	wire := gatewayConnection{Conn: conn}
	wire.receivedSeq.Store(seq)
	queue := make(chan gatewayDispatch, gatewayDispatchQueue)
	var wg sync.WaitGroup
	wg.Go(func() {
		defer close(queue)
		stop(wire.receive(live, resuming, queue))
	})
	wg.Go(func() { stop(wire.heartbeat(live, interval)) })
	defer func() {
		stop(nil)
		wg.Wait()
		cause := context.Cause(live)
		if cause == errGatewayMalformedEvent && err == nil {
			// Earlier queued callbacks succeeded. Identify rather than replay
			// this undecodable event forever. If an earlier persist failed, keep
			// its checkpoint so that valid event can still be replayed first.
			*session = GatewaySession{}
		}
		outcome := gatewayOutcome(ctx, session, cause)
		if err == nil {
			err = outcome
		}
	}()
	for d := range queue {
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.event.ID != "" {
			if err := onEvent(ctx, d.event); err != nil {
				return err
			}
		}
		if d.sessionID != "" {
			*session = GatewaySession{id: d.sessionID, seq: d.seq, joinRequests: joinRequests}
		} else if d.seq > session.seq {
			session.seq = d.seq
		}
		if d.ready {
			if err := onReady(ctx); err != nil {
				return err
			}
		}
	}
	return nil // The read/heartbeat pump supplies the sanitized terminal error.
}

func (c *gatewayConnection) receive(ctx context.Context, resuming bool, queue chan<- gatewayDispatch) error {
	readyCtx, stop := context.WithTimeout(ctx, gatewayReadyTimeout)
	defer stop()
	readCtx := readyCtx
	for {
		frame, raw, err := readGatewayFrame(readCtx, c.Conn)
		if err != nil {
			if readCtx == readyCtx && errors.Is(readyCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
				return errGatewayReady
			}
			return err
		}
		switch *frame.Op {
		case opDispatch:
			d, err := c.dispatch(frame, raw, readCtx == readyCtx, resuming)
			if err != nil {
				return err
			}
			if d.seq == noGatewaySequence {
				continue
			}
			if d.ready {
				readCtx = ctx
				stop()
			}
			select {
			case queue <- d:
			case <-ctx.Done():
				return context.Cause(ctx)
			default:
				return errGatewayBacklog
			}
		case opHeartbeat:
			// The payload docs specify Op 11 as the reply to a server heartbeat.
			if err := writeGateway(readCtx, c.Conn, struct {
				Op int `json:"op"`
			}{Op: opHeartbeatACK}); err != nil {
				return err
			}
		case opHeartbeatACK:
			c.awaitingACK.Store(false)
		case opReconnect:
			return errGatewayReconnect
		case opInvalidSession:
			return errGatewayInvalidSession
		case opHello:
			return errGatewayProtocol
		}
	}
}

func (c *gatewayConnection) dispatch(frame gatewayFrame, raw []byte, waiting, resuming bool) (gatewayDispatch, error) {
	if frame.Seq == nil || *frame.Seq < 0 || frame.Type == "" {
		return gatewayDispatch{}, errGatewayProtocol
	}
	d := gatewayDispatch{seq: *frame.Seq}
	lastReceived := c.receivedSeq.Load()
	switch frame.Type {
	case "READY":
		if !waiting || resuming {
			return gatewayDispatch{}, errGatewayProtocol
		}
		var ready gatewayPacket[struct {
			SessionID string `json:"session_id"`
		}]
		if json.Unmarshal(raw, &ready) != nil || !visibleASCII(ready.Data.SessionID, maxSessionIDBytes) {
			return gatewayDispatch{}, errGatewayProtocol
		}
		d.sessionID, d.ready = ready.Data.SessionID, true
	case "RESUMED":
		if !waiting || !resuming || d.seq < lastReceived {
			return gatewayDispatch{}, errGatewayProtocol
		}
		d.ready = true
	default:
		if waiting && !resuming {
			return gatewayDispatch{}, errGatewayProtocol
		}
		if d.seq <= lastReceived {
			// The ordered reader already queued/committed this sequence.
			return gatewayDispatch{seq: noGatewaySequence}, nil
		}
		event, err := parseGatewayEvent(frame.Type, raw)
		if err != nil {
			return gatewayDispatch{}, err
		}
		d.event = event
	}
	c.receivedSeq.Store(d.seq)
	return d, nil
}

func (c *gatewayConnection) heartbeat(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-ticker.C:
			if !c.awaitingACK.CompareAndSwap(false, true) {
				return errGatewayHeartbeat
			}
			var latest *int64
			if seq := c.receivedSeq.Load(); seq != noGatewaySequence {
				latest = &seq
			}
			if err := writeGateway(ctx, c.Conn, gatewayPacket[*int64]{Op: opHeartbeat, Data: latest}); err != nil {
				return err
			}
		}
	}
}

// Retain only the numeric WebSocket close status, not QQ's free-form reason.
type gatewayClose websocket.StatusCode

func (gatewayClose) Error() string { return errGatewayDisconnected.Error() }

func gatewayOutcome(ctx context.Context, session *GatewaySession, err error) error {
	var code gatewayClose
	switch {
	case errors.Is(err, errGatewayInvalidSession):
		*session = GatewaySession{}
	case errors.As(err, &code):
		err = errGatewayDisconnected
		switch {
		case code == 4004: // Tencent botgo's backend authentication failure.
			*session = GatewaySession{}
			err = ErrGatewayInvalidToken
		case code == 4006 || code == 4007 || (4900 <= code && code <= 4913):
			*session = GatewaySession{} // Retry with Identify, not Resume.
		case code == 4014:
			*session = GatewaySession{}
			err = ErrGatewayIntentForbidden
		case code == 4001 || code == 4002 || (4010 <= code && code <= 4013) || code == 4914 || code == 4915:
			*session = GatewaySession{}
			err = ErrGatewayBlocked
		}
		err = fmt.Errorf("%w (close %d)", err, code)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
