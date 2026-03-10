package variable

import "fmt"

// ErrVariableNotFound indicates a referenced variable was not found in the variable map.
type ErrVariableNotFound struct {
	Name string
}

func (e *ErrVariableNotFound) Error() string {
	return fmt.Sprintf("variable not found: {{%s}}", e.Name)
}

// ErrMaxDepthExceeded indicates recursive variable resolution exceeded the maximum depth.
type ErrMaxDepthExceeded struct {
	MaxDepth int
}

func (e *ErrMaxDepthExceeded) Error() string {
	return fmt.Sprintf("maximum resolution depth (%d) exceeded — possible circular reference", e.MaxDepth)
}
