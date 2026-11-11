package shell

import (
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// run types line and Enter into the shell, without the keyboard's
// checks, and returns everything written until the shell goes quiet.
func run(t *testing.T, p process.P, line string) []process.Output {
	t.Helper()
	writeString(t, p, line)
	writeEnter(t, p)
	var out []process.Output
	for range 50 {
		o := p.Read()
		if len(o) == 0 {
			break
		}
		out = append(out, o...)
	}
	// Drop the echo of what was typed.
	for len(out) > 0 && out[0].Stream == process.Stdout {
		if out[0].Data.String() == "" || strings.Contains(line+"\n", out[0].Data.String()) {
			out = out[1:]
			continue
		}
		break
	}
	return out
}

// loginSetup is testSetup with a login shell, in /home/user.
func loginSetup(t *testing.T) (*simulation.S, process.P) {
	t.Helper()
	sim, _ := testSetup(t)
	p, err := sim.Login("test", "user")
	if err != nil {
		t.Fatal(err)
	}
	return sim, p
}

func lastStatus(t *testing.T, p process.P) int {
	t.Helper()
	status, ok := p.(Info).LastStatus()
	if !ok {
		t.Fatal("no exit status: the command line hasn't finished")
	}
	return status
}

// TestPipelineReceiverOutputNotLost: closing a stage's stdin is not
// killing it. A receiver that ignores stdin still prints, and its
// output and errors are what it would write alone. (The keyboard no
// longer offers these pipelines; the shell must still run them right.)
func TestPipelineReceiverOutputNotLost(t *testing.T) {
	for _, tc := range []struct{ line, alone string }{
		{"ls | ls", "ls"},
		{"ls | pwd", "pwd"},
		{"cat notes.txt | ls docs", "ls docs"},
		{"ls | ls /root", "ls /root"},
		{"ls | cat nope", "cat nope"},
	} {
		_, p := loginSetup(t)
		want := run(t, p, tc.alone)
		got := run(t, p, tc.line)
		if process.Text(got, process.Stdout) != process.Text(want, process.Stdout) ||
			process.Text(got, process.Stderr) != process.Text(want, process.Stderr) {
			t.Errorf("%q wrote %v, but %q alone writes %v", tc.line, got, tc.alone, want)
		}
	}
}

// TestPipelineStderrInOrder: every stage's stderr reaches the terminal,
// in the order written, interleaved with stdout.
func TestPipelineStderrInOrder(t *testing.T) {
	_, p := loginSetup(t)
	out := run(t, p, "cat nope | cat notes.txt")
	if len(out) != 2 || out[0].Stream != process.Stderr || out[1].Stream != process.Stdout {
		t.Fatalf("want the error, then the file: %v", out)
	}
	if !strings.Contains(out[0].Data.String(), "cat: nope") {
		t.Fatalf("error %q", out[0].Data.String())
	}
}

func TestExitStatus(t *testing.T) {
	_, p := loginSetup(t)
	for _, tc := range []struct {
		line   string
		status int
	}{
		{"ls", 0},
		{"cat nope", 1},
		{"nosuch", 127},
		{"ls /root", 1},
		{"grep note notes.txt", 0},
		{"grep zzz notes.txt", 1},
		{"cat nope | cat", 0}, // a pipeline's status is its last stage's
		{"cat notes.txt | grep zzz", 1},
		{"cd /root", 1},
		{"cd docs", 0},
	} {
		run(t, p, tc.line)
		if got := lastStatus(t, p); got != tc.status {
			t.Errorf("%q: status %d, want %d", tc.line, got, tc.status)
		}
	}
}

// TestExitStatusWhileRunning: an ssh session has no status until it
// ends; then it is the remote shell's.
func TestExitStatusWhileRunning(t *testing.T) {
	_, p := loginSetup(t)
	run(t, p, "ssh test")
	inner := p.(Info).Active()
	if inner == p.(Info) {
		t.Fatal("ssh should run a remote shell")
	}
	if _, ok := p.(Info).LastStatus(); ok {
		t.Fatal("ssh has a status while still connected")
	}
	run(t, p, "cat nope")
	if status, ok := inner.LastStatus(); !ok || status != 1 {
		t.Fatalf("remote shell's $? = %v %v, want 1", status, ok)
	}
	run(t, p, "exit")
	if got := lastStatus(t, p); got != 1 {
		t.Fatalf("ssh exited %d, want the remote shell's 1", got)
	}
}

// TestPipelineNeverEndingSourceDoesNotHang: a source that never sends
// EOF makes each read return, a step at a time.
func TestPipelineNeverEndingSourceDoesNotHang(t *testing.T) {
	_, p := loginSetup(t)
	writeString(t, p, "yes | cat")
	writeEnter(t, p)
	for range 100 {
		p.Read()
	}
	if _, ok := p.(Info).LastStatus(); ok {
		t.Fatal("yes | cat finished")
	}
	p.Kill()
	if status, exited := p.Exit(); !exited || status != process.KilledStatus {
		t.Fatalf("killed shell: %v %v", status, exited)
	}
}
