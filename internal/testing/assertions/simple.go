package assertions

import (
	"fmt"
	"reflect"
	"strings"

	testing "github.com/khanhnguyen/promptman/internal/testing"
)

// AssertEqual checks that actual equals expected using reflect.DeepEqual.
// Panics with *TestError on failure.
func AssertEqual(actual, expected any, message string) {
	a := normalizeNumeric(actual)
	e := normalizeNumeric(expected)
	if !reflect.DeepEqual(a, e) {
		msg := message
		if msg == "" {
			msg = fmt.Sprintf("expected %v to equal %v", actual, expected)
		}
		panic(&testing.TestError{Expected: expected, Actual: actual, Message: msg})
	}
}

// AssertContains checks that haystack contains needle.
// For strings, checks substring. For slices, checks element membership.
// Panics with *TestError on failure.
func AssertContains(haystack, needle any, message string) {
	switch h := haystack.(type) {
	case string:
		n, ok := needle.(string)
		if !ok {
			panic(&testing.TestError{
				Expected: needle,
				Actual:   haystack,
				Message:  orDefault(message, "expected string needle for string contains"),
			})
		}
		if !strings.Contains(h, n) {
			panic(&testing.TestError{
				Expected: fmt.Sprintf("containing %q", n),
				Actual:   h,
				Message:  orDefault(message, fmt.Sprintf("expected %q to contain %q", h, n)),
			})
		}
	default:
		rv := reflect.ValueOf(haystack)
		if rv.Kind() != reflect.Slice {
			panic(&testing.TestError{
				Expected: needle,
				Actual:   haystack,
				Message:  orDefault(message, fmt.Sprintf("expected string or array, got %T", haystack)),
			})
		}
		for i := 0; i < rv.Len(); i++ {
			if reflect.DeepEqual(rv.Index(i).Interface(), needle) {
				return
			}
		}
		panic(&testing.TestError{
			Expected: needle,
			Actual:   haystack,
			Message:  orDefault(message, fmt.Sprintf("expected %v to contain %v", haystack, needle)),
		})
	}
}

// AssertType checks that value matches the named type.
// Supported: "string", "number", "boolean", "array", "object", "null".
// Panics with *TestError on failure.
func AssertType(value any, typeName string, message string) {
	got := goTypeLabel(value)
	if got != strings.ToLower(typeName) {
		panic(&testing.TestError{
			Expected: typeName,
			Actual:   got,
			Message:  orDefault(message, fmt.Sprintf("expected type %q, got %q", typeName, got)),
		})
	}
}

// AssertHasProperty checks that obj (expected map) has the named property.
// Panics with *TestError on failure.
func AssertHasProperty(obj any, prop string, message string) {
	m, ok := toMap(obj)
	if !ok {
		panic(&testing.TestError{
			Expected: fmt.Sprintf("object with property %q", prop),
			Actual:   obj,
			Message:  orDefault(message, fmt.Sprintf("expected a map, got %T", obj)),
		})
	}
	if _, exists := m[prop]; !exists {
		panic(&testing.TestError{
			Expected: fmt.Sprintf("property %q", prop),
			Actual:   formatKeys(m),
			Message:  orDefault(message, fmt.Sprintf("expected object to have property %q", prop)),
		})
	}
}

// AssertBelow checks that value is numerically less than limit.
// Panics with *TestError on failure.
func AssertBelow(value any, limit float64, message string) {
	n, ok := toFloat64(value)
	if !ok {
		panic(&testing.TestError{
			Expected: fmt.Sprintf("number below %v", limit),
			Actual:   value,
			Message:  orDefault(message, fmt.Sprintf("expected a number, got %T", value)),
		})
	}
	if n >= limit {
		panic(&testing.TestError{
			Expected: fmt.Sprintf("below %v", limit),
			Actual:   n,
			Message:  orDefault(message, fmt.Sprintf("expected %v to be below %v", n, limit)),
		})
	}
}

// orDefault returns msg if non-empty, otherwise returns fallback.
func orDefault(msg, fallback string) string {
	if msg != "" {
		return msg
	}
	return fallback
}
