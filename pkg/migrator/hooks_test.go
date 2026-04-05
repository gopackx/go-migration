package migrator

import (
	"errors"
	"testing"
	"time"
)

func TestNewHookManager(t *testing.T) {
	hm := NewHookManager()
	if hm == nil {
		t.Fatalf("expected non-nil")
	}
	if len(hm.beforeHooks) != 0 {
		t.Errorf("expected empty, got %v", hm.beforeHooks)
	}
	if len(hm.afterHooks) != 0 {
		t.Errorf("expected empty, got %v", hm.afterHooks)
	}
}

// --- RegisterBefore and RunBefore ---

func TestRunBefore_SingleHook(t *testing.T) {
	hm := NewHookManager()

	var calledName, calledDir string
	hm.RegisterBefore(func(name, direction string) error {
		calledName = name
		calledDir = direction
		return nil
	})

	err := hm.RunBefore("20240101000000_create_users", "up")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "20240101000000_create_users" != calledName {
		t.Errorf("got %v, want %v", calledName, "20240101000000_create_users")
	}
	if "up" != calledDir {
		t.Errorf("got %v, want %v", calledDir, "up")
	}
}

func TestRunBefore_ErrorAbortsExecution(t *testing.T) {
	hm := NewHookManager()
	hookErr := errors.New("before hook failed")

	hm.RegisterBefore(func(name, direction string) error {
		return hookErr
	})

	secondCalled := false
	hm.RegisterBefore(func(name, direction string) error {
		secondCalled = true
		return nil
	})

	err := hm.RunBefore("20240101000000_create_users", "up")
	if !errors.Is(err, hookErr) {
		t.Errorf("got %v, want %v", err, hookErr)
	}
	if secondCalled {
		t.Errorf("expected false")
	}
}

func TestRunBefore_MultipleHooksRunInOrder(t *testing.T) {
	hm := NewHookManager()
	var order []int

	hm.RegisterBefore(func(name, direction string) error {
		order = append(order, 1)
		return nil
	})
	hm.RegisterBefore(func(name, direction string) error {
		order = append(order, 2)
		return nil
	})
	hm.RegisterBefore(func(name, direction string) error {
		order = append(order, 3)
		return nil
	})

	err := hm.RunBefore("20240101000000_test", "down")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 2, 3}
	if len(order) != len(expected) {
		t.Errorf("len = %d, want %d", len(order), len(expected))
	}
	for i := range expected {
		if i < len(order) && order[i] != expected[i] {
			t.Errorf("order[%d] = %d, want %d", i, order[i], expected[i])
		}
	}
}

func TestRunBefore_NoHooksReturnsNil(t *testing.T) {
	hm := NewHookManager()
	err := hm.RunBefore("20240101000000_test", "up")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- RegisterAfter and RunAfter ---

func TestRunAfter_SingleHook(t *testing.T) {
	hm := NewHookManager()

	var calledName, calledDir string
	var calledDuration time.Duration
	hm.RegisterAfter(func(name, direction string, duration time.Duration) error {
		calledName = name
		calledDir = direction
		calledDuration = duration
		return nil
	})

	dur := 150 * time.Millisecond
	err := hm.RunAfter("20240101000000_create_users", "up", dur)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "20240101000000_create_users" != calledName {
		t.Errorf("got %v, want %v", calledName, "20240101000000_create_users")
	}
	if "up" != calledDir {
		t.Errorf("got %v, want %v", calledDir, "up")
	}
	if dur != calledDuration {
		t.Errorf("got %v, want %v", calledDuration, dur)
	}
}

func TestRunAfter_ErrorDoesNotPropagate(t *testing.T) {
	hm := NewHookManager()

	hm.RegisterAfter(func(name, direction string, duration time.Duration) error {
		return errors.New("after hook failed")
	})

	err := hm.RunAfter("20240101000000_create_users", "down", time.Second)
	if err != nil {
		t.Errorf("RunAfter should return nil even when hooks error")
	}
}

func TestRunAfter_MultipleHooksRunInOrder(t *testing.T) {
	hm := NewHookManager()
	var order []int

	hm.RegisterAfter(func(name, direction string, duration time.Duration) error {
		order = append(order, 1)
		return nil
	})
	hm.RegisterAfter(func(name, direction string, duration time.Duration) error {
		order = append(order, 2)
		return nil
	})
	hm.RegisterAfter(func(name, direction string, duration time.Duration) error {
		order = append(order, 3)
		return nil
	})

	err := hm.RunAfter("20240101000000_test", "up", time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedAfter := []int{1, 2, 3}
	if len(order) != len(expectedAfter) {
		t.Errorf("len = %d, want %d", len(order), len(expectedAfter))
	}
	for i := range expectedAfter {
		if i < len(order) && order[i] != expectedAfter[i] {
			t.Errorf("order[%d] = %d, want %d", i, order[i], expectedAfter[i])
		}
	}
}

func TestRunAfter_ErrorDoesNotStopSubsequentHooks(t *testing.T) {
	hm := NewHookManager()
	var order []int

	hm.RegisterAfter(func(name, direction string, duration time.Duration) error {
		order = append(order, 1)
		return errors.New("hook 1 error")
	})
	hm.RegisterAfter(func(name, direction string, duration time.Duration) error {
		order = append(order, 2)
		return nil
	})
	hm.RegisterAfter(func(name, direction string, duration time.Duration) error {
		order = append(order, 3)
		return errors.New("hook 3 error")
	})

	err := hm.RunAfter("20240101000000_test", "down", time.Second)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expectedAll := []int{1, 2, 3}
	if len(order) != len(expectedAll) {
		t.Errorf("len = %d, want %d", len(order), len(expectedAll))
	}
	for i := range expectedAll {
		if i < len(order) && order[i] != expectedAll[i] {
			t.Errorf("order[%d] = %d, want %d", i, order[i], expectedAll[i])
		}
	}
}

func TestRunAfter_NoHooksReturnsNil(t *testing.T) {
	hm := NewHookManager()
	err := hm.RunAfter("20240101000000_test", "up", time.Second)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Direction variants ---

func TestHooks_DownDirection(t *testing.T) {
	hm := NewHookManager()

	var beforeDir, afterDir string
	hm.RegisterBefore(func(name, direction string) error {
		beforeDir = direction
		return nil
	})
	hm.RegisterAfter(func(name, direction string, duration time.Duration) error {
		afterDir = direction
		return nil
	})

	err := hm.RunBefore("20240101000000_test", "down")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "down" != beforeDir {
		t.Errorf("got %v, want %v", beforeDir, "down")
	}

	err = hm.RunAfter("20240101000000_test", "down", time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if "down" != afterDir {
		t.Errorf("got %v, want %v", afterDir, "down")
	}
}
