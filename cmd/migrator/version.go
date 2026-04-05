package main

import (
	"fmt"
	"io"
)

// Version and BuildDate are set at build time via ldflags:
//
//	go build -ldflags "-X main.Version=v1.0.0 -X main.BuildDate=2024-01-01"
var (
	Version   = "dev"
	BuildDate = "unknown"
)

// printVersion writes the version and build date to the given writer.
func printVersion(w io.Writer) {
	fmt.Fprintf(w, "go-migration %s (built %s)\n", Version, BuildDate)
}
