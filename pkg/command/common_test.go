package command

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
)

func TestSingleValueProcess(t *testing.T) {
	p := NewSingleValueProcess("alice", "hello\n")
	if status, exited := p.Exit(); !exited || status != 0 {
		t.Fatalf("expected exit 0, got %v %v", status, exited)
	}
	out := p.Read()
	if got := process.Text(out, process.Stdout); got != "hello\n" {
		t.Fatalf("stdout %q", got)
	}
	if got := process.Text(out, process.Stderr); got != "" {
		t.Fatalf("stderr %q", got)
	}
	if len(p.Read()) != 0 {
		t.Fatal("second read should be empty")
	}
	if p.Owner() != "alice" || len(p.Next()) != 0 {
		t.Fatal("owner or next wrong")
	}
	if err := p.Stdin(process.CharsData("input")); err != process.ErrNoInput {
		t.Fatalf("expected ErrNoInput, got %v", err)
	}
}

func TestSingleValueProcessEmpty(t *testing.T) {
	if out := NewSingleValueProcess("user", "").Read(); len(out) != 0 {
		t.Fatalf("no output expected, got %v", out)
	}
}

func TestErrorProcess(t *testing.T) {
	p := NewErrorProcess("bob", "something went wrong\n")
	if status, exited := p.Exit(); !exited || status != 1 {
		t.Fatalf("expected exit 1, got %v %v", status, exited)
	}
	out := p.Read()
	if got := process.Text(out, process.Stderr); got != "something went wrong\n" {
		t.Fatalf("stderr %q", got)
	}
	if got := process.Text(out, process.Stdout); got != "" {
		t.Fatalf("stdout %q", got)
	}
	if p.Owner() != "bob" {
		t.Fatal("owner wrong")
	}
}

func TestExitedKeepsOrderAndIgnoresCloseStdin(t *testing.T) {
	p := Exited("user", "out\n", "err\n", 2)
	p.CloseStdin()
	out := p.Read()
	if len(out) != 2 || out[0].Stream != process.Stderr || out[1].Stream != process.Stdout {
		t.Fatalf("expected stderr then stdout, got %v", out)
	}
	if status, _ := p.Exit(); status != 2 {
		t.Fatalf("status %v", status)
	}
}

func TestKillDropsOutput(t *testing.T) {
	p := NewErrorProcess("user", "err")
	p.Kill()
	if out := p.Read(); len(out) != 0 {
		t.Fatalf("expected no output after kill, got %v", out)
	}
}
