package service

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"amiya-eden/internal/model"

	"github.com/google/uuid"
	"golang.org/x/net/websocket"
)

var errQQBotDisconnected = errors.New("connection ended")

type qqbotResponse struct {
	FailureClass string          `json:"-"`
	Echo         string          `json:"echo"`
	Status       string          `json:"status"`
	Retcode      *int            `json:"retcode"`
	Data         json.RawMessage `json:"data"`
}
type qqbotReply struct {
	response qqbotResponse
	err      error
}
type qqbotConnection struct {
	ws      *websocket.Conn
	write   sync.Mutex
	pending map[string]chan qqbotReply
}
type qqbotClient struct {
	mu           sync.Mutex
	current      *qqbotConnection
	closed       bool
	generation   uint64
	events       chan []byte
	ingressError string
	lastFrame    time.Time
	timeout      time.Duration
}

func newQQBotClient() *qqbotClient {
	return &qqbotClient{events: make(chan []byte, 128), timeout: 10 * time.Second}
}
func qqbotAuthenticate(r *http.Request, s model.QQBotSetting) bool {
	if !s.Enabled || s.Token == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+s.Token)) != 1 || r.Header.Get("X-Self-ID") != int64String(s.BotID) || !strings.EqualFold(r.Header.Get("X-Client-Role"), "Universal") {
		return false
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, raw := range strings.Split(s.CIDRs, ",") {
		_, block, e := net.ParseCIDR(strings.TrimSpace(raw))
		if e == nil && block.Contains(ip) {
			return true
		}
	}
	return false
}
func (c *qqbotClient) serve(w http.ResponseWriter, r *http.Request, s model.QQBotSetting, generation uint64) {
	if !qqbotAuthenticate(r, s) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	server := websocket.Server{Handshake: func(*websocket.Config, *http.Request) error { return nil }, Handler: func(ws *websocket.Conn) { c.accept(ws, s.BotID, generation) }}
	server.ServeHTTP(w, r)
}
func (c *qqbotClient) dropLocked(conn *qqbotConnection) {
	if conn == nil {
		return
	}
	_ = conn.ws.SetWriteDeadline(time.Now().Add(time.Second))
	for echo, ch := range conn.pending {
		ch <- qqbotReply{err: errQQBotDisconnected}
		delete(conn.pending, echo)
	}
	_ = conn.ws.Close()
	if c.current == conn {
		c.current = nil
	}
}
func (c *qqbotClient) disconnect() { c.mu.Lock(); defer c.mu.Unlock(); c.dropLocked(c.current) }
func (c *qqbotClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	c.dropLocked(c.current)
}
func (c *qqbotClient) accept(ws *websocket.Conn, botID int64, generation uint64) {
	ws.MaxPayloadBytes = 1 << 20
	conn := &qqbotConnection{ws: ws, pending: make(map[string]chan qqbotReply)}
	c.mu.Lock()
	if c.closed || generation != c.generation {
		c.mu.Unlock()
		_ = ws.Close()
		return
	}
	c.dropLocked(c.current)
	c.current = conn
	c.lastFrame = time.Now()
	c.mu.Unlock()
	defer func() { c.mu.Lock(); c.dropLocked(conn); c.mu.Unlock() }()
	for {
		if err := ws.SetReadDeadline(time.Now().Add(90 * time.Second)); err != nil {
			return
		}
		var raw []byte
		if err := websocket.Message.Receive(ws, &raw); err != nil {
			return
		}
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(raw, &envelope); err != nil {
			c.setIngressError("invalid JSON frame")
			return
		}
		c.mu.Lock()
		c.lastFrame = time.Now()
		c.mu.Unlock()
		if _, ok := envelope["echo"]; ok {
			var resp qqbotResponse
			err := json.Unmarshal(raw, &resp)
			if err != nil || resp.Echo == "" || resp.Retcode == nil || len(resp.Data) == 0 || (resp.Status != "ok" && resp.Status != "failed" && resp.Status != "async") || (resp.Status == "ok" && *resp.Retcode != 0) || (resp.Status == "failed" && *resp.Retcode == 0) {
				err = errors.New("invalid response")
			}
			c.mu.Lock()
			ch := conn.pending[resp.Echo]
			delete(conn.pending, resp.Echo)
			c.mu.Unlock()
			if ch != nil {
				ch <- qqbotReply{response: resp, err: err}
			} else if err != nil {
				c.setIngressError("invalid response frame")
				return
			}
			continue
		}
		var event struct {
			SelfID   int64  `json:"self_id"`
			Time     int64  `json:"time"`
			PostType string `json:"post_type"`
		}
		if json.Unmarshal(raw, &event) != nil || event.SelfID != botID || event.Time <= 0 || event.PostType == "" {
			c.setIngressError("invalid event frame")
			return
		}
		// The authenticated connection identity is also checked in the event worker.
		select {
		case c.events <- raw:
		default:
			c.setIngressError("event ingress full; connection closed before persistence")
			return
		}
	}
}
func (c *qqbotClient) setIngressError(s string) { c.mu.Lock(); c.ingressError = s; c.mu.Unlock() }
func (c *qqbotClient) connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current != nil && !c.closed
}

// call is private: only the durable dispatcher may execute protocol operations.
func (c *qqbotClient) call(ctx context.Context, action string, params interface{}) (qqbotResponse, string) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if ctx.Err() != nil {
		return qqbotResponse{}, "disconnected"
	}
	c.mu.Lock()
	conn := c.current
	if conn == nil || c.closed {
		c.mu.Unlock()
		return qqbotResponse{}, "disconnected"
	}
	if len(conn.pending) >= 128 {
		c.mu.Unlock()
		return qqbotResponse{}, "busy"
	}
	echo := uuid.NewString()
	ch := make(chan qqbotReply, 1)
	conn.pending[echo] = ch
	c.mu.Unlock()
	defer func() { c.mu.Lock(); delete(conn.pending, echo); c.mu.Unlock() }()
	conn.write.Lock()
	if ctx.Err() != nil {
		conn.write.Unlock()
		return qqbotResponse{}, "disconnected"
	}
	deadline := time.Now().Add(c.timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	err := conn.ws.SetWriteDeadline(deadline)
	if err == nil {
		err = websocket.JSON.Send(conn.ws, struct {
			Action string      `json:"action"`
			Params interface{} `json:"params"`
			Echo   string      `json:"echo"`
		}{action, params, echo})
	}
	conn.write.Unlock()
	if err != nil {
		return qqbotResponse{FailureClass: "send"}, "unknown"
	}
	select {
	case reply := <-ch:
		if reply.err != nil {
			cause := "protocol"
			if errors.Is(reply.err, errQQBotDisconnected) {
				cause = "disconnected"
			}
			return qqbotResponse{FailureClass: cause}, "unknown"
		}
		resp := reply.response
		if resp.Status == "async" {
			resp.FailureClass = "async"
			return resp, "unknown"
		}
		if resp.Status == "failed" {
			if *resp.Retcode == 100 || *resp.Retcode == 102 || *resp.Retcode == 103 || *resp.Retcode == 104 || *resp.Retcode == 1400 || *resp.Retcode == 1401 || *resp.Retcode == 1403 || *resp.Retcode == 1404 {
				return resp, "permanent"
			}
			return resp, "transient"
		}
		return resp, "success"
	case <-ctx.Done():
		cause := "cancelled"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			cause = "timeout"
		}
		return qqbotResponse{FailureClass: cause}, "unknown"
	}
}

func (c *qqbotClient) reconfigure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.generation++
	c.dropLocked(c.current)
}
