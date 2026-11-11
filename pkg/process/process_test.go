package process

import (
	"reflect"
	"testing"
)

// mockProcess is a minimal process for testing Space.
type mockProcess struct {
	Written
	owner  string
	killed bool
}

func (p *mockProcess) Stdin(Data) error { return nil }
func (p *mockProcess) Next() []Datum    { return nil }
func (p *mockProcess) Owner() string    { return p.owner }
func (p *mockProcess) Kill()            { p.killed = true; p.Written.Kill() }

func TestSpaceAddAndList(t *testing.T) {
	s := NewSpace()
	id1 := s.Add(&mockProcess{owner: "a"}, "ls")
	id2 := s.Add(&mockProcess{owner: "b"}, "shell -l")
	want := []Info{{PID: 1, Owner: "a", Command: "ls"}, {PID: 2, Owner: "b", Command: "shell -l"}}
	if id1 != 1 || id2 != 2 {
		t.Fatalf("PIDs %d %d, want 1 2", id1, id2)
	}
	if got := s.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("List = %v, want %v", got, want)
	}
}

func TestSpaceDropsExited(t *testing.T) {
	s := NewSpace()
	done := &mockProcess{owner: "a"}
	s.Add(done, "ls")
	s.Add(&mockProcess{owner: "b"}, "shell")
	done.Finish(0)
	if got := s.List(); len(got) != 1 || got[0].Command != "shell" {
		t.Fatalf("exited process still listed: %v", got)
	}
	if _, ok := s.Get(1); ok {
		t.Fatal("Get found an exited process")
	}
	if id := s.Add(&mockProcess{}, "x"); id != 3 {
		t.Fatalf("PIDs are not reused: got %d", id)
	}
}

func TestSpaceKill(t *testing.T) {
	s := NewSpace()
	p := &mockProcess{owner: "user"}
	id := s.Add(p, "cat")
	if err := s.Kill(id); err != nil {
		t.Fatal(err)
	}
	if !p.killed {
		t.Fatal("process should be killed")
	}
	if len(s.List()) != 0 {
		t.Fatal("process should be removed from list")
	}
	if err := s.Kill(id); err != ErrNoProcess {
		t.Fatalf("killing it again: %v, want ErrNoProcess", err)
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
