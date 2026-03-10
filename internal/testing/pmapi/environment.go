package pmapi

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/khanhnguyen/promptman/internal/testing/assertions"
)

// VariableScope holds key-value pairs for a single variable scope
// (environment, variables, or collectionVariables). All mutations
// are runtime-only and not persisted to disk.
type VariableScope struct {
	data map[string]string
}

// NewVariableScope creates a scope initialized with the given values.
// A nil map is treated as empty.
func NewVariableScope(initial map[string]string) *VariableScope {
	data := make(map[string]string)
	for k, v := range initial {
		data[k] = v
	}
	return &VariableScope{data: data}
}

// Get returns the value for key, or empty string if not set.
func (vs *VariableScope) Get(key string) string {
	return vs.data[key]
}

// Set stores a key-value pair in the scope (runtime only).
func (vs *VariableScope) Set(key, value string) {
	vs.data[key] = value
}

// All returns a copy of all variables in this scope.
func (vs *VariableScope) All() map[string]string {
	cp := make(map[string]string, len(vs.data))
	for k, v := range vs.data {
		cp[k] = v
	}
	return cp
}

// injectScopeObject creates a goja object with get(key) and set(key, value)
// methods for a VariableScope.
func injectScopeObject(vm *goja.Runtime, scope *VariableScope) *goja.Object {
	obj := vm.NewObject()

	_ = obj.Set("get", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		val := scope.Get(key)
		if val == "" {
			return goja.Undefined()
		}
		return vm.ToValue(val)
	})

	_ = obj.Set("set", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		value := call.Argument(1).String()
		scope.Set(key, value)
		return goja.Undefined()
	})

	return obj
}

// injectAssertObject creates the pm.assert.* simple assertion API
// as a goja object.
func injectAssertObject(vm *goja.Runtime) *goja.Object {
	obj := vm.NewObject()

	_ = obj.Set("equal", func(call goja.FunctionCall) goja.Value {
		actual := call.Argument(0).Export()
		expected := call.Argument(1).Export()
		msg := optionalString(call, 2)
		assertions.AssertEqual(actual, expected, msg)
		return goja.Undefined()
	})

	_ = obj.Set("contains", func(call goja.FunctionCall) goja.Value {
		haystack := call.Argument(0).Export()
		needle := call.Argument(1).Export()
		msg := optionalString(call, 2)
		assertions.AssertContains(haystack, needle, msg)
		return goja.Undefined()
	})

	_ = obj.Set("type", func(call goja.FunctionCall) goja.Value {
		value := call.Argument(0).Export()
		typeName := call.Argument(1).String()
		msg := optionalString(call, 2)
		assertions.AssertType(value, typeName, msg)
		return goja.Undefined()
	})

	_ = obj.Set("hasProperty", func(call goja.FunctionCall) goja.Value {
		obj := call.Argument(0).Export()
		prop := call.Argument(1).String()
		msg := optionalString(call, 2)
		assertions.AssertHasProperty(obj, prop, msg)
		return goja.Undefined()
	})

	_ = obj.Set("below", func(call goja.FunctionCall) goja.Value {
		value := call.Argument(0).Export()
		limit := call.Argument(1).ToFloat()
		msg := optionalString(call, 2)
		assertions.AssertBelow(value, limit, msg)
		return goja.Undefined()
	})

	return obj
}

// optionalString extracts an optional string argument at the given index.
func optionalString(call goja.FunctionCall, idx int) string {
	if idx >= len(call.Arguments) {
		return ""
	}
	v := call.Arguments[idx]
	if goja.IsUndefined(v) || goja.IsNull(v) {
		return ""
	}
	return fmt.Sprint(v.Export())
}
