package grep

import (
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func testSetup(t *testing.T) *simulation.S {
	t.Helper()
	fs := &file.F{
		Type: file.Folder, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Files: map[string]*file.F{
			"log.txt": {Type: file.Text, Owner: "user",
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Data: "info: starting\nerror: disk full\ninfo: running\nerror: timeout"},
		},
	}
	sim := simulation.New()
	if err := sim.Boot("test", fs); err != nil {
		t.Fatal(err)
	}
	return sim
}

func stdout(t *testing.T, p process.P) string {
	t.Helper()
	out, _, _ := p.Stdout()
	var s string
	for _, d := range out {
		if c, ok := d.(process.Chars); ok {
			s += string(c)
		}
	}
	return s
}

func stderr(t *testing.T, p process.P) string {
	t.Helper()
	out, _, _ := p.Stderr()
	var s string
	for _, d := range out {
		if c, ok := d.(process.Chars); ok {
			s += string(c)
		}
	}
	return s
}

func TestGrepFile(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"error", "log.txt"})
	out := stdout(t, p)
	if !strings.Contains(out, "error: disk full") || !strings.Contains(out, "error: timeout") {
		t.Fatalf("expected error lines, got %q", out)
	}
	if strings.Contains(out, "info") {
		t.Fatalf("should not contain info lines, got %q", out)
	}
}

func TestGrepNoMatch(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"warning", "log.txt"})
	out := stdout(t, p)
	if out != "" {
		t.Fatalf("expected empty output, got %q", out)
	}
}

func TestGrepMissingPattern(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, nil)
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "missing pattern") {
		t.Fatalf("expected missing pattern error, got %q", errOut)
	}
}

func TestGrepStdin(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"hello"})
	p.Stdin(process.CharsData("hello world\ngoodbye\nhello again"))
	p.Kill() // Signal end of input
	out := stdout(t, p)
	if !strings.Contains(out, "hello world") {
		t.Fatalf("expected 'hello world', got %q", out)
	}
	if strings.Contains(out, "goodbye") {
		t.Fatalf("should not contain 'goodbye', got %q", out)
	}
}
