package testing

import (
	"fmt"

	"github.com/dop251/goja"
)

// hookNames are the recognized lifecycle hook keys in module.exports.
var hookNames = []string{"beforeAll", "afterAll", "beforeEach", "afterEach"}

// Hooks holds the lifecycle hook functions extracted from module.exports.
// Any field may be nil if the corresponding hook was not defined.
type Hooks struct {
	BeforeAll  goja.Callable // runs once before the suite
	AfterAll   goja.Callable // runs once after the suite
	BeforeEach goja.Callable // runs before each test
	AfterEach  goja.Callable // runs after each test
}

// ParsedScript is the result of evaluating a test script's module.exports.
// It contains lifecycle hooks and a map of request keys to test functions.
type ParsedScript struct {
	Hooks Hooks
	Tests map[string]goja.Callable // request key → test function
}

// TestKeys returns all test keys in the parsed script.
func (p *ParsedScript) TestKeys() []string {
	keys := make([]string, 0, len(p.Tests))
	for k := range p.Tests {
		keys = append(keys, k)
	}
	return keys
}

// isHookName reports whether name is a recognized lifecycle hook.
func isHookName(name string) bool {
	for _, h := range hookNames {
		if h == name {
			return true
		}
	}
	return false
}

// ParseExports extracts lifecycle hooks and test functions from a
// module.exports value. The exports value should be an object whose keys
// are either hook names (beforeAll, afterAll, beforeEach, afterEach) or
// request keys mapping to test functions.
//
// Non-callable values are silently skipped to handle metadata keys.
func ParseExports(vm *goja.Runtime, exports goja.Value) (*ParsedScript, error) {
	if exports == nil || goja.IsUndefined(exports) || goja.IsNull(exports) {
		return nil, ErrScriptParse.Wrap("module.exports is undefined or null")
	}

	obj := exports.ToObject(vm)
	if obj == nil {
		return nil, ErrScriptParse.Wrap("module.exports is not an object")
	}

	ps := &ParsedScript{
		Tests: make(map[string]goja.Callable),
	}

	for _, key := range obj.Keys() {
		val := obj.Get(key)
		fn, ok := goja.AssertFunction(val)
		if !ok {
			// Non-callable values (e.g., metadata) are silently skipped.
			continue
		}

		switch key {
		case "beforeAll":
			ps.Hooks.BeforeAll = fn
		case "afterAll":
			ps.Hooks.AfterAll = fn
		case "beforeEach":
			ps.Hooks.BeforeEach = fn
		case "afterEach":
			ps.Hooks.AfterEach = fn
		default:
			ps.Tests[key] = fn
		}
	}

	return ps, nil
}

// RunHook executes a lifecycle hook if it is non-nil.
// The hook receives the pm object as its first argument.
// Returns an error if the hook panics or returns an error.
func RunHook(vm *goja.Runtime, hook goja.Callable, pmObj goja.Value) error {
	if hook == nil {
		return nil
	}

	_, err := hook(goja.Undefined(), pmObj)
	if err != nil {
		return fmt.Errorf("hook execution failed: %w", err)
	}
	return nil
}
