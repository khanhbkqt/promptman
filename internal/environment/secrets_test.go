package environment

import (
	"testing"
)

func TestResolveSecrets_SetEnvVar(t *testing.T) {
	t.Setenv("TEST_API_KEY", "sk-12345")
	t.Setenv("TEST_DB_PASS", "mypassword")

	secrets := map[string]string{
		"apiKey":     "$ENV{TEST_API_KEY}",
		"dbPassword": "$ENV{TEST_DB_PASS}",
	}

	resolved, err := ResolveSecrets(secrets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved["apiKey"] != "sk-12345" {
		t.Errorf("apiKey: got %q, want %q", resolved["apiKey"], "sk-12345")
	}
	if resolved["dbPassword"] != "mypassword" {
		t.Errorf("dbPassword: got %q, want %q", resolved["dbPassword"], "mypassword")
	}
}

func TestResolveSecrets_UnsetEnvVar(t *testing.T) {
	secrets := map[string]string{
		"apiKey": "$ENV{NONEXISTENT_VAR_12345}",
	}

	_, err := ResolveSecrets(secrets)
	if err == nil {
		t.Fatal("expected error for unset env var, got nil")
	}

	de, ok := err.(*DomainError)
	if !ok {
		t.Fatalf("expected *DomainError, got %T", err)
	}
	if de.Code != ErrSecretResolveFailed.Code {
		t.Errorf("error code: got %q, want %q", de.Code, ErrSecretResolveFailed.Code)
	}
}

func TestResolveSecrets_MixedValues(t *testing.T) {
	t.Setenv("TEST_SECRET_VAL", "resolved-value")

	secrets := map[string]string{
		"fromEnv":     "$ENV{TEST_SECRET_VAL}",
		"plainText":   "not-a-reference",
		"emptyString": "",
	}

	resolved, err := ResolveSecrets(secrets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved["fromEnv"] != "resolved-value" {
		t.Errorf("fromEnv: got %q, want %q", resolved["fromEnv"], "resolved-value")
	}
	if resolved["plainText"] != "not-a-reference" {
		t.Errorf("plainText: got %q, want %q", resolved["plainText"], "not-a-reference")
	}
	if resolved["emptyString"] != "" {
		t.Errorf("emptyString: got %q, want %q", resolved["emptyString"], "")
	}
}

func TestResolveSecrets_EmptyMap(t *testing.T) {
	resolved, err := ResolveSecrets(map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("expected empty map, got %d entries", len(resolved))
	}
}

func TestResolveSecrets_NilMap(t *testing.T) {
	resolved, err := ResolveSecrets(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != nil {
		t.Errorf("expected nil, got %v", resolved)
	}
}

func TestResolveSecrets_EmptyEnvValue(t *testing.T) {
	// An env var that IS set but has an empty value should resolve successfully.
	t.Setenv("TEST_EMPTY_VAR", "")

	secrets := map[string]string{
		"emptySecret": "$ENV{TEST_EMPTY_VAR}",
	}

	resolved, err := ResolveSecrets(secrets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved["emptySecret"] != "" {
		t.Errorf("emptySecret: got %q, want empty string", resolved["emptySecret"])
	}
}

func TestResolveSecrets_InvalidSyntax(t *testing.T) {
	// Values that look like $ENV{} but don't match the regex should pass through.
	tests := []struct {
		name  string
		value string
	}{
		{"lowercase var", "$ENV{lowercase}"},
		{"partial match", "$ENV{PARTIAL"},
		{"no braces", "$ENV_NO_BRACES"},
		{"embedded in string", "prefix$ENV{VAR}suffix"},
		{"empty braces", "$ENV{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets := map[string]string{"key": tt.value}
			resolved, err := ResolveSecrets(secrets)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolved["key"] != tt.value {
				t.Errorf("got %q, want %q (pass-through)", resolved["key"], tt.value)
			}
		})
	}
}

func TestMaskSecrets_Basic(t *testing.T) {
	secrets := map[string]string{
		"apiKey":     "sk-12345",
		"dbPassword": "mypassword",
	}

	masked := MaskSecrets(secrets)
	for key, val := range masked {
		if val != "***" {
			t.Errorf("key %q: got %q, want %q", key, val, "***")
		}
	}
	if len(masked) != len(secrets) {
		t.Errorf("masked map length: got %d, want %d", len(masked), len(secrets))
	}
}

func TestMaskSecrets_EmptyMap(t *testing.T) {
	masked := MaskSecrets(map[string]string{})
	if len(masked) != 0 {
		t.Errorf("expected empty map, got %d entries", len(masked))
	}
}

func TestMaskSecrets_NilMap(t *testing.T) {
	masked := MaskSecrets(nil)
	if masked != nil {
		t.Errorf("expected nil, got %v", masked)
	}
}

func TestMaskSecrets_DoesNotModifyOriginal(t *testing.T) {
	secrets := map[string]string{
		"key": "secret-value",
	}

	_ = MaskSecrets(secrets)

	if secrets["key"] != "secret-value" {
		t.Errorf("original map was modified: got %q", secrets["key"])
	}
}
