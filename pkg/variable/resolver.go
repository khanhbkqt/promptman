package variable

import (
	"fmt"
	"reflect"
	"strings"
)

const defaultMaxDepth = 5

// Options configures the resolution behavior.
type Options struct {
	// Strict mode: if true, missing variables return an error.
	// If false (lenient), missing variables are preserved as {{varName}}.
	Strict bool

	// MaxDepth limits recursive resolution depth. Default: 5.
	// Set to 0 to use the default.
	MaxDepth int
}

func (o Options) maxDepth() int {
	if o.MaxDepth <= 0 {
		return defaultMaxDepth
	}
	return o.MaxDepth
}

// Resolve resolves all {{variable}} references in a template string using the provided
// variable map. It supports recursive resolution (variables pointing to other variables)
// up to MaxDepth levels.
//
// In strict mode, a missing variable returns ErrVariableNotFound.
// In lenient mode, a missing variable preserves the {{varName}} text.
func Resolve(template string, vars map[string]any, opts ...Options) (string, error) {
	opt := Options{}
	if len(opts) > 0 {
		opt = opts[0]
	}
	return resolveWithDepth(template, vars, opt, 0)
}

func resolveWithDepth(template string, vars map[string]any, opts Options, depth int) (string, error) {
	if depth > opts.maxDepth() {
		return "", &ErrMaxDepthExceeded{MaxDepth: opts.maxDepth()}
	}

	tokens := parse(template)
	var builder strings.Builder
	needsResolve := false

	for _, tok := range tokens {
		if !tok.isVar {
			builder.WriteString(tok.value)
			continue
		}

		val, exists := vars[tok.value]
		if !exists {
			if opts.Strict {
				return "", &ErrVariableNotFound{Name: tok.value}
			}
			// Lenient: preserve the original {{varName}}
			builder.WriteString("{{")
			builder.WriteString(tok.value)
			builder.WriteString("}}")
			continue
		}

		str := fmt.Sprintf("%v", val)
		builder.WriteString(str)

		// Check if resolved value contains more variables
		if containsVariable(str) {
			needsResolve = true
		}
	}

	result := builder.String()

	// Recursive resolution: if the result contains more variables, resolve again
	if needsResolve {
		return resolveWithDepth(result, vars, opts, depth+1)
	}

	return result, nil
}

// ResolveStruct resolves all string fields in a struct (and nested structs)
// using the provided variable map. It modifies the struct in place.
// The obj parameter must be a pointer to a struct.
func ResolveStruct(obj any, vars map[string]any, opts ...Options) error {
	opt := Options{}
	if len(opts) > 0 {
		opt = opts[0]
	}
	return resolveValue(reflect.ValueOf(obj), vars, opt)
}

func resolveValue(v reflect.Value, vars map[string]any, opts Options) error {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return resolveValue(v.Elem(), vars, opts)

	case reflect.Struct:
		for i := range v.NumField() {
			field := v.Field(i)
			if !field.CanSet() {
				continue
			}
			if err := resolveValue(field, vars, opts); err != nil {
				return err
			}
		}

	case reflect.String:
		if !v.CanSet() {
			return nil
		}
		resolved, err := Resolve(v.String(), vars, opts)
		if err != nil {
			return err
		}
		v.SetString(resolved)

	case reflect.Slice:
		for i := range v.Len() {
			if err := resolveValue(v.Index(i), vars, opts); err != nil {
				return err
			}
		}

	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			val := iter.Value()
			if val.Kind() == reflect.String {
				resolved, err := Resolve(val.String(), vars, opts)
				if err != nil {
					return err
				}
				v.SetMapIndex(iter.Key(), reflect.ValueOf(resolved))
			}
		}
	}

	return nil
}
