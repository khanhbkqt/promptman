package assertions

import (
	"testing"

	pkgtesting "github.com/khanhnguyen/promptman/internal/testing"
)

func TestChaiAssertion_Equal(t *testing.T) {
	tests := []struct {
		name     string
		actual   any
		expected any
		wantFail bool
	}{
		{"int match", 42, 42, false},
		{"int mismatch", 42, 99, true},
		{"string match", "hello", "hello", false},
		{"string mismatch", "hello", "world", true},
		{"float match", 3.14, 3.14, false},
		{"int64 vs float64", int64(42), float64(42), false},
		{"nil match", nil, nil, false},
		{"nil vs value", nil, "x", true},
		{"slice match", []any{1, 2}, []any{1, 2}, false},
		{"map match", map[string]any{"a": float64(1)}, map[string]any{"a": float64(1)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChaiAssertion(tt.actual)
			err := catchTestError(func() { c.Equal(tt.expected) })
			if tt.wantFail && err == nil {
				t.Error("expected failure but assertion passed")
			}
			if !tt.wantFail && err != nil {
				t.Errorf("expected pass but got failure: %s", err.Message)
			}
		})
	}
}

func TestChaiAssertion_Equal_Negated(t *testing.T) {
	tests := []struct {
		name     string
		actual   any
		expected any
		wantFail bool
	}{
		{"not equal pass", 1, 2, false},
		{"not equal fail", 1, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChaiAssertion(tt.actual)
			c.Not()
			err := catchTestError(func() { c.Equal(tt.expected) })
			if tt.wantFail && err == nil {
				t.Error("expected failure but assertion passed")
			}
			if !tt.wantFail && err != nil {
				t.Errorf("expected pass but got failure: %s", err.Message)
			}
		})
	}
}

func TestChaiAssertion_An(t *testing.T) {
	tests := []struct {
		name     string
		actual   any
		typeName string
		wantFail bool
	}{
		{"array", []any{1, 2, 3}, "array", false},
		{"object", map[string]any{"a": 1}, "object", false},
		{"string", "hello", "string", false},
		{"number int", 42, "number", false},
		{"number float", 3.14, "number", false},
		{"boolean", true, "boolean", false},
		{"null", nil, "null", false},
		{"wrong type", "hello", "number", true},
		{"array not string", []any{1}, "string", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChaiAssertion(tt.actual)
			err := catchTestError(func() { c.An(tt.typeName) })
			if tt.wantFail && err == nil {
				t.Error("expected failure but assertion passed")
			}
			if !tt.wantFail && err != nil {
				t.Errorf("expected pass but got failure: %s", err.Message)
			}
		})
	}
}

func TestChaiAssertion_Property(t *testing.T) {
	obj := map[string]any{"id": 1, "name": "test"}

	tests := []struct {
		name     string
		actual   any
		prop     string
		wantFail bool
	}{
		{"existing property", obj, "id", false},
		{"missing property", obj, "missing", true},
		{"not a map", "hello", "x", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChaiAssertion(tt.actual)
			err := catchTestError(func() { c.Property(tt.prop) })
			if tt.wantFail && err == nil {
				t.Error("expected failure but assertion passed")
			}
			if !tt.wantFail && err != nil {
				t.Errorf("expected pass but got failure: %s", err.Message)
			}
		})
	}
}

func TestChaiAssertion_Include(t *testing.T) {
	tests := []struct {
		name     string
		actual   any
		needle   any
		wantFail bool
	}{
		{"string contains", "hello world", "world", false},
		{"string not contains", "hello world", "xyz", true},
		{"slice contains", []any{"a", "b", "c"}, "b", false},
		{"slice not contains", []any{"a", "b"}, "z", true},
		{"empty string", "", "x", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChaiAssertion(tt.actual)
			err := catchTestError(func() { c.Include(tt.needle) })
			if tt.wantFail && err == nil {
				t.Error("expected failure but assertion passed")
			}
			if !tt.wantFail && err != nil {
				t.Errorf("expected pass but got failure: %s", err.Message)
			}
		})
	}
}

func TestChaiAssertion_Below(t *testing.T) {
	tests := []struct {
		name     string
		actual   any
		limit    float64
		wantFail bool
	}{
		{"below pass", 100, 500, false},
		{"below fail", 600, 500, true},
		{"equal fail", 500, 500, true},
		{"float below", 3.14, 4.0, false},
		{"not a number", "hello", 5.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChaiAssertion(tt.actual)
			err := catchTestError(func() { c.Below(tt.limit) })
			if tt.wantFail && err == nil {
				t.Error("expected failure but assertion passed")
			}
			if !tt.wantFail && err != nil {
				t.Errorf("expected pass but got failure: %s", err.Message)
			}
		})
	}
}

func TestChaiAssertion_Above(t *testing.T) {
	tests := []struct {
		name     string
		actual   any
		limit    float64
		wantFail bool
	}{
		{"above pass", 600, 500, false},
		{"above fail", 100, 500, true},
		{"equal fail", 500, 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChaiAssertion(tt.actual)
			err := catchTestError(func() { c.Above(tt.limit) })
			if tt.wantFail && err == nil {
				t.Error("expected failure but assertion passed")
			}
			if !tt.wantFail && err != nil {
				t.Errorf("expected pass but got failure: %s", err.Message)
			}
		})
	}
}

func TestChaiAssertion_Chaining(t *testing.T) {
	t.Run("to.be.an", func(t *testing.T) {
		c := NewChaiAssertion([]any{1, 2, 3})
		err := catchTestError(func() {
			c.To().Be().An("array")
		})
		if err != nil {
			t.Errorf("unexpected failure: %s", err.Message)
		}
	})

	t.Run("to.not.equal", func(t *testing.T) {
		c := NewChaiAssertion(1)
		err := catchTestError(func() {
			c.To().Not().Equal(2)
		})
		if err != nil {
			t.Errorf("unexpected failure: %s", err.Message)
		}
	})

	t.Run("to.have.property", func(t *testing.T) {
		c := NewChaiAssertion(map[string]any{"id": 1})
		err := catchTestError(func() {
			c.To().Have().Property("id")
		})
		if err != nil {
			t.Errorf("unexpected failure: %s", err.Message)
		}
	})

	t.Run("to.not.be.an", func(t *testing.T) {
		c := NewChaiAssertion("hello")
		err := catchTestError(func() {
			c.To().Not().Be().An("array")
		})
		if err != nil {
			t.Errorf("unexpected failure: %s", err.Message)
		}
	})
}

func TestChaiAssertion_ErrorFormat(t *testing.T) {
	c := NewChaiAssertion(42)
	err := catchTestError(func() { c.Equal(99) })
	if err == nil {
		t.Fatal("expected failure")
	}
	if err.Expected == nil || err.Actual == nil {
		t.Error("expected and actual should be set")
	}
	if err.Message == "" {
		t.Error("message should not be empty")
	}
}

func TestGoTypeLabel(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{"nil", nil, "null"},
		{"int", 42, "number"},
		{"float64", 3.14, "number"},
		{"string", "hello", "string"},
		{"bool", true, "boolean"},
		{"slice", []any{1}, "array"},
		{"map", map[string]any{}, "object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := goTypeLabel(tt.value)
			if got != tt.expected {
				t.Errorf("goTypeLabel(%v) = %q, want %q", tt.value, got, tt.expected)
			}
		})
	}
}

// catchTestError runs fn and returns the *TestError if one was panicked.
func catchTestError(fn func()) (te *pkgtesting.TestError) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(*pkgtesting.TestError); ok {
				te = e
			} else {
				panic(r) // re-panic non-TestError panics
			}
		}
	}()
	fn()
	return nil
}
