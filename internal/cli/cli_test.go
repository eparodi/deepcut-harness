package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if got := stdout.String(); !strings.HasPrefix(got, "harness ") {
		t.Fatalf("version output = %q, want prefix \"harness \"", got)
	}
}

func TestUnknownCommandPrintsUsageAndExitsTwo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"nope"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "usage: harness") {
		t.Fatalf("usage not printed on unknown command: %q", stderr.String())
	}
}
