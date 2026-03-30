package pwd

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
)

func TestPwdRoot(t *testing.T) {
	p := command.NewSingleValueProcess("user", "/\n")
	out, eof, err := p.Stdout()
	if err != nil {
		t.Fatal(err)
	}
	if !eof {
		t.Fatal("expected eof")
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 datum, got %v", len(out))
	}
	if string(out[0].(process.Chars)) != "/\n" {
		t.Fatalf("expected '/\\n', got %v", out[0])
	}
}

func TestPwdNested(t *testing.T) {
	p, err := launch(nil, "user", "host", []string{"home", "user"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out, eof, err := p.Stdout()
	if err != nil {
		t.Fatal(err)
	}
	if !eof {
		t.Fatal("expected eof")
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 datum, got %v", len(out))
	}
	if string(out[0].(process.Chars)) != "/home/user\n" {
		t.Fatalf("expected '/home/user\\n', got %v", out[0])
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
	next := p.Next()
	if len(next) != 0 {
		t.Fatalf("expected no valid inputs, got %v", next)
	}
}

func TestPwdStderrEmpty(t *testing.T) {
	p, err := launch(nil, "user", "host", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, eof, _ := p.Stderr()
	if !eof {
		t.Fatal("expected stderr eof")
	}
}
