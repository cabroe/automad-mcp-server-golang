package main

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
)

func TestParseFlagsVersion(t *testing.T) {
	// Both spellings must work: Go's flag package accepts either, and users
	// reach for the double dash.
	for _, arg := range []string{"--version", "-version"} {
		var out, errOut bytes.Buffer
		code, stop := parseFlags([]string{arg}, &out, &errOut)
		if code != 0 || !stop {
			t.Errorf("%s: code=%d stop=%v, want 0/true", arg, code, stop)
		}
		line := strings.TrimSpace(out.String())
		if !strings.HasPrefix(line, commandName+" ") {
			t.Errorf("%s printed %q, want it to start with the command name", arg, line)
		}
		if !strings.Contains(line, runtime.GOOS+"/"+runtime.GOARCH) {
			t.Errorf("%s printed %q, want the platform in it", arg, line)
		}
		if errOut.Len() != 0 {
			t.Errorf("%s wrote to stderr: %q", arg, errOut.String())
		}
	}
}

func TestParseFlagsRunsServerByDefault(t *testing.T) {
	var out, errOut bytes.Buffer
	code, stop := parseFlags(nil, &out, &errOut)
	if stop || code != 0 {
		t.Fatalf("no arguments: code=%d stop=%v, want 0/false so the server starts", code, stop)
	}
	if out.Len() != 0 {
		// stdout carries the MCP protocol; nothing else may be written there.
		t.Errorf("stdout polluted before startup: %q", out.String())
	}
}

func TestParseFlagsRejectsUnknownInput(t *testing.T) {
	cases := map[string][]string{
		"unknown flag":       {"--nope"},
		"unexpected operand": {"serve"},
	}
	for name, args := range cases {
		var out, errOut bytes.Buffer
		code, stop := parseFlags(args, &out, &errOut)
		if code != 2 || !stop {
			t.Errorf("%s: code=%d stop=%v, want 2/true", name, code, stop)
		}
		if errOut.Len() == 0 {
			t.Errorf("%s: nothing explained on stderr", name)
		}
		if out.Len() != 0 {
			t.Errorf("%s: wrote to stdout: %q", name, out.String())
		}
	}
}

func TestParseFlagsHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	code, stop := parseFlags([]string{"-h"}, &out, &errOut)
	if code != 0 || !stop {
		t.Errorf("-h: code=%d stop=%v, want 0/true", code, stop)
	}
	if !strings.Contains(errOut.String(), "Usage:") {
		t.Errorf("-h did not print usage, got %q", errOut.String())
	}
}

func TestVersionLineReportsBuildVersion(t *testing.T) {
	original := version
	t.Cleanup(func() { version = original })

	version = "1.2.3"
	if got := versionLine(); !strings.HasPrefix(got, commandName+" 1.2.3 (") {
		t.Errorf("versionLine() = %q, want the ldflag version in it", got)
	}
}
