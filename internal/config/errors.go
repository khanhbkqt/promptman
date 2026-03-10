package config

import "errors"

// Sentinel errors for the config package.
var (
	// ErrConfigNotFound indicates the config file does not exist.
	ErrConfigNotFound = errors.New("config: file not found")

	// ErrInvalidConfig indicates the config file could not be parsed.
	ErrInvalidConfig = errors.New("config: invalid configuration")
)
