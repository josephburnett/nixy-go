package guide

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// mockProcess implements process.P for testing the guide.
type mockProcess struct {
	valid  []process.Datum
	stdin  process.Data
	stdout process.Data
	stderr process.Data
}

func (p *mockProcess) Stdout() (process.Data, bool, error) {
	d := p.stdout
	p.stdout = nil
	return d, false, nil
}

func (p *mockProcess) Stderr() (process.Data, bool, error) {
	d := p.stderr
	p.stderr = nil
	return d, false, nil
}

func (p *mockProcess) Stdin(in process.Data) (bool, error) {
	p.stdin = append(p.stdin, in...)
	return false, nil
}

func (p *mockProcess) Next() []process.Datum {
	return p.valid
}

func (p *mockProcess) Owner() string { return "user" }
func (p *mockProcess) Kill() error   { return nil }

func TestGuideAllowsValidInput(t *testing.T) {
	p := &mockProcess{
		valid: []process.Datum{process.Chars("a"), process.Chars("b")},
	}
	g := New(p)

	_, err := g.Stdin(process.Data{process.Chars("a")})
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

	_, err := g.Stdin(process.Data{process.Chars("z")})
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

	_, err := g.Stdin(process.Data{process.TermEnter})
	if err != nil {
		t.Fatalf("expected TermEnter to pass, got %v", err)
	}
}

func TestGuideBlocksTermCode(t *testing.T) {
	p := &mockProcess{
		valid: []process.Datum{process.Chars("a")},
	}
	g := New(p)

	_, err := g.Stdin(process.Data{process.TermEnter})
	if err == nil {
		t.Fatal("expected TermEnter to be blocked when not in Next()")
	}
}

func TestGuideRejectsMultipleDatums(t *testing.T) {
	p := &mockProcess{
		valid: []process.Datum{process.Chars("a")},
	}
	g := New(p)

	_, err := g.Stdin(process.Data{process.Chars("a"), process.Chars("b")})
	if err == nil {
		t.Fatal("expected error for multiple datums")
	}
}

func TestGuidePassthroughStdout(t *testing.T) {
	p := &mockProcess{
		stdout: process.Data{process.Chars("output")},
	}
	g := New(p)

	out, _, _ := g.Stdout()
	if len(out) != 1 || string(out[0].(process.Chars)) != "output" {
		t.Fatalf("expected stdout passthrough, got %v", out)
	}
}

func TestGuidePassthroughStderr(t *testing.T) {
	p := &mockProcess{
		stderr: process.Data{process.Chars("error")},
	}
	g := New(p)

	out, _, _ := g.Stderr()
	if len(out) != 1 || string(out[0].(process.Chars)) != "error" {
		t.Fatalf("expected stderr passthrough, got %v", out)
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

	_, err := g.Stdin(process.Data{process.Chars("a")})
	if err == nil {
		t.Fatal("expected everything to be blocked when Next() is empty")
	}
}
