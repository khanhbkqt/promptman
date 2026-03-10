package cli

import "context"

// contextKey is the unexported type used for CLI context keys to avoid collisions.
type contextKey int

const (
	// globalFlagsKey is the context key for the GlobalFlags value.
	globalFlagsKey contextKey = iota
)

// withGlobalFlags returns a new context that carries the given GlobalFlags.
func withGlobalFlags(ctx context.Context, flags *GlobalFlags) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, globalFlagsKey, flags)
}

// GlobalFlagsFrom retrieves the GlobalFlags stored in the context.
// Returns nil if no flags are present.
func GlobalFlagsFrom(ctx context.Context) *GlobalFlags {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(globalFlagsKey).(*GlobalFlags)
	return v
}
