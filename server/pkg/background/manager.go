package background

import (
	"context"
	"errors"
	"runtime/debug"
	"sync"
	"time"

	"go.uber.org/zap"
)

var nopLogger = zap.NewNop()

type Manager struct {
	ctx    context.Context
	cancel context.CancelFunc
	logger func() *zap.Logger

	mu           sync.Mutex
	shuttingDown bool
	waitGroup    sync.WaitGroup
}

func New(parent context.Context, logger func() *zap.Logger) *Manager {
	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)
	return &Manager{
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}
}

func (m *Manager) Context() context.Context {
	if m == nil || m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

func (m *Manager) Go(name string, run func(context.Context) error) bool {
	if m == nil || run == nil {
		return false
	}

	m.mu.Lock()
	if m.shuttingDown {
		m.mu.Unlock()
		return false
	}
	m.waitGroup.Add(1)
	m.mu.Unlock()

	go func() {
		defer m.waitGroup.Done()
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			m.loggerOrNop().Error("background task panicked",
				zap.String("task", name),
				zap.Any("panic", recovered),
				zap.ByteString("stack", debug.Stack()),
			)
		}()

		if err := run(m.Context()); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			m.loggerOrNop().Warn("background task failed", zap.String("task", name), zap.Error(err))
		}
	}()
	return true
}

func RunOrSchedule(ctx context.Context, manager *Manager, name string, run func(context.Context) error) error {
	if run == nil {
		return nil
	}
	if manager != nil && manager.Go(name, run) {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return run(ctx)
}

func (m *Manager) Shutdown(timeout time.Duration) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	if !m.shuttingDown {
		m.shuttingDown = true
	}
	m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
	}

	done := make(chan struct{})
	go func() {
		m.waitGroup.Wait()
		close(done)
	}()

	if timeout <= 0 {
		<-done
		return nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
		return nil
	case <-timer.C:
		return context.DeadlineExceeded
	}
}

func (m *Manager) loggerOrNop() *zap.Logger {
	if m != nil && m.logger != nil {
		if logger := m.logger(); logger != nil {
			return logger
		}
	}
	return nopLogger
}
