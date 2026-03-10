package pmapi

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/khanhnguyen/promptman/internal/request"
	testing "github.com/khanhnguyen/promptman/internal/testing"
	"github.com/khanhnguyen/promptman/internal/testing/assertions"
)

// PM holds the state for a pm.* API instance within a test execution.
// It collects TestCase results from pm.test() calls and provides
// pm.expect(), pm.response, pm.assert, pm.environment, pm.variables,
// pm.collectionVariables, and pm.timeout().
type PM struct {
	vm                  *goja.Runtime
	response            *request.Response
	tests               []testing.TestCase
	timeout             time.Duration // per-test timeout override (0 = use default)
	environment         *VariableScope
	variables           *VariableScope
	collectionVariables *VariableScope
	executor            RequestExecutor // optional: for pm.sendRequest
	collectionID        string          // collection context for sendRequest
	env                 string          // environment context for sendRequest
}

// NewPM creates a new PM instance for the given VM and response.
// If env/vars/collVars are nil, empty scopes are created.
func NewPM(vm *goja.Runtime, resp *request.Response) *PM {
	return &PM{
		vm:                  vm,
		response:            resp,
		environment:         NewVariableScope(nil),
		variables:           NewVariableScope(nil),
		collectionVariables: NewVariableScope(nil),
	}
}

// NewPMWithScopes creates a PM with pre-populated variable scopes.
func NewPMWithScopes(vm *goja.Runtime, resp *request.Response, env, vars, collVars map[string]string) *PM {
	return &PM{
		vm:                  vm,
		response:            resp,
		environment:         NewVariableScope(env),
		variables:           NewVariableScope(vars),
		collectionVariables: NewVariableScope(collVars),
	}
}

// Environment returns the environment variable scope.
func (p *PM) Environment() *VariableScope { return p.environment }

// Variables returns the test-scoped variable scope.
func (p *PM) Variables() *VariableScope { return p.variables }

// CollectionVariables returns the collection-scoped variable scope.
func (p *PM) CollectionVariables() *VariableScope { return p.collectionVariables }

// SetExecutor configures a request executor for pm.sendRequest.
func (p *PM) SetExecutor(executor RequestExecutor, collectionID, env string) {
	p.executor = executor
	p.collectionID = collectionID
	p.env = env
}

// Tests returns the collected test case results.
func (p *PM) Tests() []testing.TestCase {
	return p.tests
}

// TimeoutOverride returns the per-test timeout override.
// Zero means use the default timeout.
func (p *PM) TimeoutOverride() time.Duration {
	return p.timeout
}

// InjectInto registers the pm object as a global in the goja VM.
func (p *PM) InjectInto(vm *goja.Runtime) error {
	obj := vm.NewObject()

	// pm.test(name, fn)
	if err := obj.Set("test", p.testFn()); err != nil {
		return fmt.Errorf("setting pm.test: %w", err)
	}

	// pm.expect(value)
	if err := obj.Set("expect", p.expectFn()); err != nil {
		return fmt.Errorf("setting pm.expect: %w", err)
	}

	// pm.response
	if p.response != nil {
		rw := NewResponseWrapper(vm, p.response)
		if err := obj.Set("response", rw.ToObject()); err != nil {
			return fmt.Errorf("setting pm.response: %w", err)
		}
	}

	// pm.timeout(ms)
	if err := obj.Set("timeout", p.timeoutFn()); err != nil {
		return fmt.Errorf("setting pm.timeout: %w", err)
	}

	// pm.assert.*
	if err := obj.Set("assert", injectAssertObject(vm)); err != nil {
		return fmt.Errorf("setting pm.assert: %w", err)
	}

	// pm.environment.get/set
	if err := obj.Set("environment", injectScopeObject(vm, p.environment)); err != nil {
		return fmt.Errorf("setting pm.environment: %w", err)
	}

	// pm.variables.get/set
	if err := obj.Set("variables", injectScopeObject(vm, p.variables)); err != nil {
		return fmt.Errorf("setting pm.variables: %w", err)
	}

	// pm.collectionVariables.get/set
	if err := obj.Set("collectionVariables", injectScopeObject(vm, p.collectionVariables)); err != nil {
		return fmt.Errorf("setting pm.collectionVariables: %w", err)
	}

	// pm.sendRequest(reqIdOrConfig, callback)
	if err := obj.Set("sendRequest", injectSendRequest(vm, p.executor, p.collectionID, p.env)); err != nil {
		return fmt.Errorf("setting pm.sendRequest: %w", err)
	}

	return vm.Set("pm", obj)
}

// testFn returns the pm.test(name, fn) implementation.
func (p *PM) testFn() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		fn, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			panic(p.vm.NewGoError(fmt.Errorf("pm.test: second argument must be a function")))
		}

		start := time.Now()
		tc := testing.TestCase{
			Name:   name,
			Status: "passed",
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					switch e := r.(type) {
					case *testing.TestError:
						tc.Status = "failed"
						tc.Error = e
					case *goja.InterruptedError:
						// Re-panic timeout interrupts so they propagate.
						panic(e)
					default:
						tc.Status = "error"
						tc.Error = &testing.TestError{
							Message: fmt.Sprint(r),
						}
					}
				}
			}()
			_, err := fn(goja.Undefined())
			if err != nil {
				tc.Status = "error"
				tc.Error = &testing.TestError{
					Message: err.Error(),
				}
			}
		}()

		tc.Duration = int(time.Since(start).Milliseconds())
		p.tests = append(p.tests, tc)
		return goja.Undefined()
	}
}

// expectFn returns the pm.expect(value) implementation.
func (p *PM) expectFn() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		val := call.Argument(0).Export()
		chain := assertions.NewChaiAssertion(val)
		return p.wrapChain(chain)
	}
}

// wrapChain wraps a ChaiAssertion into a goja object with getter
// properties for chaining and callable terminal methods.
func (p *PM) wrapChain(chain *assertions.ChaiAssertion) goja.Value {
	obj := p.vm.NewObject()

	// Pass-through getters — return the same object for chaining.
	self := obj
	for _, name := range []string{
		"to", "be", "been", "is", "that", "which", "and",
		"has", "have", "with", "at", "of", "same", "but",
		"does", "still", "also",
	} {
		_ = obj.Set(name, self)
	}

	// Negation — returns same object but flips the chain's negation.
	_ = obj.Set("not", p.vm.ToValue(p.vm.NewDynamicObject(&notAccessor{
		pm:    p,
		chain: chain,
		obj:   obj,
	})))

	// Actually, the goja dynamic object approach is complex. Let's use
	// a simpler approach: when "not" is accessed, flip the chain's flag
	// and return the same object. We need a property getter for this.
	// Since goja doesn't support property getters via Set(), we use
	// DefineAccessorProperty.
	_ = obj.DefineAccessorProperty("not", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		chain.Not()
		return obj
	}), goja.Undefined(), goja.FLAG_FALSE, goja.FLAG_TRUE)

	// Terminal methods
	_ = obj.Set("equal", func(call goja.FunctionCall) goja.Value {
		expected := call.Argument(0).Export()
		chain.Equal(expected)
		return goja.Undefined()
	})

	_ = obj.Set("eql", func(call goja.FunctionCall) goja.Value {
		expected := call.Argument(0).Export()
		chain.Equal(expected)
		return goja.Undefined()
	})

	_ = obj.Set("a", func(call goja.FunctionCall) goja.Value {
		typeName := call.Argument(0).String()
		chain.A(typeName)
		return goja.Undefined()
	})

	_ = obj.Set("an", func(call goja.FunctionCall) goja.Value {
		typeName := call.Argument(0).String()
		chain.An(typeName)
		return goja.Undefined()
	})

	_ = obj.Set("property", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		chain.Property(name)
		return goja.Undefined()
	})

	_ = obj.Set("include", func(call goja.FunctionCall) goja.Value {
		needle := call.Argument(0).Export()
		chain.Include(needle)
		return goja.Undefined()
	})

	_ = obj.Set("contain", func(call goja.FunctionCall) goja.Value {
		needle := call.Argument(0).Export()
		chain.Include(needle)
		return goja.Undefined()
	})

	_ = obj.Set("below", func(call goja.FunctionCall) goja.Value {
		limit := call.Argument(0).ToFloat()
		chain.Below(limit)
		return goja.Undefined()
	})

	_ = obj.Set("above", func(call goja.FunctionCall) goja.Value {
		limit := call.Argument(0).ToFloat()
		chain.Above(limit)
		return goja.Undefined()
	})

	return obj
}

// timeoutFn returns the pm.timeout(ms) implementation.
func (p *PM) timeoutFn() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		ms := call.Argument(0).ToInteger()
		p.timeout = time.Duration(ms) * time.Millisecond
		return goja.Undefined()
	}
}

// notAccessor implements goja.DynamicObject for the "not" property.
// This is unused — we use DefineAccessorProperty instead — but kept
// as a reference for potential future dynamic property needs.
type notAccessor struct {
	pm    *PM
	chain *assertions.ChaiAssertion
	obj   *goja.Object
}

func (n *notAccessor) Get(key string) goja.Value           { return goja.Undefined() }
func (n *notAccessor) Set(key string, val goja.Value) bool { return false }
func (n *notAccessor) Has(key string) bool                 { return false }
func (n *notAccessor) Delete(key string) bool              { return false }
func (n *notAccessor) Keys() []string                      { return nil }
