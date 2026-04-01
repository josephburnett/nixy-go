package command

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
)

func TestSingleValueProcessStdout(t *testing.T) {
	p := NewSingleValueProcess("user", "hello\n")
	out, eof, err := p.Stdout()
	if err != nil {
		t.Fatal(err)
	}
	if !eof {
		t.Fatal("expected eof on first read")
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 datum, got %d", len(out))
	}
	if string(out[0].(process.Chars)) != "hello\n" {
		t.Fatalf("expected 'hello\\n', got %v", out[0])
	}
}

func TestSingleValueProcessStdoutSecondRead(t *testing.T) {
	p := NewSingleValueProcess("user", "data")
	p.Stdout() // first read
	_, eof, err := p.Stdout()
	if err != ErrEndOfFile {
		t.Fatalf("expected ErrEndOfFile, got %v", err)
	}
	if !eof {
		t.Fatal("expected eof")
	}
}

func TestSingleValueProcessStderr(t *testing.T) {
	p := NewSingleValueProcess("user", "data")
	_, eof, _ := p.Stderr()
	if !eof {
		t.Fatal("stderr should always be eof")
	}
}

func TestSingleValueProcessStdin(t *testing.T) {
	p := NewSingleValueProcess("user", "data")
	_, err := p.Stdin(process.CharsData("input"))
	if err != ErrReadOnlyProcess {
		t.Fatalf("expected ErrReadOnlyProcess, got %v", err)
	}
}

func TestSingleValueProcessNext(t *testing.T) {
	p := NewSingleValueProcess("user", "data")
	if len(p.Next()) != 0 {
		t.Fatal("expected no valid inputs")
	}
}

func TestSingleValueProcessOwner(t *testing.T) {
	p := NewSingleValueProcess("alice", "data")
	if p.Owner() != "alice" {
		t.Fatalf("expected 'alice', got %q", p.Owner())
	}
}

func TestSingleValueProcessKill(t *testing.T) {
	p := NewSingleValueProcess("user", "data")
	p.Kill()
	_, eof, _ := p.Stdout()
	if !eof {
		t.Fatal("expected eof after kill")
	}
}

func TestErrorProcessStderr(t *testing.T) {
	p := NewErrorProcess("user", "something went wrong\n")
	out, eof, _ := p.Stderr()
	if !eof {
		t.Fatal("expected eof")
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 datum, got %d", len(out))
	}
	if string(out[0].(process.Chars)) != "something went wrong\n" {
		t.Fatalf("expected error message, got %v", out[0])
	}
}

func TestErrorProcessStderrSecondRead(t *testing.T) {
	p := NewErrorProcess("user", "err")
	p.Stderr() // first read
	out, eof, _ := p.Stderr()
	if !eof {
		t.Fatal("expected eof")
	}
	if len(out) != 0 {
		t.Fatal("expected empty on second read")
	}
}

func TestErrorProcessStdout(t *testing.T) {
	p := NewErrorProcess("user", "err")
	_, eof, _ := p.Stdout()
	if !eof {
		t.Fatal("stdout should always be eof")
	}
}

func TestErrorProcessStdin(t *testing.T) {
	p := NewErrorProcess("user", "err")
	_, err := p.Stdin(process.CharsData("input"))
	if err != ErrReadOnlyProcess {
		t.Fatalf("expected ErrReadOnlyProcess, got %v", err)
	}
}

func TestErrorProcessOwner(t *testing.T) {
	p := NewErrorProcess("bob", "err")
	if p.Owner() != "bob" {
		t.Fatalf("expected 'bob', got %q", p.Owner())
	}
}

func TestErrorProcessKill(t *testing.T) {
	p := NewErrorProcess("user", "err")
	p.Kill()
	out, _, _ := p.Stderr()
	if len(out) != 0 {
		t.Fatal("expected empty stderr after kill")
	}
}
