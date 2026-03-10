package testing

import (
	"testing"

	"github.com/dop251/goja"
)

// newTestVM creates a goja runtime with module.exports pattern set up.
func newTestVM(t *testing.T, script string) (*goja.Runtime, goja.Value) {
	t.Helper()
	vm := goja.New()

	// Set up module.exports pattern.
	_ = vm.Set("module", vm.NewObject())
	_, err := vm.RunString("module.exports = {};")
	if err != nil {
		t.Fatalf("setup module.exports: %v", err)
	}

	// Run the test script.
	_, err = vm.RunString(script)
	if err != nil {
		t.Fatalf("run script: %v", err)
	}

	// Get module.exports.
	exports := vm.Get("module").ToObject(vm).Get("exports")
	return vm, exports
}

func TestParseExports_AllHooksAndTests(t *testing.T) {
	script := `
		module.exports = {
			beforeAll: function(pm) { },
			afterAll: function(pm) { },
			beforeEach: function(pm) { },
			afterEach: function(pm) { },
			"users/list": function(pm) { },
			"admin/get": function(pm) { },
		};
	`

	vm, exports := newTestVM(t, script)
	ps, err := ParseExports(vm, exports)
	if err != nil {
		t.Fatalf("ParseExports error: %v", err)
	}

	if ps.Hooks.BeforeAll == nil {
		t.Error("expected beforeAll hook to be set")
	}
	if ps.Hooks.AfterAll == nil {
		t.Error("expected afterAll hook to be set")
	}
	if ps.Hooks.BeforeEach == nil {
		t.Error("expected beforeEach hook to be set")
	}
	if ps.Hooks.AfterEach == nil {
		t.Error("expected afterEach hook to be set")
	}

	if len(ps.Tests) != 2 {
		t.Fatalf("expected 2 tests, got %d", len(ps.Tests))
	}
	if ps.Tests["users/list"] == nil {
		t.Error("expected users/list test to be set")
	}
	if ps.Tests["admin/get"] == nil {
		t.Error("expected admin/get test to be set")
	}
}

func TestParseExports_NoHooks(t *testing.T) {
	script := `
		module.exports = {
			"users/list": function(pm) { },
			"health": function(pm) { },
		};
	`

	vm, exports := newTestVM(t, script)
	ps, err := ParseExports(vm, exports)
	if err != nil {
		t.Fatalf("ParseExports error: %v", err)
	}

	if ps.Hooks.BeforeAll != nil {
		t.Error("expected beforeAll hook to be nil")
	}
	if ps.Hooks.AfterAll != nil {
		t.Error("expected afterAll hook to be nil")
	}
	if ps.Hooks.BeforeEach != nil {
		t.Error("expected beforeEach hook to be nil")
	}
	if ps.Hooks.AfterEach != nil {
		t.Error("expected afterEach hook to be nil")
	}

	if len(ps.Tests) != 2 {
		t.Fatalf("expected 2 tests, got %d", len(ps.Tests))
	}
}

func TestParseExports_NoTests(t *testing.T) {
	script := `
		module.exports = {
			beforeAll: function(pm) { },
		};
	`

	vm, exports := newTestVM(t, script)
	ps, err := ParseExports(vm, exports)
	if err != nil {
		t.Fatalf("ParseExports error: %v", err)
	}

	if ps.Hooks.BeforeAll == nil {
		t.Error("expected beforeAll hook to be set")
	}
	if len(ps.Tests) != 0 {
		t.Errorf("expected 0 tests, got %d", len(ps.Tests))
	}
}

func TestParseExports_SkipsNonCallable(t *testing.T) {
	script := `
		module.exports = {
			"description": "My test suite",
			"version": 1,
			"users/list": function(pm) { },
		};
	`

	vm, exports := newTestVM(t, script)
	ps, err := ParseExports(vm, exports)
	if err != nil {
		t.Fatalf("ParseExports error: %v", err)
	}

	if len(ps.Tests) != 1 {
		t.Fatalf("expected 1 test (skipping non-callable), got %d", len(ps.Tests))
	}
}

func TestParseExports_NilExports(t *testing.T) {
	vm := goja.New()
	_, err := ParseExports(vm, nil)
	if err == nil {
		t.Fatal("expected error for nil exports")
	}
}

func TestParseExports_UndefinedExports(t *testing.T) {
	vm := goja.New()
	_, err := ParseExports(vm, goja.Undefined())
	if err == nil {
		t.Fatal("expected error for undefined exports")
	}
}

func TestParseExports_EmptyObject(t *testing.T) {
	script := `module.exports = {};`

	vm, exports := newTestVM(t, script)
	ps, err := ParseExports(vm, exports)
	if err != nil {
		t.Fatalf("ParseExports error: %v", err)
	}

	if len(ps.Tests) != 0 {
		t.Errorf("expected 0 tests, got %d", len(ps.Tests))
	}
}

func TestTestKeys(t *testing.T) {
	script := `
		module.exports = {
			beforeAll: function(pm) { },
			"users/list": function(pm) { },
			"admin/get": function(pm) { },
		};
	`

	vm, exports := newTestVM(t, script)
	ps, err := ParseExports(vm, exports)
	if err != nil {
		t.Fatalf("ParseExports error: %v", err)
	}

	keys := ps.TestKeys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	// Verify both keys are present (order not guaranteed).
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["users/list"] || !keySet["admin/get"] {
		t.Errorf("unexpected keys: %v", keys)
	}
}

func TestRunHook_NilHook(t *testing.T) {
	vm := goja.New()
	err := RunHook(vm, nil, goja.Undefined())
	if err != nil {
		t.Fatalf("RunHook with nil should not error: %v", err)
	}
}

func TestRunHook_Success(t *testing.T) {
	vm := goja.New()
	called := false
	fn := func(call goja.FunctionCall) goja.Value {
		called = true
		return goja.Undefined()
	}
	callable, _ := goja.AssertFunction(vm.ToValue(fn))

	err := RunHook(vm, callable, goja.Undefined())
	if err != nil {
		t.Fatalf("RunHook error: %v", err)
	}
	if !called {
		t.Error("hook function was not called")
	}
}

func TestRunHook_ReceivesPMObject(t *testing.T) {
	vm := goja.New()
	var receivedArg goja.Value
	fn := func(call goja.FunctionCall) goja.Value {
		receivedArg = call.Argument(0)
		return goja.Undefined()
	}
	callable, _ := goja.AssertFunction(vm.ToValue(fn))

	pmObj := vm.NewObject()
	_ = pmObj.Set("test", "value")

	err := RunHook(vm, callable, pmObj)
	if err != nil {
		t.Fatalf("RunHook error: %v", err)
	}

	if receivedArg == nil || goja.IsUndefined(receivedArg) {
		t.Fatal("hook did not receive pm object")
	}
}
