package sandbox

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	testing "github.com/khanhnguyen/promptman/internal/testing"
)

// Sandbox wraps a goja JavaScript runtime with security constraints
// and a curated standard library. Dangerous APIs are blocked, while
// safe utilities (atob/btoa, crypto, console capture) are injected.
type Sandbox struct {
	vm      *goja.Runtime
	console []string // captured console output
}

// New creates a Sandbox with a locked-down goja VM and injected globals.
func New() *Sandbox {
	vm := goja.New()
	s := &Sandbox{vm: vm}
	s.lockdown()
	s.injectGlobals()
	return s
}

// Execute runs the given JavaScript source and returns the result.
// Returns ErrScriptParse for syntax errors or ErrSandboxViolation
// if a blocked API is invoked.
func (s *Sandbox) Execute(script string) (goja.Value, error) {
	v, err := s.vm.RunString(script)
	if err != nil {
		return nil, classifyError(err)
	}
	return v, nil
}

// Console returns all captured console output (log, warn, error)
// accumulated since creation or last Reset.
func (s *Sandbox) Console() []string {
	return s.console
}

// Reset clears the captured console output for reuse.
func (s *Sandbox) Reset() {
	s.console = nil
}

// sandboxViolationPrefix is the marker used to identify blocked-API errors.
const sandboxViolationPrefix = "sandbox:"

// classifyError converts a goja error into the appropriate DomainError.
func classifyError(err error) error {
	msg := err.Error()

	// Check for sandbox violation markers in the error message.
	if strings.Contains(msg, sandboxViolationPrefix) {
		// Extract the human-readable part after the prefix.
		idx := strings.Index(msg, sandboxViolationPrefix)
		detail := strings.TrimSpace(msg[idx+len(sandboxViolationPrefix):])
		// Trim any trailing goja stack trace info.
		if nl := strings.IndexByte(detail, '\n'); nl > 0 {
			detail = detail[:nl]
		}
		return testing.ErrSandboxViolation.Wrap(detail)
	}

	// Check for syntax errors.
	var syntaxErr *goja.CompilerSyntaxError
	if errors.As(err, &syntaxErr) {
		return testing.ErrScriptParse.Wrapf("syntax error: %s", syntaxErr.Error())
	}
	// Also catch SyntaxError strings from the runtime.
	if strings.Contains(msg, "SyntaxError") {
		return testing.ErrScriptParse.Wrapf("syntax error: %s", msg)
	}

	return fmt.Errorf("script execution error: %w", err)
}

// blockedFunctions lists names overridden with throwing stubs.
var blockedFunctions = []string{
	"require",
	"eval",
	"setTimeout",
	"setInterval",
	"setImmediate",
	"clearTimeout",
	"clearInterval",
	"clearImmediate",
}

// blockedValues lists names set to throwing proxy objects.
var blockedValues = []string{
	"process",
	"__dirname",
	"__filename",
}

// lockdown removes or overrides dangerous globals in the VM.
func (s *Sandbox) lockdown() {
	// Override callable blocked APIs with throwing functions.
	for _, name := range blockedFunctions {
		errMsg := fmt.Sprintf("%s %s is not defined in sandbox", sandboxViolationPrefix, name)
		_ = s.vm.Set(name, func(call goja.FunctionCall) goja.Value {
			panic(s.vm.ToValue(errMsg))
		})
	}

	// Override non-callable blocked APIs (accessed as values).
	for _, name := range blockedValues {
		errMsg := fmt.Sprintf("%s %s is not defined in sandbox", sandboxViolationPrefix, name)
		_ = s.vm.Set(name, s.vm.ToValue(errMsg))
	}
}
