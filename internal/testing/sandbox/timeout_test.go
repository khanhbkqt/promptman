package sandbox

import (
	"context"
	"runtime"
	stdtesting "testing"
	"time"

	"github.com/khanhnguyen/promptman/internal/testing"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestNewTimeoutManager_Defaults(t *stdtesting.T) {
	tm := NewTimeoutManager(0, 0)
	if tm.PerTest() != DefaultPerTestTimeout {
		t.Errorf("PerTest() = %v, want %v", tm.PerTest(), DefaultPerTestTimeout)
	}
	if tm.PerSuite() != DefaultPerSuiteTimeout {
		t.Errorf("PerSuite() = %v, want %v", tm.PerSuite(), DefaultPerSuiteTimeout)
	}
}

func TestNewTimeoutManager_CustomValues(t *stdtesting.T) {
	tm := NewTimeoutManager(5*time.Second, 30*time.Second)
	if tm.PerTest() != 5*time.Second {
		t.Errorf("PerTest() = %v, want 5s", tm.PerTest())
	}
	if tm.PerSuite() != 30*time.Second {
		t.Errorf("PerSuite() = %v, want 30s", tm.PerSuite())
	}
}

func TestNewTimeoutManager_NegativeUsesDefaults(t *stdtesting.T) {
	tm := NewTimeoutManager(-1, -1)
	if tm.PerTest() != DefaultPerTestTimeout {
		t.Errorf("PerTest() = %v, want default %v", tm.PerTest(), DefaultPerTestTimeout)
	}
	if tm.PerSuite() != DefaultPerSuiteTimeout {
		t.Errorf("PerSuite() = %v, want default %v", tm.PerSuite(), DefaultPerSuiteTimeout)
	}
}

func TestSuiteContext_CreatesDeadline(t *stdtesting.T) {
	tm := NewTimeoutManager(1*time.Second, 5*time.Second)
	ctx, cancel := tm.SuiteContext(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected suite context to have a deadline")
	}
	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > 6*time.Second {
		t.Errorf("unexpected remaining time: %v", remaining)
	}
}

func TestRunWithTimeout_NormalExecution(t *stdtesting.T) {
	s := New()
	tm := NewTimeoutManager(5*time.Second, 30*time.Second)
	ctx := context.Background()

	var result string
	err := tm.RunWithTimeout(ctx, s.VM(), func() error {
		v, err := s.VM().RunString(`"hello timeout"`)
		if err != nil {
			return err
		}
		result = v.String()
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello timeout" {
		t.Errorf("result = %q, want %q", result, "hello timeout")
	}
}

func TestRunWithTimeout_PerTestTimeout(t *stdtesting.T) {
	s := New()
	tm := NewTimeoutManager(50*time.Millisecond, 30*time.Second)
	ctx := context.Background()

	err := tm.RunWithTimeout(ctx, s.VM(), func() error {
		// Infinite loop that will be interrupted by timeout.
		_, err := s.VM().RunString(`while(true) {}`)
		return err
	})

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !testing.IsDomainError(err, envelope.CodeTestTimeout) {
		t.Errorf("expected ErrTestTimeout, got: %v", err)
	}
}

func TestRunWithTimeout_SuiteTimeoutAlreadyExpired(t *stdtesting.T) {
	s := New()
	tm := NewTimeoutManager(5*time.Second, 30*time.Second)

	// Create an already-cancelled context to simulate suite timeout.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := tm.RunWithTimeout(ctx, s.VM(), func() error {
		t.Fatal("function should not be called when suite is expired")
		return nil
	})

	if err == nil {
		t.Fatal("expected suite timeout error, got nil")
	}
	if !testing.IsDomainError(err, envelope.CodeTestTimeout) {
		t.Errorf("expected ErrTestTimeout, got: %v", err)
	}
}

func TestRunWithTimeout_SuiteTimeoutDuringExecution(t *stdtesting.T) {
	s := New()
	tm := NewTimeoutManager(5*time.Second, 50*time.Millisecond)

	// Suite timeout is 50ms — the infinite loop will be interrupted.
	ctx, cancel := tm.SuiteContext(context.Background())
	defer cancel()

	err := tm.RunWithTimeout(ctx, s.VM(), func() error {
		_, err := s.VM().RunString(`while(true) {}`)
		return err
	})

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !testing.IsDomainError(err, envelope.CodeTestTimeout) {
		t.Errorf("expected ErrTestTimeout, got: %v", err)
	}
}

func TestRunWithTimeout_VMReusableAfterTimeout(t *stdtesting.T) {
	s := New()
	tm := NewTimeoutManager(50*time.Millisecond, 30*time.Second)
	ctx := context.Background()

	// First: trigger a timeout.
	err := tm.RunWithTimeout(ctx, s.VM(), func() error {
		_, err := s.VM().RunString(`while(true) {}`)
		return err
	})
	if err == nil {
		t.Fatal("expected timeout error on first run")
	}

	// Second: the VM should still work after ClearInterrupt.
	v, err := s.Execute(`1 + 1`)
	if err != nil {
		t.Fatalf("VM not reusable after timeout: %v", err)
	}
	if v.Export() != int64(2) {
		t.Errorf("got %v, want 2", v.Export())
	}
}

func TestRunWithTimeout_NoGoroutineLeak(t *stdtesting.T) {
	s := New()
	tm := NewTimeoutManager(5*time.Second, 30*time.Second)
	ctx := context.Background()

	before := runtime.NumGoroutine()

	for i := 0; i < 10; i++ {
		_ = tm.RunWithTimeout(ctx, s.VM(), func() error {
			_, err := s.VM().RunString(`1 + 1`)
			return err
		})
	}

	// Give goroutines time to exit.
	time.Sleep(50 * time.Millisecond)
	after := runtime.NumGoroutine()

	// Allow small delta for runtime fluctuations.
	if after > before+2 {
		t.Errorf("goroutine leak: before=%d, after=%d", before, after)
	}
}

func TestExecuteWithTimeout_Success(t *stdtesting.T) {
	s := New()
	ctx := context.Background()

	v, err := s.ExecuteWithTimeout(ctx, `2 + 3`, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Export() != int64(5) {
		t.Errorf("got %v, want 5", v.Export())
	}
}

func TestExecuteWithTimeout_Timeout(t *stdtesting.T) {
	s := New()
	ctx := context.Background()

	_, err := s.ExecuteWithTimeout(ctx, `while(true) {}`, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !testing.IsDomainError(err, envelope.CodeTestTimeout) {
		t.Errorf("expected ErrTestTimeout, got: %v", err)
	}
}

func TestExecuteWithTimeout_SuiteContextCancelled(t *stdtesting.T) {
	s := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.ExecuteWithTimeout(ctx, `1 + 1`, 5*time.Second)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !testing.IsDomainError(err, envelope.CodeTestTimeout) {
		t.Errorf("expected ErrTestTimeout, got: %v", err)
	}
}

func TestExecuteWithTimeout_SandboxViolationNotMasked(t *stdtesting.T) {
	s := New()
	ctx := context.Background()

	_, err := s.ExecuteWithTimeout(ctx, `require('fs')`, 5*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should be sandbox violation, not timeout.
	if !testing.IsDomainError(err, envelope.CodeSandboxViolation) {
		t.Errorf("expected ErrSandboxViolation, got: %v", err)
	}
}

func TestExecuteWithTimeout_ScriptParseErrorNotMasked(t *stdtesting.T) {
	s := New()
	ctx := context.Background()

	_, err := s.ExecuteWithTimeout(ctx, `function f() {`, 5*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !testing.IsDomainError(err, envelope.CodeScriptParse) {
		t.Errorf("expected ErrScriptParse, got: %v", err)
	}
}
