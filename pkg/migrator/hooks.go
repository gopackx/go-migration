package migrator

import "time"

// HookFunc is a callback invoked before a migration executes.
// It receives the migration name and direction ("up" or "down").
type HookFunc func(name string, direction string) error

// AfterHookFunc is a callback invoked after a migration executes.
// It receives the migration name, direction, and execution duration.
type AfterHookFunc func(name string, direction string, duration time.Duration) error

// HookManager manages before and after migration hooks.
type HookManager struct {
	beforeHooks []HookFunc
	afterHooks  []AfterHookFunc
}

// NewHookManager creates a new HookManager with empty hook slices.
func NewHookManager() *HookManager {
	return &HookManager{
		beforeHooks: make([]HookFunc, 0),
		afterHooks:  make([]AfterHookFunc, 0),
	}
}

// RegisterBefore adds a before-migration hook. Hooks run in registration order.
func (h *HookManager) RegisterBefore(fn HookFunc) {
	h.beforeHooks = append(h.beforeHooks, fn)
}

// RegisterAfter adds an after-migration hook. Hooks run in registration order.
func (h *HookManager) RegisterAfter(fn AfterHookFunc) {
	h.afterHooks = append(h.afterHooks, fn)
}

// RunBefore invokes all registered before hooks in order.
// If any hook returns an error, execution stops and the error is returned (abort).
func (h *HookManager) RunBefore(name, direction string) error {
	for _, fn := range h.beforeHooks {
		if err := fn(name, direction); err != nil {
			return err
		}
	}
	return nil
}

// RunAfter invokes all registered after hooks in order.
// If any hook returns an error, it is ignored and execution continues.
// The caller (Migration_Runner) is responsible for logging hook errors.
func (h *HookManager) RunAfter(name, direction string, duration time.Duration) error {
	for _, fn := range h.afterHooks {
		_ = fn(name, direction, duration)
	}
	return nil
}
