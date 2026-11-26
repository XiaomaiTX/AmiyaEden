package handler

import (
	"errors"
	"fmt"
	"testing"

	"amiya-eden/internal/service"
)

func TestFittingSaveErrorMessage(t *testing.T) {
	t.Run("keeps rebinding instruction", func(t *testing.T) {
		err := fmt.Errorf("token refresh failed: %w", service.ErrFittingTokenUnavailable)
		if got := fittingSaveErrorMessage(err); got != "人物 Token 不可用，请重新绑定" {
			t.Fatalf("unexpected rebinding message: %q", got)
		}
	})
	t.Run("hides upstream error", func(t *testing.T) {
		err := errors.New("upstream ESI diagnostic details")
		if got := fittingSaveErrorMessage(err); got != "保存装配失败，请检查装配数据和人物授权后重试" {
			t.Fatalf("unexpected public error message: %q", got)
		}
	})
}
