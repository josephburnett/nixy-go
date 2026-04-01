package process

import (
	"testing"
)

// mockProcess is a minimal process for testing Space.
type mockProcess struct {
	owner  string
	killed bool
}

func (p *mockProcess) Stdout() (Data, bool, error)  { return nil, false, nil }
func (p *mockProcess) Stderr() (Data, bool, error)  { return nil, false, nil }
func (p *mockProcess) Stdin(_ Data) (bool, error)    { return false, nil }
func (p *mockProcess) Next() []Datum                 { return nil }
func (p *mockProcess) Owner() string                 { return p.owner }
func (p *mockProcess) Kill() error                   { p.killed = true; return nil }

func TestSpaceAddAndList(t *testing.T) {
	s := NewSpace()
	p1 := &mockProcess{owner: "a"}
	p2 := &mockProcess{owner: "b"}

	id1 := s.Add(p1)
	id2 := s.Add(p2)

	if id1 == id2 {
		t.Fatal("IDs should be unique")
	}

	list := s.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 processes, got %d", len(list))
	}
	if list[id1].Owner() != "a" {
		t.Fatal("wrong process for id1")
	}
	if list[id2].Owner() != "b" {
		t.Fatal("wrong process for id2")
	}
}

func TestSpaceKill(t *testing.T) {
	s := NewSpace()
	p := &mockProcess{owner: "user"}
	id := s.Add(p)

	err := s.Kill(id)
	if err != nil {
		t.Fatal(err)
	}
	if !p.killed {
		t.Fatal("process should be killed")
	}
	if len(s.List()) != 0 {
		t.Fatal("process should be removed from list")
	}
}

func TestSpaceKillInvalid(t *testing.T) {
	s := NewSpace()
	err := s.Kill(999)
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

func TestSpaceIDsIncrement(t *testing.T) {
	s := NewSpace()
	id1 := s.Add(&mockProcess{})
	id2 := s.Add(&mockProcess{})
	id3 := s.Add(&mockProcess{})
	if id1 != 0 || id2 != 1 || id3 != 2 {
		t.Fatalf("expected sequential IDs 0,1,2 — got %d,%d,%d", id1, id2, id3)
	}
}

func TestCharsData(t *testing.T) {
	d := CharsData("hello")
	if len(d) != 1 {
		t.Fatalf("expected 1 datum, got %d", len(d))
	}
	if c, ok := d[0].(Chars); !ok || string(c) != "hello" {
		t.Fatalf("expected Chars('hello'), got %v", d[0])
	}
}
