package cli

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs fn with os.Stdout swapped for a pipe and returns what
// was written. The version command prints via fmt.Println, which targets
// os.Stdout directly.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestVersionCommand(t *testing.T) {
	root := newRootCmd()
	out := captureStdout(t, func() {
		root.SetArgs([]string{"version"})
		if err := root.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	if !strings.HasPrefix(out, "thoth ") {
		t.Fatalf("unexpected version output %q", out)
	}
	if !strings.Contains(out, version) {
		t.Fatalf("version output %q missing %q", out, version)
	}
}

func TestRootRejectsUnknownCommand(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"bogus-command"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
}

func TestExecuteReturnsErrorForUnknownCommand(t *testing.T) {
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"thoth", "bogus"}
	if err := Execute(); err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
}

func TestRootHasExpectedSubcommands(t *testing.T) {
	root := newRootCmd()
	names := map[string]bool{}
	for _, c := range root.Commands() {
		names[c.Name()] = true
	}
	for _, want := range []string{"serve", "init", "version"} {
		if !names[want] {
			t.Fatalf("root command missing %q (have %v)", want, names)
		}
	}
}
