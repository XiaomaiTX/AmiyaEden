package taskregistry

import (
	"context"
	"fmt"
	"sync"
)

type TaskCategory string

const (
	TaskCategoryESI       TaskCategory = "esi"
	TaskCategoryOperation TaskCategory = "operation"
	TaskCategorySystem    TaskCategory = "system"
)

type TaskType string

const (
	TaskTypeRecurring TaskType = "recurring"
	TaskTypeTriggered TaskType = "triggered"
)

type TaskDefinition struct {
	Name        string
	Description string
	Category    TaskCategory
	Type        TaskType
	DefaultCron string
	RunFunc     func(ctx context.Context) error
}

type lockHandleContextKey struct{}

type LockHandle struct {
	registry *Registry
	taskName string
	token    uint64
	claimMu  sync.Mutex
	claimed  bool
	once     sync.Once
}

func (h *LockHandle) Claim() bool {
	if h == nil {
		return false
	}

	h.claimMu.Lock()
	defer h.claimMu.Unlock()

	if h.claimed {
		return false
	}
	h.claimed = true
	return true
}

func (h *LockHandle) Release() {
	if h == nil || h.registry == nil {
		return
	}

	h.once.Do(func() {
		h.registry.release(h.taskName, h.token)
	})
}

func ContextWithLockHandle(ctx context.Context, handle *LockHandle) context.Context {
	return context.WithValue(ctx, lockHandleContextKey{}, handle)
}

func LockHandleFromContext(ctx context.Context) *LockHandle {
	handle, _ := ctx.Value(lockHandleContextKey{}).(*LockHandle)
	return handle
}

type Registry struct {
	mu        sync.RWMutex
	tasks     map[string]TaskDefinition
	locks     map[string]*sync.Mutex
	held      map[string]uint64
	nextToken uint64
}

func New() *Registry {
	return &Registry{
		tasks: make(map[string]TaskDefinition),
		locks: make(map[string]*sync.Mutex),
		held:  make(map[string]uint64),
	}
}

func (r *Registry) Register(task TaskDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.Name]; exists {
		panic(fmt.Sprintf("task %q already registered", task.Name))
	}

	r.tasks[task.Name] = task
	r.locks[task.Name] = &sync.Mutex{}
}

func (r *Registry) Get(name string) (TaskDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[name]
	return task, ok
}

func (r *Registry) All() []TaskDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := make([]TaskDefinition, 0, len(r.tasks))
	for _, task := range r.tasks {
		all = append(all, task)
	}

	return all
}

func (r *Registry) TryLock(name string) (*LockHandle, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	lock, ok := r.locks[name]
	if !ok {
		return nil, false
	}

	if !lock.TryLock() {
		return nil, false
	}

	r.nextToken++
	token := r.nextToken
	r.held[name] = token

	return &LockHandle{registry: r, taskName: name, token: token}, true
}

func (r *Registry) HoldsLock(name string, handle *LockHandle) bool {
	if handle == nil || handle.registry != r || handle.taskName != name {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	token, ok := r.held[name]
	if !ok {
		return false
	}

	return token == handle.token
}

func (r *Registry) ClaimLock(name string, handle *LockHandle) bool {
	if handle == nil || handle.registry != r || handle.taskName != name {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	token, ok := r.held[name]
	if !ok || token != handle.token {
		return false
	}

	return handle.Claim()
}

func (r *Registry) release(name string, token uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	heldToken, ok := r.held[name]
	if !ok || heldToken != token {
		return
	}

	delete(r.held, name)
	if lock, exists := r.locks[name]; exists {
		lock.Unlock()
	}
}

func (r *Registry) IsRunning(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.held[name]
	return ok
}
