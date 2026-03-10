package pmapi

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/khanhnguyen/promptman/internal/request"
)

func TestResponseWrapper_Status(t *testing.T) {
	vm := goja.New()
	resp := &request.Response{Status: 200}
	rw := NewResponseWrapper(vm, resp)
	obj := rw.ToObject()

	v := obj.Get("status")
	if v.ToInteger() != 200 {
		t.Errorf("status = %v, want 200", v)
	}
}

func TestResponseWrapper_Headers(t *testing.T) {
	vm := goja.New()
	resp := &request.Response{
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Custom":     "value",
		},
	}
	rw := NewResponseWrapper(vm, resp)
	obj := rw.ToObject()

	headers := obj.Get("headers").ToObject(vm)
	ct := headers.Get("Content-Type")
	if ct.String() != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestResponseWrapper_Body(t *testing.T) {
	vm := goja.New()
	resp := &request.Response{Body: `{"id": 1}`}
	rw := NewResponseWrapper(vm, resp)
	obj := rw.ToObject()

	body := obj.Get("body")
	if body.String() != `{"id": 1}` {
		t.Errorf("body = %q, want %q", body, `{"id": 1}`)
	}
}

func TestResponseWrapper_JSON(t *testing.T) {
	vm := goja.New()
	resp := &request.Response{Body: `{"id": 1, "name": "test"}`}
	rw := NewResponseWrapper(vm, resp)
	obj := rw.ToObject()

	// Call json() function
	jsonFn, ok := goja.AssertFunction(obj.Get("json"))
	if !ok {
		t.Fatal("json is not a function")
	}
	result, err := jsonFn(goja.Undefined())
	if err != nil {
		t.Fatalf("json() error: %v", err)
	}

	parsed := result.ToObject(vm)
	id := parsed.Get("id")
	if id.ToInteger() != 1 {
		t.Errorf("json().id = %v, want 1", id)
	}
}

func TestResponseWrapper_JSON_Invalid(t *testing.T) {
	vm := goja.New()
	resp := &request.Response{Body: "not json"}
	rw := NewResponseWrapper(vm, resp)
	obj := rw.ToObject()

	jsonFn, ok := goja.AssertFunction(obj.Get("json"))
	if !ok {
		t.Fatal("json is not a function")
	}

	// goja catches the panic(NewGoError) and returns it as an exception.
	_, err := jsonFn(goja.Undefined())
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestResponseWrapper_Text(t *testing.T) {
	vm := goja.New()
	resp := &request.Response{Body: "hello world"}
	rw := NewResponseWrapper(vm, resp)
	obj := rw.ToObject()

	textFn, ok := goja.AssertFunction(obj.Get("text"))
	if !ok {
		t.Fatal("text is not a function")
	}
	result, err := textFn(goja.Undefined())
	if err != nil {
		t.Fatalf("text() error: %v", err)
	}
	if result.String() != "hello world" {
		t.Errorf("text() = %q, want %q", result, "hello world")
	}
}

func TestResponseWrapper_Time(t *testing.T) {
	vm := goja.New()
	resp := &request.Response{
		Timing: &request.RequestTiming{Total: 150},
	}
	rw := NewResponseWrapper(vm, resp)
	obj := rw.ToObject()

	v := obj.Get("time")
	if v.ToInteger() != 150 {
		t.Errorf("time = %v, want 150", v)
	}
}

func TestResponseWrapper_Time_NilTiming(t *testing.T) {
	vm := goja.New()
	resp := &request.Response{}
	rw := NewResponseWrapper(vm, resp)
	obj := rw.ToObject()

	v := obj.Get("time")
	if v.ToInteger() != 0 {
		t.Errorf("time = %v, want 0", v)
	}
}
