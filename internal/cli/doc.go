// Package cli implements the Promptman CLI thin client.
// It provides a Cobra-based command tree, daemon discovery and auto-start,
// an HTTP client with Bearer authentication, and a pluggable output formatter
// (json/table/minimal). All business logic executes in the daemon process;
// the CLI is a thin HTTP client that dispatches requests and formats responses.
package cli
