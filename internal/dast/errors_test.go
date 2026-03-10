package dast

import (
	"testing"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestDomainError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *DomainError
		want string
	}{
		{"invalid rule", ErrInvalidRule, "INVALID_RULE: invalid rule"},
		{"profile not found", ErrProfileNotFound, "PROFILE_NOT_FOUND: profile not found"},
		{"rule load failed", ErrRuleLoadFailed, "RULE_LOAD_FAILED: rule load failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("DomainError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDomainError_Wrap(t *testing.T) {
	wrapped := ErrInvalidRule.Wrap("rule 'bad-rule' missing severity field")

	if wrapped.Code != envelope.CodeInvalidRule {
		t.Errorf("Code = %q, want %q", wrapped.Code, envelope.CodeInvalidRule)
	}
	if wrapped.Message != "rule 'bad-rule' missing severity field" {
		t.Errorf("Message = %q, want %q", wrapped.Message, "rule 'bad-rule' missing severity field")
	}
	// Original should be unchanged
	if ErrInvalidRule.Message != "invalid rule" {
		t.Errorf("Original modified: Message = %q", ErrInvalidRule.Message)
	}
}

func TestDomainError_Wrapf(t *testing.T) {
	wrapped := ErrProfileNotFound.Wrapf("profile %q not found", "ultra")

	if wrapped.Code != envelope.CodeProfileNotFound {
		t.Errorf("Code = %q, want %q", wrapped.Code, envelope.CodeProfileNotFound)
	}
	expected := `profile "ultra" not found`
	if wrapped.Message != expected {
		t.Errorf("Message = %q, want %q", wrapped.Message, expected)
	}
}

func TestIsDomainError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
		want bool
	}{
		{"matching code", ErrInvalidRule, envelope.CodeInvalidRule, true},
		{"wrong code", ErrInvalidRule, envelope.CodeProfileNotFound, false},
		{"wrapped error", ErrRuleLoadFailed.Wrap("disk full"), envelope.CodeRuleLoadFailed, true},
		{"nil error", nil, envelope.CodeInvalidRule, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDomainError(tt.err, tt.code); got != tt.want {
				t.Errorf("IsDomainError() = %v, want %v", got, tt.want)
			}
		})
	}
}
