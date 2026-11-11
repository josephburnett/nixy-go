package pwd

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
)

func run(t *testing.T, cwd []string) []process.Output {
	t.Helper()
	p, err := launch(nil, "user", "host", cwd, nil)
	if err != nil {
		t.Fatal(err)
	}
	if status, exited := p.Exit(); !exited || status != 0 {
		t.Fatalf("pwd should exit 0 at once, got %v %v", status, exited)
	}
	return p.Read()
}

func TestPwdRoot(t *testing.T) {
	if got := process.Text(run(t, nil), process.Stdout); got != "/\n" {
		t.Fatalf("expected '/\\n', got %q", got)
	}
}

func TestPwdNested(t *testing.T) {
	if got := process.Text(run(t, []string{"home", "user"}), process.Stdout); got != "/home/user\n" {
		t.Fatalf("expected '/home/user\\n', got %q", got)
	}
}

func TestPwdRejectsArgs(t *testing.T) {
	_, err := launch(nil, "user", "host", nil, []string{"foo"})
	if err == nil {
		t.Fatal("expected error when args given")
	}
}

func TestPwdNext(t *testing.T) {
	p, err := launch(nil, "user", "host", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if next := p.Next(); len(next) != 0 {
		t.Fatalf("expected no valid inputs, got %v", next)
	}
}

func TestPwdStderrEmpty(t *testing.T) {
	if got := process.Text(run(t, nil), process.Stderr); got != "" {
		t.Fatalf("expected no stderr, got %q", got)
	}
}
