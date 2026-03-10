package pmapi

import (
	"errors"
	"testing"

	"github.com/dop251/goja"
	"github.com/khanhnguyen/promptman/internal/request"
)

type mockExecutor struct {
	responses map[string]*request.Response
	errs      map[string]error
}

func (m *mockExecutor) Execute(collectionID, requestID, env string) (*request.Response, error) {
	if err, ok := m.errs[requestID]; ok {
		return nil, err
	}
	if resp, ok := m.responses[requestID]; ok {
		return resp, nil
	}
	return nil, errors.New("not found")
}

func TestSendRequest_Success(t *testing.T) {
	vm := goja.New()

	executor := &mockExecutor{
		responses: map[string]*request.Response{
			"req1": {
				Status: 200,
				Body:   `{"success": true}`,
			},
		},
	}

	pm := NewPM(vm, nil)
	pm.SetExecutor(executor, "coll1", "env1")
	if err := pm.InjectInto(vm); err != nil {
		t.Fatalf("InjectInto: %v", err)
	}

	_, err := vm.RunString(`
		var resultRes = null;
		var resultErr = null;
		pm.sendRequest("req1", function(err, res) {
			resultErr = err;
			resultRes = res;
		});
	`)

	if err != nil {
		t.Fatalf("RunString: %v", err)
	}

	errVal := vm.Get("resultErr")
	if errVal != nil && !goja.IsNull(errVal) && !goja.IsUndefined(errVal) {
		t.Errorf("expected no error, got %v", errVal)
	}

	resVal := vm.Get("resultRes")
	if resVal == nil || goja.IsUndefined(resVal) || goja.IsNull(resVal) {
		t.Fatal("expected a response object")
	}

	resObj := resVal.ToObject(vm)
	status := resObj.Get("status").ToInteger()
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
}

func TestSendRequest_Error(t *testing.T) {
	vm := goja.New()

	executor := &mockExecutor{
		errs: map[string]error{
			"req1": errors.New("network error"),
		},
	}

	pm := NewPM(vm, nil)
	pm.SetExecutor(executor, "coll1", "env1")
	if err := pm.InjectInto(vm); err != nil {
		t.Fatalf("InjectInto: %v", err)
	}

	_, err := vm.RunString(`
		var resultErr = null;
		pm.sendRequest("req1", function(err, res) {
			resultErr = err;
		});
	`)

	if err != nil {
		t.Fatalf("RunString: %v", err)
	}

	errVal := vm.Get("resultErr")
	if errVal == nil || goja.IsUndefined(errVal) || goja.IsNull(errVal) {
		t.Fatal("expected error, got null/undefined")
	}

	if errVal.String() != "network error" {
		t.Errorf("err = %q, want network error", errVal.String())
	}
}

func TestSendRequest_NoExecutorThrows(t *testing.T) {
	vm := goja.New()

	pm := NewPM(vm, nil)
	// Do not set executor
	if err := pm.InjectInto(vm); err != nil {
		t.Fatalf("InjectInto: %v", err)
	}

	_, err := vm.RunString(`
		var caught = false;
		try {
			pm.sendRequest("req1", function(err, res) {});
		} catch (e) {
			caught = true;
		}
		if (!caught) throw new Error("expected throw");
	`)
	if err != nil {
		t.Fatalf("RunString: %v", err)
	}
}
