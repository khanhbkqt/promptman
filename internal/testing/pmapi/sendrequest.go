package pmapi

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/khanhnguyen/promptman/internal/request"
)

// RequestExecutor abstracts the ability to execute HTTP requests.
// This interface is defined in pmapi to avoid a hard dependency on
// the request module. The test runner passes a concrete implementation
// that delegates to the request engine.
type RequestExecutor interface {
	// Execute runs a request by collection and request ID, returning
	// the response or an error.
	Execute(collectionID, requestID, env string) (*request.Response, error)
}

// sendRequestConfig holds the configuration parsed from a pm.sendRequest
// call when the first argument is an object rather than a string.
type sendRequestConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// injectSendRequest creates the pm.sendRequest(reqIdOrConfig, callback)
// function. If executor is nil, the function throws an error when called.
func injectSendRequest(vm *goja.Runtime, executor RequestExecutor, collectionID, env string) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if executor == nil {
			panic(vm.NewGoError(fmt.Errorf("pm.sendRequest: no request executor configured")))
		}

		callback, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			panic(vm.NewGoError(fmt.Errorf("pm.sendRequest: second argument must be a callback function")))
		}

		arg0 := call.Argument(0)

		// If the first argument is a string, treat it as a request ID.
		if arg0.ExportType().Kind().String() == "string" {
			reqID := arg0.String()
			resp, err := executor.Execute(collectionID, reqID, env)
			if err != nil {
				_, _ = callback(goja.Undefined(), vm.ToValue(err.Error()), goja.Undefined())
			} else {
				rw := NewResponseWrapper(vm, resp)
				_, _ = callback(goja.Undefined(), goja.Null(), rw.ToObject())
			}
			return goja.Undefined()
		}

		// Otherwise, treat as a request config object.
		// For now, we only support string request IDs. Object config
		// support will be added when the request engine supports it.
		panic(vm.NewGoError(fmt.Errorf("pm.sendRequest: object config not yet supported; use a request ID string")))
	}
}
