// Package main is the entry point for the Promptman CLI.
// The CLI is a Cobra-based thin client that communicates with the daemon.
package main

import "github.com/khanhnguyen/promptman/internal/cli"

func main() {
	cli.Execute()
}
