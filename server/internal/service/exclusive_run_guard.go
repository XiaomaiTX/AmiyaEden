package service

import (
	"fmt"
	"sync/atomic"
)

type exclusiveRunGuard struct {
	running atomic.Bool
}

func (g *exclusiveRunGuard) Start(action string) error {
	if g.running.CompareAndSwap(false, true) {
		return nil
	}
	return fmt.Errorf("%s正在执行，请稍后重试", action)
}

func (g *exclusiveRunGuard) Finish() {
	g.running.Store(false)
}
