package guide

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// mockProcess implements process.P for testing the guide.
type mockProcess struct {
	process.Written
	valid []process.Datum
	stdin process.Data
}

func (p *mockProcess) Stdin(in process.Data) error {
	p.stdin = append(p.stdin, in...)
	return nil
}

func (p *mockProcess) Next() []process.Datum {
	return p.valid
}

func (p *mockProcess) Owner() string { return "user" }

func TestGuideAllowsValidInput(t *testing.T) {
	p := &mockProcess{
		valid: []process.Datum{process.Chars("a"), process.Chars("b")},
	}
	g := New(p)

	err := g.Stdin(process.Data{process.Chars("a")})
	if err != nil {
		t.Fatalf("expected valid input to pass, got %v", err)
	}
	if len(p.stdin) != 1 {
		t.Fatal("expected input to reach process")
	}
}

func TestGuideBlocksInvalidInput(t *testing.T) {
	p := &mockProcess{
		valid: []process.Datum{process.Chars("a"), process.Chars("b")},
	}
	g := New(p)

	err := g.Stdin(process.Data{process.Chars("z")})
	if err == nil {
		t.Fatal("expected invalid input to be blocked")
	}
	if len(p.stdin) != 0 {
		t.Fatal("invalid input should not reach process")
	}
}

func TestGuideAllowsTermCode(t *testing.T) {
	p := &mockProcess{
		valid: []process.Datum{process.TermEnter, process.TermBackspace},
	}
	g := New(p)

	err := g.Stdin(process.Data{process.TermEnter})
	if err != nil {
		t.Fatalf("expected TermEnter to pass, got %v", err)
	}
}

func TestGuideBlocksTermCode(t *testing.T) {
	p := &mockProcess{
		valid: []process.Datum{process.Chars("a")},
	}
	g := New(p)

	err := g.Stdin(process.Data{process.TermEnter})
	if err == nil {
		t.Fatal("expected TermEnter to be blocked when not in Next()")
	}
}

func TestGuideRejectsMultipleDatums(t *testing.T) {
	p := &mockProcess{
		valid: []process.Datum{process.Chars("a")},
	}
	g := New(p)

	err := g.Stdin(process.Data{process.Chars("a"), process.Chars("b")})
	if err == nil {
		t.Fatal("expected error for multiple datums")
	}
}

func TestGuidePassthroughOutput(t *testing.T) {
	p := &mockProcess{}
	p.Write(process.Stdout, process.CharsData("output"))
	p.Write(process.Stderr, process.CharsData("error"))
	p.Finish(3)
	g := New(p)

	out := g.Read()
	if len(out) != 2 || out[0].Data.String() != "output" || out[1].Stream != process.Stderr {
		t.Fatalf("expected output passthrough in order, got %v", out)
	}
	if status, exited := g.Exit(); !exited || status != 3 {
		t.Fatalf("expected exit passthrough, got %v %v", status, exited)
	}
}

func TestGuideNextPassthrough(t *testing.T) {
	p := &mockProcess{
		valid: []process.Datum{process.Chars("x"), process.TermEnter},
	}
	g := New(p)

	next := g.Next()
	if len(next) != 2 {
		t.Fatalf("expected 2 valid inputs, got %d", len(next))
	}
}

func TestGuideEmptyNext(t *testing.T) {
	p := &mockProcess{valid: nil}
	g := New(p)

	err := g.Stdin(process.Data{process.Chars("a")})
	if err == nil {
		t.Fatal("expected everything to be blocked when Next() is empty")
	}
}
