package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/service"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
)

// QQGovernanceOneBotHandler 只负责 OneBot V11 协议、连接鉴权与 echo 匹配；业务决策在 service 层。
type QQGovernanceOneBotHandler struct {
	manager *qqGovernanceOneBotConnectionManager
}

func NewQQGovernanceOneBotHandler() *QQGovernanceOneBotHandler {
	svc := service.DefaultQQGovernanceService()
	m := newQQGovernanceOneBotConnectionManager(svc.HandleOneBotEvent)
	svc.SetOneBotActionExecutor(m)
	return &QQGovernanceOneBotHandler{manager: m}
}
func (h *QQGovernanceOneBotHandler) ReverseWebSocket(c *gin.Context) {
	if err := validateOneBotReverseRequest(c.Request); err != nil {
		oneBotLogger().Warn("OneBot reverse WebSocket rejected",
			zap.String("remote_addr", c.Request.RemoteAddr),
			zap.String("self_id", strings.TrimSpace(c.GetHeader("X-Self-ID"))),
			zap.Bool("has_authorization", strings.TrimSpace(c.GetHeader("Authorization")) != ""),
			zap.String("request_id", c.GetString("request-id")),
			zap.Error(err),
		)
		c.String(http.StatusForbidden, "OneBot reverse WebSocket rejected")
		return
	}
	oneBotLogger().Info("OneBot reverse WebSocket accepted",
		zap.String("remote_addr", c.Request.RemoteAddr),
		zap.String("self_id", strings.TrimSpace(c.GetHeader("X-Self-ID"))),
		zap.String("request_id", c.GetString("request-id")),
	)
	websocket.Server{
		// NapCat is a non-browser OneBot client and normally does not send Origin.
		// Authentication and source-network checks above remain the access control.
		Handshake: func(*websocket.Config, *http.Request) error { return nil },
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			h.manager.Serve(ws, c.Request.Context())
		}),
	}.ServeHTTP(c.Writer, c.Request)
}

func validateOneBotReverseRequest(r *http.Request) error {
	if r == nil {
		return errors.New("OneBot 配置不可用")
	}
	cfg := service.NewSysConfigService().GetOneBotConfig()
	if !cfg.Enabled || cfg.BotQQ <= 0 || strings.TrimSpace(cfg.AccessToken) == "" {
		return errors.New("OneBot 反向连接未启用或配置不完整")
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		return errors.New("OneBot 反向连接缺少令牌")
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))), []byte(cfg.AccessToken)) != 1 {
		return errors.New("OneBot 令牌无效")
	}
	selfID, err := strconv.ParseInt(strings.TrimSpace(r.Header.Get("X-Self-ID")), 10, 64)
	if err != nil || selfID != cfg.BotQQ {
		return errors.New("OneBot 机器人 QQ 不匹配")
	}
	if len(cfg.AllowedCIDRs) == 0 {
		return errors.New("OneBot 未配置受控网段")
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return errors.New("OneBot 来源 IP 无效")
	}
	for _, raw := range cfg.AllowedCIDRs {
		_, subnet, parseErr := net.ParseCIDR(strings.TrimSpace(raw))
		if parseErr == nil && subnet.Contains(ip) {
			return nil
		}
	}
	return errors.New("OneBot 来源不在受控网段")
}

type qqGovernanceOneBotConnectionManager struct {
	mu      sync.RWMutex
	current *qqGovernanceOneBotConnection
	pending map[string]chan oneBotActionResponse
	handle  func(context.Context, service.QQGovernanceInboundEvent) error
}
type qqGovernanceOneBotConnection struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}
type oneBotActionResponse struct {
	Status  string          `json:"status"`
	RetCode int             `json:"retcode"`
	Wording string          `json:"wording"`
	Data    json.RawMessage `json:"data"`
}

func newQQGovernanceOneBotConnectionManager(handle func(context.Context, service.QQGovernanceInboundEvent) error) *qqGovernanceOneBotConnectionManager {
	return &qqGovernanceOneBotConnectionManager{pending: map[string]chan oneBotActionResponse{}, handle: handle}
}
func (m *qqGovernanceOneBotConnectionManager) OneBotConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current != nil
}
func (m *qqGovernanceOneBotConnectionManager) Serve(conn *websocket.Conn, ctx context.Context) {
	if conn == nil {
		return
	}
	connection := &qqGovernanceOneBotConnection{conn: conn}
	m.mu.Lock()
	old := m.current
	m.current = connection
	m.mu.Unlock()
	if old != nil {
		m.logger().Warn("OneBot reverse WebSocket replaced an existing connection")
		_ = old.conn.Close()
	}
	m.logger().Info("OneBot reverse WebSocket connected")
	defer func() {
		m.mu.Lock()
		if m.current == connection {
			m.current = nil
		}
		m.mu.Unlock()
		_ = conn.Close()
		m.logger().Info("OneBot reverse WebSocket disconnected")
	}()
	for {
		var raw string
		if err := websocket.Message.Receive(conn, &raw); err != nil {
			m.logger().Info("OneBot reverse WebSocket receive stopped", zap.Error(err))
			return
		}
		m.handleMessage(ctx, []byte(raw))
	}
}
func (m *qqGovernanceOneBotConnectionManager) handleMessage(ctx context.Context, raw []byte) {
	var envelope struct {
		Echo json.RawMessage `json:"echo"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		m.logger().Debug("Ignoring invalid OneBot message", zap.Error(err))
		return
	}
	if len(envelope.Echo) > 0 && string(envelope.Echo) != "null" {
		var echo string
		if json.Unmarshal(envelope.Echo, &echo) == nil && m.deliver(echo, raw) {
			return
		}
	}
	event, ok := parseOneBotGovernanceEvent(raw)
	if !ok || m.handle == nil {
		return
	}
	if err := m.handle(ctx, event); err != nil {
		m.logger().Warn("处理 OneBot 群治理事件失败", zap.String("event_type", event.EventType), zap.Int64("group_id", event.GroupID), zap.Int64("qq", event.QQ), zap.Error(err))
	}
}
func (m *qqGovernanceOneBotConnectionManager) deliver(echo string, raw []byte) bool {
	var response oneBotActionResponse
	if json.Unmarshal(raw, &response) != nil {
		return false
	}
	m.mu.Lock()
	ch, ok := m.pending[echo]
	if ok {
		delete(m.pending, echo)
	}
	m.mu.Unlock()
	if !ok {
		return false
	}
	select {
	case ch <- response:
	default:
	}
	return true
}
func (m *qqGovernanceOneBotConnectionManager) CallOneBot(ctx context.Context, action string, params map[string]any) (json.RawMessage, error) {
	echo := uuid.NewString()
	ch := make(chan oneBotActionResponse, 1)
	m.mu.Lock()
	connection := m.current
	if connection == nil {
		m.mu.Unlock()
		return nil, &service.OneBotActionError{Message: "OneBot 机器人未连接", Retryable: true}
	}
	m.pending[echo] = ch
	m.mu.Unlock()
	defer func() { m.mu.Lock(); delete(m.pending, echo); m.mu.Unlock() }()
	connection.writeMu.Lock()
	err := websocket.JSON.Send(connection.conn, map[string]any{"action": action, "params": params, "echo": echo})
	connection.writeMu.Unlock()
	if err != nil {
		return nil, &service.OneBotActionError{Message: "发送 OneBot 动作失败: " + err.Error(), Retryable: true}
	}
	select {
	case <-ctx.Done():
		return nil, &service.OneBotActionError{Message: "等待 OneBot 动作响应超时", Retryable: true}
	case resp := <-ch:
		if resp.Status == "ok" || resp.RetCode == 0 || (action == "set_group_add_request" && strings.Contains(resp.Wording, "已处理")) {
			return resp.Data, nil
		}
		return nil, &service.OneBotActionError{Message: fmt.Sprintf("OneBot 动作失败（retcode=%d）：%s", resp.RetCode, resp.Wording), Retryable: retryableOneBotResponse(resp)}
	}
}
func retryableOneBotResponse(response oneBotActionResponse) bool {
	wording := strings.ToLower(response.Wording)
	for _, marker := range []string{"权限", "参数", "群不存在", "成员不存在", "已离群", "名片", "permission", "invalid", "not found", "not in group"} {
		if strings.Contains(wording, marker) {
			return false
		}
	}
	return true
}
func (m *qqGovernanceOneBotConnectionManager) logger() *zap.Logger {
	return oneBotLogger()
}

func oneBotLogger() *zap.Logger {
	if l := global.CurrentLogger(); l != nil {
		return l
	}
	return zap.NewNop()
}

func parseOneBotGovernanceEvent(raw []byte) (service.QQGovernanceInboundEvent, bool) {
	var v struct {
		PostType    string `json:"post_type"`
		RequestType string `json:"request_type"`
		NoticeType  string `json:"notice_type"`
		SubType     string `json:"sub_type"`
		GroupID     int64  `json:"group_id"`
		UserID      int64  `json:"user_id"`
		SelfID      int64  `json:"self_id"`
		Time        int64  `json:"time"`
		Flag        string `json:"flag"`
	}
	if json.Unmarshal(raw, &v) != nil || v.GroupID <= 0 || v.UserID <= 0 {
		return service.QQGovernanceInboundEvent{}, false
	}
	kind := ""
	if v.PostType == "request" && v.RequestType == "group" && v.SubType == "add" {
		kind = "request/group_add"
	} else if v.PostType == "notice" && v.NoticeType == "group_increase" {
		kind = "notice/group_increase"
	} else if v.PostType == "notice" && v.NoticeType == "group_decrease" {
		kind = "notice/group_decrease"
	}
	if kind == "" {
		return service.QQGovernanceInboundEvent{}, false
	}
	unique := v.Flag
	if unique == "" {
		unique = fmt.Sprintf("%d:%d", v.SelfID, v.Time)
	}
	return service.QQGovernanceInboundEvent{EventKey: fmt.Sprintf("%s:%d:%d:%s", kind, v.GroupID, v.UserID, unique), EventType: kind, GroupID: v.GroupID, QQ: v.UserID, RequestFlag: v.Flag}, true
}
