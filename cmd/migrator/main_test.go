package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	buf := &bytes.Buffer{}
	printVersion(buf)

	if !strings.Contains(buf.String(), "go-migration") {
		t.Errorf("%q not in %q", "go-migration", buf.String())
	}
	if !strings.Contains(buf.String(), Version) {
		t.Errorf("%q not in %q", Version, buf.String())
	}
	if !strings.Contains(buf.String(), BuildDate) {
		t.Errorf("%q not in %q", BuildDate, buf.String())
	}
}

func TestVersionCommandOutput(t *testing.T) {
	origVersion := Version
	origBuild := BuildDate
	Version = "v1.2.3"
	BuildDate = "2024-06-15"
	defer func() {
		Version = origVersion
		BuildDate = origBuild
	}()

	buf := &bytes.Buffer{}
	printVersion(buf)

	if "go-migration v1.2.3 (built 2024-06-15)\n" != buf.String() {
		t.Errorf("got %v, want %v", buf.String(), "go-migration v1.2.3 (built 2024-06-15)\n")
	}
}
