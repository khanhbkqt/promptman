package sandbox

import (
	"strings"
	stdtesting "testing"

	"github.com/khanhnguyen/promptman/internal/testing"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestNew(t *stdtesting.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.vm == nil {
		t.Fatal("VM is nil")
	}
}

func TestExecute_BasicJS(t *stdtesting.T) {
	tests := []struct {
		name     string
		script   string
		expected any
	}{
		{"variable", "var x = 42; x", int64(42)},
		{"string", "'hello'", "hello"},
		{"function", "function add(a,b){return a+b}; add(2,3)", int64(5)},
		{"loop", "var s=0; for(var i=0;i<5;i++){s+=i}; s", int64(10)},
		{"boolean", "true", true},
		{"null", "null", nil},
		{"array length", "[1,2,3].length", int64(3)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			s := New()
			v, err := s.Execute(tt.script)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := v.Export()
			if got != tt.expected {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.expected, tt.expected)
			}
		})
	}
}

func TestExecute_BlockedFunctions(t *stdtesting.T) {
	blocked := []struct {
		name   string
		script string
		errMsg string
	}{
		{"require", "require('fs')", "require is not defined in sandbox"},
		{"eval", "eval('1+1')", "eval is not defined in sandbox"},
		{"setTimeout", "setTimeout(function(){}, 100)", "setTimeout is not defined in sandbox"},
		{"setInterval", "setInterval(function(){}, 100)", "setInterval is not defined in sandbox"},
		{"setImmediate", "setImmediate(function(){})", "setImmediate is not defined in sandbox"},
		{"clearTimeout", "clearTimeout(1)", "clearTimeout is not defined in sandbox"},
		{"clearInterval", "clearInterval(1)", "clearInterval is not defined in sandbox"},
		{"clearImmediate", "clearImmediate(1)", "clearImmediate is not defined in sandbox"},
	}

	for _, tt := range blocked {
		t.Run(tt.name, func(t *stdtesting.T) {
			s := New()
			_, err := s.Execute(tt.script)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testing.IsDomainError(err, envelope.CodeSandboxViolation) {
				t.Errorf("expected ErrSandboxViolation, got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("error message %q does not contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestExecute_BlockedValues(t *stdtesting.T) {
	// Blocked values are set to marker strings, not usable as real objects.
	values := []struct {
		name   string
		script string
	}{
		{"process.exit", "process.exit()"},
		{"process access", "'' + process"},
		{"__dirname concat", "__dirname + '/file'"},
		{"__filename concat", "__filename + '.js'"},
	}

	for _, tt := range values {
		t.Run(tt.name, func(t *stdtesting.T) {
			s := New()
			v, err := s.Execute(tt.script)
			if err != nil {
				// process.exit() will fail because string has no method exit
				if !strings.Contains(err.Error(), "not defined in sandbox") &&
					!strings.Contains(err.Error(), "has no member") {
					t.Errorf("unexpected error type: %v", err)
				}
				return
			}
			// For concatenation, the result contains the marker string.
			result := v.String()
			if !strings.Contains(result, "is not defined in sandbox") {
				t.Errorf("expected marker string, got: %q", result)
			}
		})
	}
}

func TestExecute_AllowedAPIs(t *stdtesting.T) {
	allowed := []struct {
		name   string
		script string
	}{
		{"JSON.parse", `JSON.parse('{"a":1}').a`},
		{"JSON.stringify", `JSON.stringify({a:1})`},
		{"Date", "new Date().getTime() > 0"},
		{"Math", "Math.floor(3.7)"},
		{"RegExp", "/abc/.test('xabcy')"},
	}

	for _, tt := range allowed {
		t.Run(tt.name, func(t *stdtesting.T) {
			s := New()
			_, err := s.Execute(tt.script)
			if err != nil {
				t.Fatalf("allowed API %q should work, got error: %v", tt.name, err)
			}
		})
	}
}

func TestExecute_ScriptParseError(t *stdtesting.T) {
	scripts := []struct {
		name   string
		script string
	}{
		{"unclosed brace", "function f() {"},
		{"invalid syntax", "var 123abc = 1;"},
		{"unclosed string", "'hello"},
	}

	for _, tt := range scripts {
		t.Run(tt.name, func(t *stdtesting.T) {
			s := New()
			_, err := s.Execute(tt.script)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testing.IsDomainError(err, envelope.CodeScriptParse) {
				t.Errorf("expected ErrScriptParse, got: %v", err)
			}
		})
	}
}

func TestConsole_Empty(t *stdtesting.T) {
	s := New()
	if len(s.Console()) != 0 {
		t.Fatal("expected empty console initially")
	}
}

func TestReset(t *stdtesting.T) {
	s := New()
	s.console = []string{"something"}
	s.Reset()
	if len(s.Console()) != 0 {
		t.Fatal("expected empty console after reset")
	}
}
