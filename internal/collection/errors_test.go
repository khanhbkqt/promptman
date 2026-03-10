package collection

import (
	"testing"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestDomainError_Error(t *testing.T) {
	err := &DomainError{Code: "TEST_CODE", Message: "something went wrong"}
	want := "TEST_CODE: something went wrong"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestDomainError_Wrap(t *testing.T) {
	original := &DomainError{Code: "ORIG_CODE", Message: "original"}
	wrapped := original.Wrap("new message")

	if wrapped.Code != original.Code {
		t.Errorf("Wrap().Code = %q, want %q", wrapped.Code, original.Code)
	}
	if wrapped.Message != "new message" {
		t.Errorf("Wrap().Message = %q, want %q", wrapped.Message, "new message")
	}
	// Ensure it's a new pointer, not mutating original
	if wrapped == original {
		t.Error("Wrap() should return a new DomainError, not the same pointer")
	}
	if original.Message != "original" {
		t.Error("Wrap() mutated the original error's message")
	}
}

func TestDomainError_Wrapf(t *testing.T) {
	original := &DomainError{Code: "ORIG_CODE", Message: "original"}
	wrapped := original.Wrapf("id %q not found", "abc-123")

	if wrapped.Code != original.Code {
		t.Errorf("Wrapf().Code = %q, want %q", wrapped.Code, original.Code)
	}
	want := `id "abc-123" not found`
	if wrapped.Message != want {
		t.Errorf("Wrapf().Message = %q, want %q", wrapped.Message, want)
	}
}

func TestIsDomainError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
		want bool
	}{
		{
			name: "matching code",
			err:  ErrCollectionNotFound,
			code: envelope.CodeCollectionNotFound,
			want: true,
		},
		{
			name: "wrong code",
			err:  ErrCollectionNotFound,
			code: envelope.CodeInvalidYAML,
			want: false,
		},
		{
			name: "wrapped domain error matches",
			err:  ErrInvalidYAML.Wrapf("bad file %q", "x.yaml"),
			code: envelope.CodeInvalidYAML,
			want: true,
		},
		{
			name: "nil error",
			err:  nil,
			code: envelope.CodeCollectionNotFound,
			want: false,
		},
		{
			name: "non-domain error",
			err:  &ValidationError{Errors: []string{"oops"}},
			code: envelope.CodeInvalidRequest,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDomainError(tt.err, tt.code); got != tt.want {
				t.Errorf("IsDomainError(%v, %q) = %v, want %v", tt.err, tt.code, got, tt.want)
			}
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	// Verify sentinel errors have expected codes
	tests := []struct {
		name string
		err  *DomainError
		code string
	}{
		{"ErrCollectionNotFound", ErrCollectionNotFound, envelope.CodeCollectionNotFound},
		{"ErrInvalidYAML", ErrInvalidYAML, envelope.CodeInvalidYAML},
		{"ErrInvalidRequest", ErrInvalidRequest, envelope.CodeInvalidRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("%s.Code = %q, want %q", tt.name, tt.err.Code, tt.code)
			}
		})
	}
}
