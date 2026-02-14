package main

import (
	"fmt"
	"os"

	"codegate/cmd"
)

// Version information - set via build flags
// Example: go build -ldflags "-X main.version=1.0.0 -X main.commit=abc123 -X main.date=2026-01-01"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Set version information for the CLI
	cmd.SetVersionInfo(version, commit, date)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
