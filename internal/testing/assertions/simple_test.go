package assertions

import (
	"testing"

	pkgtesting "github.com/khanhnguyen/promptman/internal/testing"
)

func TestAssertEqual_Pass(t *testing.T) {
	tests := []struct {
		name     string
		actual   any
		expected any
	}{
		{"int", 42, 42},
		{"string", "hello", "hello"},
		{"nil", nil, nil},
		{"int64 vs float64", int64(10), float64(10)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := catchTestError(func() { AssertEqual(tt.actual, tt.expected, "") })
			if err != nil {
				t.Errorf("unexpected failure: %s", err.Message)
			}
		})
	}
}

func TestAssertEqual_Fail(t *testing.T) {
	err := catchTestError(func() { AssertEqual(1, 2, "") })
	if err == nil {
		t.Error("expected failure")
	}
}

func TestAssertEqual_CustomMessage(t *testing.T) {
	err := catchTestError(func() { AssertEqual(1, 2, "custom msg") })
	if err == nil {
		t.Fatal("expected failure")
	}
	if err.Message != "custom msg" {
		t.Errorf("message = %q, want %q", err.Message, "custom msg")
	}
}

func TestAssertContains_String_Pass(t *testing.T) {
	err := catchTestError(func() { AssertContains("hello world", "world", "") })
	if err != nil {
		t.Errorf("unexpected failure: %s", err.Message)
	}
}

func TestAssertContains_String_Fail(t *testing.T) {
	err := catchTestError(func() { AssertContains("hello", "xyz", "") })
	if err == nil {
		t.Error("expected failure")
	}
}

func TestAssertContains_Slice_Pass(t *testing.T) {
	err := catchTestError(func() { AssertContains([]any{"a", "b", "c"}, "b", "") })
	if err != nil {
		t.Errorf("unexpected failure: %s", err.Message)
	}
}

func TestAssertContains_Slice_Fail(t *testing.T) {
	err := catchTestError(func() { AssertContains([]any{"a", "b"}, "z", "") })
	if err == nil {
		t.Error("expected failure")
	}
}

func TestAssertContains_InvalidType(t *testing.T) {
	err := catchTestError(func() { AssertContains(42, "x", "") })
	if err == nil {
		t.Error("expected failure")
	}
}

func TestAssertType_Pass(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		typeName string
	}{
		{"string", "hello", "string"},
		{"number", 42, "number"},
		{"boolean", true, "boolean"},
		{"array", []any{1}, "array"},
		{"object", map[string]any{}, "object"},
		{"null", nil, "null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := catchTestError(func() { AssertType(tt.value, tt.typeName, "") })
			if err != nil {
				t.Errorf("unexpected failure: %s", err.Message)
			}
		})
	}
}

func TestAssertType_Fail(t *testing.T) {
	err := catchTestError(func() { AssertType("hello", "number", "") })
	if err == nil {
		t.Error("expected failure")
	}
}

func TestAssertHasProperty_Pass(t *testing.T) {
	obj := map[string]any{"id": 1, "name": "test"}
	err := catchTestError(func() { AssertHasProperty(obj, "id", "") })
	if err != nil {
		t.Errorf("unexpected failure: %s", err.Message)
	}
}

func TestAssertHasProperty_Missing(t *testing.T) {
	obj := map[string]any{"id": 1}
	err := catchTestError(func() { AssertHasProperty(obj, "missing", "") })
	if err == nil {
		t.Error("expected failure")
	}
}

func TestAssertHasProperty_NotMap(t *testing.T) {
	err := catchTestError(func() { AssertHasProperty("hello", "x", "") })
	if err == nil {
		t.Error("expected failure")
	}
}

func TestAssertBelow_Pass(t *testing.T) {
	err := catchTestError(func() { AssertBelow(100, 500, "") })
	if err != nil {
		t.Errorf("unexpected failure: %s", err.Message)
	}
}

func TestAssertBelow_Fail(t *testing.T) {
	err := catchTestError(func() { AssertBelow(600, 500, "") })
	if err == nil {
		t.Error("expected failure")
	}
}

func TestAssertBelow_Equal(t *testing.T) {
	err := catchTestError(func() { AssertBelow(500, 500, "") })
	if err == nil {
		t.Error("expected failure for equal values")
	}
}

func TestAssertBelow_NotNumber(t *testing.T) {
	err := catchTestError(func() { AssertBelow("hello", 5, "") })
	if err == nil {
		t.Error("expected failure")
	}
}

// catchTestError is defined in chai_test.go — reusing signature.
func catchSimpleTestError(fn func()) (te *pkgtesting.TestError) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(*pkgtesting.TestError); ok {
				te = e
			} else {
				panic(r)
			}
		}
	}()
	fn()
	return nil
}
