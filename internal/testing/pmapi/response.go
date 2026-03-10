package pmapi

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
	"github.com/khanhnguyen/promptman/internal/request"
)

// ResponseWrapper wraps a request.Response for JavaScript access
// inside the goja sandbox. It exposes status, headers, body, json(),
// text(), and time properties.
type ResponseWrapper struct {
	resp *request.Response
	vm   *goja.Runtime
}

// NewResponseWrapper creates a ResponseWrapper for the given response.
func NewResponseWrapper(vm *goja.Runtime, resp *request.Response) *ResponseWrapper {
	return &ResponseWrapper{resp: resp, vm: vm}
}

// ToObject creates a goja object with all pm.response properties.
func (rw *ResponseWrapper) ToObject() *goja.Object {
	obj := rw.vm.NewObject()

	_ = obj.Set("status", rw.resp.Status)
	_ = obj.Set("headers", rw.headersObject())
	_ = obj.Set("body", rw.resp.Body)
	_ = obj.Set("time", rw.totalTime())

	_ = obj.Set("json", func(call goja.FunctionCall) goja.Value {
		var parsed any
		if err := json.Unmarshal([]byte(rw.resp.Body), &parsed); err != nil {
			panic(rw.vm.NewGoError(fmt.Errorf("pm.response.json(): invalid JSON: %w", err)))
		}
		return rw.vm.ToValue(parsed)
	})

	_ = obj.Set("text", func(call goja.FunctionCall) goja.Value {
		return rw.vm.ToValue(rw.resp.Body)
	})

	return obj
}

// headersObject converts the flat header map to a goja object.
func (rw *ResponseWrapper) headersObject() *goja.Object {
	obj := rw.vm.NewObject()
	for k, v := range rw.resp.Headers {
		_ = obj.Set(k, v)
	}
	return obj
}

// totalTime returns the total response time in milliseconds.
func (rw *ResponseWrapper) totalTime() int {
	if rw.resp.Timing != nil {
		return rw.resp.Timing.Total
	}
	return 0
}
