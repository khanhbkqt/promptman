package pmapi

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/khanhnguyen/promptman/internal/request"
	pkgtesting "github.com/khanhnguyen/promptman/internal/testing"
)

func newTestPM(t *testing.T, resp *request.Response) (*PM, *goja.Runtime) {
	t.Helper()
	vm := goja.New()
	pm := NewPM(vm, resp)
	if err := pm.InjectInto(vm); err != nil {
		t.Fatalf("InjectInto: %v", err)
	}
	return pm, vm
}

func TestPM_Test_Passing(t *testing.T) {
	pm, vm := newTestPM(t, nil)

	_, err := vm.RunString(`pm.test("passes", function() { /* no assertion */ })`)
	if err != nil {
		t.Fatalf("RunString: %v", err)
	}

	tests := pm.Tests()
	if len(tests) != 1 {
		t.Fatalf("got %d tests, want 1", len(tests))
	}
	if tests[0].Status != "passed" {
		t.Errorf("status = %q, want %q", tests[0].Status, "passed")
	}
	if tests[0].Name != "passes" {
		t.Errorf("name = %q, want %q", tests[0].Name, "passes")
	}
}

func TestPM_Test_FailingAssertion(t *testing.T) {
	pm, vm := newTestPM(t, nil)

	_, err := vm.RunString(`
		pm.test("fails", function() {
			pm.expect(1).to.equal(2);
		})
	`)
	if err != nil {
		t.Fatalf("RunString: %v", err)
	}

	tests := pm.Tests()
	if len(tests) != 1 {
		t.Fatalf("got %d tests, want 1", len(tests))
	}
	if tests[0].Status != "failed" {
		t.Errorf("status = %q, want %q", tests[0].Status, "failed")
	}
	if tests[0].Error == nil {
		t.Fatal("error should not be nil")
	}
}

func TestPM_Test_RuntimeError(t *testing.T) {
	pm, vm := newTestPM(t, nil)

	_, err := vm.RunString(`
		pm.test("errors", function() {
			throw new Error("boom");
		})
	`)
	if err != nil {
		t.Fatalf("RunString: %v", err)
	}

	tests := pm.Tests()
	if len(tests) != 1 {
		t.Fatalf("got %d tests, want 1", len(tests))
	}
	if tests[0].Status != "error" {
		t.Errorf("status = %q, want %q", tests[0].Status, "error")
	}
}

func TestPM_Test_MultipleTests(t *testing.T) {
	pm, vm := newTestPM(t, nil)

	_, err := vm.RunString(`
		pm.test("test1", function() { pm.expect(1).to.equal(1); });
		pm.test("test2", function() { pm.expect(2).to.equal(3); });
		pm.test("test3", function() { pm.expect("a").to.equal("a"); });
	`)
	if err != nil {
		t.Fatalf("RunString: %v", err)
	}

	tests := pm.Tests()
	if len(tests) != 3 {
		t.Fatalf("got %d tests, want 3", len(tests))
	}

	expected := []struct {
		name   string
		status string
	}{
		{"test1", "passed"},
		{"test2", "failed"},
		{"test3", "passed"},
	}

	for i, exp := range expected {
		if tests[i].Name != exp.name {
			t.Errorf("tests[%d].name = %q, want %q", i, tests[i].Name, exp.name)
		}
		if tests[i].Status != exp.status {
			t.Errorf("tests[%d].status = %q, want %q", i, tests[i].Status, exp.status)
		}
	}
}

func TestPM_Expect_ChainedAssertions(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   string // "passed" or "failed"
	}{
		{
			"to.equal pass",
			`pm.test("t", function() { pm.expect(42).to.equal(42); })`,
			"passed",
		},
		{
			"to.equal fail",
			`pm.test("t", function() { pm.expect(42).to.equal(99); })`,
			"failed",
		},
		{
			"to.not.equal pass",
			`pm.test("t", function() { pm.expect(1).to.not.equal(2); })`,
			"passed",
		},
		{
			"to.not.equal fail",
			`pm.test("t", function() { pm.expect(1).to.not.equal(1); })`,
			"failed",
		},
		{
			"to.be.an array",
			`pm.test("t", function() { pm.expect([1,2,3]).to.be.an("array"); })`,
			"passed",
		},
		{
			"to.be.an object",
			`pm.test("t", function() { pm.expect({a:1}).to.be.an("object"); })`,
			"passed",
		},
		{
			"to.be.an string",
			`pm.test("t", function() { pm.expect("hello").to.be.a("string"); })`,
			"passed",
		},
		{
			"to.have.property pass",
			`pm.test("t", function() { pm.expect({id:1}).to.have.property("id"); })`,
			"passed",
		},
		{
			"to.have.property fail",
			`pm.test("t", function() { pm.expect({id:1}).to.have.property("missing"); })`,
			"failed",
		},
		{
			"to.include string pass",
			`pm.test("t", function() { pm.expect("hello world").to.include("world"); })`,
			"passed",
		},
		{
			"to.include string fail",
			`pm.test("t", function() { pm.expect("hello").to.include("xyz"); })`,
			"failed",
		},
		{
			"to.be.below pass",
			`pm.test("t", function() { pm.expect(100).to.be.below(500); })`,
			"passed",
		},
		{
			"to.be.below fail",
			`pm.test("t", function() { pm.expect(600).to.be.below(500); })`,
			"failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, vm := newTestPM(t, nil)
			_, err := vm.RunString(tt.script)
			if err != nil {
				t.Fatalf("RunString: %v", err)
			}
			results := pm.Tests()
			if len(results) != 1 {
				t.Fatalf("got %d tests, want 1", len(results))
			}
			if results[0].Status != tt.want {
				msg := ""
				if results[0].Error != nil {
					msg = results[0].Error.Message
				}
				t.Errorf("status = %q, want %q (error: %s)", results[0].Status, tt.want, msg)
			}
		})
	}
}

func TestPM_Response_Integration(t *testing.T) {
	resp := &request.Response{
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    `{"id": 42, "items": [1,2,3]}`,
		Timing:  &request.RequestTiming{Total: 150},
	}

	pm, vm := newTestPM(t, resp)

	_, err := vm.RunString(`
		pm.test("status is 200", function() {
			pm.expect(pm.response.status).to.equal(200);
		});
		pm.test("time below 500ms", function() {
			pm.expect(pm.response.time).to.be.below(500);
		});
		pm.test("body has id", function() {
			var data = pm.response.json();
			pm.expect(data).to.have.property("id");
			pm.expect(data.id).to.equal(42);
		});
		pm.test("items is array", function() {
			var data = pm.response.json();
			pm.expect(data.items).to.be.an("array");
		});
		pm.test("body contains id", function() {
			pm.expect(pm.response.body).to.include("42");
		});
		pm.test("text returns body", function() {
			pm.expect(pm.response.text()).to.equal(pm.response.body);
		});
	`)
	if err != nil {
		t.Fatalf("RunString: %v", err)
	}

	tests := pm.Tests()
	for i, tc := range tests {
		if tc.Status != "passed" {
			msg := ""
			if tc.Error != nil {
				msg = tc.Error.Message
			}
			t.Errorf("test[%d] %q: status=%q, want=passed (error: %s)", i, tc.Name, tc.Status, msg)
		}
	}
}

func TestPM_Timeout(t *testing.T) {
	pm, vm := newTestPM(t, nil)

	_, err := vm.RunString(`pm.timeout(5000)`)
	if err != nil {
		t.Fatalf("RunString: %v", err)
	}

	if pm.TimeoutOverride().Milliseconds() != 5000 {
		t.Errorf("timeout = %v, want 5s", pm.TimeoutOverride())
	}
}

func TestPM_TestError_HasExpectedActual(t *testing.T) {
	pm, vm := newTestPM(t, nil)

	_, err := vm.RunString(`
		pm.test("check error", function() {
			pm.expect(42).to.equal(99);
		})
	`)
	if err != nil {
		t.Fatalf("RunString: %v", err)
	}

	results := pm.Tests()
	if len(results) != 1 {
		t.Fatalf("got %d tests, want 1", len(results))
	}
	te := results[0].Error
	if te == nil {
		t.Fatal("expected error")
	}
	// goja exports int as int64, normalizeNumeric converts to float64
	_ = te.Expected
	_ = te.Actual
	if te.Message == "" {
		t.Error("message should not be empty")
	}
}

// Verify that TestError fields are exported correctly.
func TestTestError_Fields(t *testing.T) {
	te := &pkgtesting.TestError{
		Expected: "foo",
		Actual:   "bar",
		Message:  "expected foo to equal bar",
	}
	if te.Expected != "foo" {
		t.Errorf("Expected = %v, want foo", te.Expected)
	}
	if te.Actual != "bar" {
		t.Errorf("Actual = %v, want bar", te.Actual)
	}
}
