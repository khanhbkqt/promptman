package assertions

import (
	"fmt"
	"reflect"
	"strings"

	testing "github.com/khanhnguyen/promptman/internal/testing"
)

// ChaiAssertion implements a Chai BDD-style assertion chain.
// Pass-through getters (.To, .Be, etc.) return the same instance.
// The .Not getter flips the negated flag. Terminal methods (.Equal,
// .Property, etc.) perform the actual check and panic with
// *testing.TestError on failure.
type ChaiAssertion struct {
	actual  any
	negated bool
}

// NewChaiAssertion creates a new assertion chain for the given value.
func NewChaiAssertion(actual any) *ChaiAssertion {
	return &ChaiAssertion{actual: actual}
}

// --- pass-through getters (return self for chaining) ---

// To is a pass-through getter for readability.
func (c *ChaiAssertion) To() *ChaiAssertion { return c }

// Be is a pass-through getter for readability.
func (c *ChaiAssertion) Be() *ChaiAssertion { return c }

// Been is a pass-through getter for readability.
func (c *ChaiAssertion) Been() *ChaiAssertion { return c }

// Is is a pass-through getter for readability.
func (c *ChaiAssertion) Is() *ChaiAssertion { return c }

// That is a pass-through getter for readability.
func (c *ChaiAssertion) That() *ChaiAssertion { return c }

// Which is a pass-through getter for readability.
func (c *ChaiAssertion) Which() *ChaiAssertion { return c }

// And is a pass-through getter for readability.
func (c *ChaiAssertion) And() *ChaiAssertion { return c }

// Has is a pass-through getter for readability.
func (c *ChaiAssertion) Has() *ChaiAssertion { return c }

// Have is a pass-through getter for readability.
func (c *ChaiAssertion) Have() *ChaiAssertion { return c }

// With is a pass-through getter for readability.
func (c *ChaiAssertion) With() *ChaiAssertion { return c }

// At is a pass-through getter for readability.
func (c *ChaiAssertion) At() *ChaiAssertion { return c }

// Of is a pass-through getter for readability.
func (c *ChaiAssertion) Of() *ChaiAssertion { return c }

// Same is a pass-through getter for readability.
func (c *ChaiAssertion) Same() *ChaiAssertion { return c }

// But is a pass-through getter for readability.
func (c *ChaiAssertion) But() *ChaiAssertion { return c }

// Does is a pass-through getter for readability.
func (c *ChaiAssertion) Does() *ChaiAssertion { return c }

// Still is a pass-through getter for readability.
func (c *ChaiAssertion) Still() *ChaiAssertion { return c }

// Also is a pass-through getter for readability.
func (c *ChaiAssertion) Also() *ChaiAssertion { return c }

// --- negation ---

// Not flips the negation flag and returns the chain.
func (c *ChaiAssertion) Not() *ChaiAssertion {
	c.negated = !c.negated
	return c
}

// --- terminal methods ---

// Equal asserts that the actual value equals the expected value
// using reflect.DeepEqual.
func (c *ChaiAssertion) Equal(expected any) {
	actual := normalizeNumeric(c.actual)
	expected = normalizeNumeric(expected)
	ok := reflect.DeepEqual(actual, expected)
	c.check(ok, expected, c.actual, "equal")
}

// An asserts that the actual value is of the named type.
// Supported type names: "array", "object", "string", "number",
// "boolean", "null", "undefined".
func (c *ChaiAssertion) An(typeName string) {
	c.assertType(typeName)
}

// A is an alias for An.
func (c *ChaiAssertion) A(typeName string) {
	c.assertType(typeName)
}

// Property asserts that the actual value (expected to be a map) has
// the named key.
func (c *ChaiAssertion) Property(name string) {
	m, ok := toMap(c.actual)
	if !ok {
		c.fail(
			fmt.Sprintf("a map with property '%s'", name),
			c.actual,
			fmt.Sprintf("expected a map, got %T", c.actual),
		)
		return
	}
	_, exists := m[name]
	c.check(exists, fmt.Sprintf("property '%s'", name), formatKeys(m), "have property")
}

// Include asserts that the actual value contains the needle.
// For strings, it checks for a substring. For slices, it checks for
// element membership using reflect.DeepEqual.
func (c *ChaiAssertion) Include(needle any) {
	switch v := c.actual.(type) {
	case string:
		s, ok := needle.(string)
		if !ok {
			c.fail(needle, c.actual, "expected string needle for string include")
			return
		}
		c.check(strings.Contains(v, s), needle, c.actual, "include")
	default:
		rv := reflect.ValueOf(c.actual)
		if rv.Kind() == reflect.Slice {
			found := false
			for i := 0; i < rv.Len(); i++ {
				if reflect.DeepEqual(rv.Index(i).Interface(), needle) {
					found = true
					break
				}
			}
			c.check(found, needle, c.actual, "include")
		} else {
			c.fail(needle, c.actual, fmt.Sprintf("expected string or array for include, got %T", c.actual))
		}
	}
}

// Below asserts that the actual numeric value is less than limit.
func (c *ChaiAssertion) Below(limit float64) {
	n, ok := toFloat64(c.actual)
	if !ok {
		c.fail(limit, c.actual, fmt.Sprintf("expected a number, got %T", c.actual))
		return
	}
	c.check(n < limit, fmt.Sprintf("below %v", limit), n, "be below")
}

// Above asserts that the actual numeric value is greater than limit.
func (c *ChaiAssertion) Above(limit float64) {
	n, ok := toFloat64(c.actual)
	if !ok {
		c.fail(limit, c.actual, fmt.Sprintf("expected a number, got %T", c.actual))
		return
	}
	c.check(n > limit, fmt.Sprintf("above %v", limit), n, "be above")
}

// --- internal helpers ---

func (c *ChaiAssertion) assertType(typeName string) {
	got := goTypeLabel(c.actual)
	ok := got == strings.ToLower(typeName)
	c.check(ok, typeName, got, "be a")
}

func (c *ChaiAssertion) check(ok bool, expected, actual any, verb string) {
	if c.negated {
		ok = !ok
	}
	if !ok {
		neg := ""
		if c.negated {
			neg = "not "
		}
		msg := fmt.Sprintf("expected %v to %s%s %v", c.actual, neg, verb, expected)
		c.fail(expected, actual, msg)
	}
}

func (c *ChaiAssertion) fail(expected, actual any, message string) {
	panic(&testing.TestError{
		Expected: expected,
		Actual:   actual,
		Message:  message,
	})
}

// goTypeLabel returns a Chai-compatible type name for a Go value.
func goTypeLabel(v any) string {
	if v == nil {
		return "null"
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map:
		return "object"
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	default:
		return rv.Type().String()
	}
}

// normalizeNumeric converts integer types to float64 for consistent
// equality comparison between Go int64 (goja default) and float64.
func normalizeNumeric(v any) any {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	default:
		return v
	}
}

// toFloat64 converts a numeric value to float64.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

// toMap attempts to convert v to a map[string]any.
func toMap(v any) (map[string]any, bool) {
	if m, ok := v.(map[string]any); ok {
		return m, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map {
		return nil, false
	}
	m := make(map[string]any, rv.Len())
	for _, key := range rv.MapKeys() {
		m[fmt.Sprint(key.Interface())] = rv.MapIndex(key).Interface()
	}
	return m, true
}

// formatKeys returns a list of map keys for error messages.
func formatKeys(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return fmt.Sprintf("[%s]", strings.Join(keys, ", "))
}
