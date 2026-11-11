package process

import (
	"reflect"
	"testing"
)

// mockProcess is a minimal process for testing Space.
type mockProcess struct {
	Written
	owner  string
	caught Signal // a signal this process handles instead of dying
	got    []Signal
}

func (p *mockProcess) Stdin(Data) error { return nil }
func (p *mockProcess) Next() []Datum    { return nil }
func (p *mockProcess) Owner() string    { return p.owner }
func (p *mockProcess) Signal(sig Signal) {
	p.got = append(p.got, sig)
	if sig != p.caught {
		p.Written.Signal(sig)
	}
}

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

func TestSpaceSignal(t *testing.T) {
	s := NewSpace()
	p := &mockProcess{owner: "user"}
	id := s.Add(p, "cat")
	if err := s.Signal(id, SigTerm); err != nil {
		t.Fatal(err)
	}
	if status, exited := p.Exit(); !exited || status != 143 {
		t.Fatalf("after SIGTERM: status %d exited %v, want 143", status, exited)
	}
	if len(s.List()) != 0 {
		t.Fatal("process should be removed from list")
	}
	if err := s.Signal(id, SigKill); err != ErrNoProcess {
		t.Fatalf("killing it again: %v, want ErrNoProcess", err)
	}
}

// A process that handles a signal stays in the table.
func TestSpaceSignalCaught(t *testing.T) {
	s := NewSpace()
	p := &mockProcess{owner: "user", caught: SigTerm}
	id := s.Add(p, "shell")
	if err := s.Signal(id, SigTerm); err != nil {
		t.Fatal(err)
	}
	if _, exited := p.Exit(); exited || len(s.List()) != 1 {
		t.Fatal("a process that caught SIGTERM died")
	}
	if err := s.Signal(id, SigKill); err != nil {
		t.Fatal(err)
	}
	if status, _ := p.Exit(); status != KilledStatus || len(s.List()) != 0 {
		t.Fatalf("SIGKILL: status %d, %d left", status, len(s.List()))
	}
}

func TestSignalStatus(t *testing.T) {
	for sig, want := range map[Signal]int{SigHup: 129, SigInt: 130, SigKill: 137, SigPipe: 141, SigTerm: 143} {
		if got := sig.Status(); got != want {
			t.Errorf("%v.Status() = %d, want %d", sig, got, want)
		}
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
