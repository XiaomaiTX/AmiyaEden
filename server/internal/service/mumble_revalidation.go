package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/repository"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const defaultMumbleRevalidateTimeout = time.Second

// NotifyMumbleIdentityChanged sends a best-effort invalidation signal. The
// Mumble server must pull the current claims from Seat instead of trusting this
// notification as an authorization fact.
func NotifyMumbleIdentityChanged(ctx context.Context, seatUserID uint) {
	if err := notifyMumbleIdentityChanged(ctx, seatUserID); err != nil && global.Logger != nil {
		global.Logger.Warn("[Mumble] 通知在线身份重校验失败", zap.Uint("user_id", seatUserID), zap.Error(err))
	}
}

func notifyMumbleIdentityChanged(ctx context.Context, seatUserID uint) error {
	if seatUserID == 0 {
		return nil
	}
	cfg := NewSysConfigService().GetMumbleConfig()
	serverURL, token := strings.TrimSpace(cfg.ServerURL), strings.TrimSpace(cfg.RevalidateToken)
	if serverURL == "" || token == "" {
		return nil
	}
	parsed, err := url.Parse(serverURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("mumble server_url 无效")
	}

	identity, err := repository.NewMumbleIdentityRepository().GetBySeatUserID(seatUserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询 Mumble 身份: %w", err)
	}
	payload, err := json.Marshal(struct {
		UserID uint32 `json:"user_id"`
	}{UserID: identity.StableMumbleUserID()})
	if err != nil {
		return fmt.Errorf("编码重校验请求: %w", err)
	}

	timeout := defaultMumbleRevalidateTimeout
	if cfg.RevalidateTimeoutMS > 0 {
		timeout = time.Duration(cfg.RevalidateTimeoutMS) * time.Millisecond
	}
	if ctx == nil {
		ctx = context.Background()
	}
	requestURL := strings.TrimRight(serverURL, "/") + "/internal/identity/v1/revalidate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("创建重校验请求: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return fmt.Errorf("发送重校验请求: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("重校验请求返回 HTTP %d", resp.StatusCode)
	}
	return nil
}
