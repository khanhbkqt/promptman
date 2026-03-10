package testing

import (
	"errors"
	"testing"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestDomainError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *DomainError
		expected string
	}{
		{"sandbox violation", ErrSandboxViolation, "SANDBOX_VIOLATION: sandbox violation"},
		{"test timeout", ErrTestTimeout, "TEST_TIMEOUT: test timeout"},
		{"script parse", ErrScriptParse, "SCRIPT_PARSE: script parse error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDomainError_Wrap(t *testing.T) {
	wrapped := ErrSandboxViolation.Wrap("require is not defined in sandbox")

	if wrapped.Code != envelope.CodeSandboxViolation {
		t.Errorf("Code = %q, want %q", wrapped.Code, envelope.CodeSandboxViolation)
	}
	if wrapped.Message != "require is not defined in sandbox" {
		t.Errorf("Message = %q, want %q", wrapped.Message, "require is not defined in sandbox")
	}
	// Original sentinel should be unchanged
	if ErrSandboxViolation.Message != "sandbox violation" {
		t.Error("original sentinel was mutated")
	}
}

func TestDomainError_Wrapf(t *testing.T) {
	wrapped := ErrTestTimeout.Wrapf("test %q exceeded %dms limit", "auth check", 10000)

	if wrapped.Code != envelope.CodeTestTimeout {
		t.Errorf("Code = %q, want %q", wrapped.Code, envelope.CodeTestTimeout)
	}
	expected := `test "auth check" exceeded 10000ms limit`
	if wrapped.Message != expected {
		t.Errorf("Message = %q, want %q", wrapped.Message, expected)
	}
}

func TestIsDomainError_MatchingCode(t *testing.T) {
	if !IsDomainError(ErrSandboxViolation, envelope.CodeSandboxViolation) {
		t.Error("expected IsDomainError to return true for matching code")
	}
}

func TestIsDomainError_DifferentCode(t *testing.T) {
	if IsDomainError(ErrSandboxViolation, envelope.CodeTestTimeout) {
		t.Error("expected IsDomainError to return false for different code")
	}
}

func TestIsDomainError_NonDomainError(t *testing.T) {
	err := errors.New("some other error")
	if IsDomainError(err, envelope.CodeSandboxViolation) {
		t.Error("expected IsDomainError to return false for non-DomainError")
	}
}

func TestIsDomainError_Nil(t *testing.T) {
	if IsDomainError(nil, envelope.CodeSandboxViolation) {
		t.Error("expected IsDomainError to return false for nil error")
	}
}
