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
	return process.Text(p.Read(), process.Stdout)
}

func stderr(t *testing.T, p process.P) string {
	t.Helper()
	return process.Text(p.Read(), process.Stderr)
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
	if !strings.Contains(errOut, "Usage: grep") {
		t.Fatalf("expected usage error, got %q", errOut)
	}
}

func TestGrepStdin(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"hello"})
	p.Stdin(process.CharsData("hello world\ngoodbye\nhello again"))
	p.CloseStdin()
	out := stdout(t, p)
	if !strings.Contains(out, "hello world") {
		t.Fatalf("expected 'hello world', got %q", out)
	}
	if strings.Contains(out, "goodbye") {
		t.Fatalf("should not contain 'goodbye', got %q", out)
	}
}

// Exit status is 0 on a match, 1 on none, 2 on trouble, as in grep(1).
func TestGrepStatus(t *testing.T) {
	sim := testSetup(t)
	for _, tc := range []struct {
		args []string
		want int
	}{
		{[]string{"error", "log.txt"}, 0},
		{[]string{"warning", "log.txt"}, 1},
		{[]string{"error", "missing.txt"}, 2},
		{nil, 2},
	} {
		p, _ := launch(sim, "user", "test", nil, tc.args)
		if status, _ := p.Exit(); status != tc.want {
			t.Errorf("grep %v: status %d, want %d", tc.args, status, tc.want)
		}
	}
}
