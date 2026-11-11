package shell

import (
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// runUntilPrompt reads p until its job is reaped, at most n times.
func runUntilPrompt(p process.P, n int) (stdout, stderr string, ok bool) {
	var outs []process.Output
	for range n {
		outs = append(outs, p.Read()...)
		if s := p.(*shell); s.job == nil {
			return process.Text(outs, process.Stdout), process.Text(outs, process.Stderr), true
		}
	}
	return process.Text(outs, process.Stdout), process.Text(outs, process.Stderr), false
}

// Ctrl+C is offered only while a command runs, and interrupts it: the
// terminal shows ^C and $? is 130, as in bash.
func TestShellCtrlCInterruptsJob(t *testing.T) {
	for _, line := range []string{"yes", "yes | cat", "yes | grep y", "yes | grep x"} {
		_, p := testSetup(t)
		if containsDatum(p.Next(), process.SigInt) {
			t.Fatal("Ctrl+C offered at an idle prompt")
		}
		writeString(t, p, line)
		writeEnter(t, p)
		if _, _, done := runUntilPrompt(p, 20); done {
			t.Fatalf("%q ended on its own", line)
		}
		if !containsDatum(p.Next(), process.SigInt) {
			t.Fatalf("%q: Ctrl+C not offered while it runs", line)
		}
		if err := p.Stdin(process.Data{process.SigInt}); err != nil {
			t.Fatal(err)
		}
		out, errOut, done := runUntilPrompt(p, 5)
		if !done || !strings.Contains(out, "^C\n") || errOut != "" {
			t.Fatalf("%q after Ctrl+C: done=%v stdout %q stderr %q", line, done, out, errOut)
		}
		if status, ok := p.(*shell).LastStatus(); !ok || status != 130 {
			t.Fatalf("%q: $? = %d, want 130", line, status)
		}
		if containsDatum(p.Next(), process.SigInt) {
			t.Fatalf("%q: Ctrl+C still offered at the prompt", line)
		}
	}
}

// Through ssh, Ctrl+C belongs to the remote shell: offered only when
// its job runs.
func TestShellCtrlCThroughSSH(t *testing.T) {
	_, p := testSetup(t)
	writeString(t, p, "ssh test")
	writeEnter(t, p)
	_, _, _ = runUntilPrompt(p, 2)
	if containsDatum(p.Next(), process.SigInt) {
		t.Fatal("Ctrl+C offered at the remote prompt")
	}
	writeString(t, p, "yes")
	writeEnter(t, p)
	_ = p.Read()
	if !containsDatum(p.Next(), process.SigInt) {
		t.Fatal("Ctrl+C not offered for the remote job")
	}
	if err := p.Stdin(process.Data{process.SigInt}); err != nil {
		t.Fatal(err)
	}
	for range 5 {
		_ = p.Read()
	}
	if p.(*shell).Active() == Info(p.(*shell)) {
		t.Fatal("Ctrl+C ended the ssh session, not the remote job")
	}
	if containsDatum(p.Next(), process.SigInt) {
		t.Fatal("remote job still running after Ctrl+C")
	}
}

// A writer whose reader has gone gets SIGPIPE, so the pipeline ends.
func TestPipelineSigPipe(t *testing.T) {
	sim, p := testSetup(t)
	writeString(t, p, "yes | cat")
	writeEnter(t, p)
	_ = p.Read()
	pl := p.(*shell).job.(*pipeline)
	pl.stages[1].Kill() // the reader goes away
	if _, _, done := runUntilPrompt(p, 5); !done {
		t.Fatal("yes kept running with no reader")
	}
	if status, _ := pl.stages[0].Exit(); status != process.SigPipe.Status() {
		t.Fatalf("yes exited %d, want SIGPIPE's %d", status, process.SigPipe.Status())
	}
	c, _ := sim.GetComputer("test")
	if n := len(c.Processes()); n != 2 {
		t.Fatalf("%d processes left, want just the shells", n)
	}
}

// An interactive shell ignores SIGTERM and SIGINT; a job killed by a
// signal is reported the way bash does.
func TestShellSignals(t *testing.T) {
	sim, p := testSetup(t)
	p.Signal(process.SigTerm)
	p.Signal(process.SigInt)
	if _, exited := p.Exit(); exited {
		t.Fatal("shell died of SIGTERM or SIGINT")
	}
	writeString(t, p, "yes")
	writeEnter(t, p)
	_ = p.Read()
	c, _ := sim.GetComputer("test")
	procs := c.Processes()
	job := procs[len(procs)-1]
	if job.Command != "yes" {
		t.Fatalf("last process is %q, want yes", job.Command)
	}
	if err := c.Signal(job.PID, "user", process.SigTerm); err != nil {
		t.Fatal(err)
	}
	if _, errOut, done := runUntilPrompt(p, 5); !done || errOut != "Terminated\n" {
		t.Fatalf("after SIGTERM: done=%v stderr %q", done, errOut)
	}
	if status, _ := p.(*shell).LastStatus(); status != 143 {
		t.Fatalf("$? = %d, want 143", status)
	}
	p.Signal(process.SigKill)
	if status, exited := p.Exit(); !exited || status != process.KilledStatus {
		t.Fatalf("SIGKILL: exited=%v status %d", exited, status)
	}
}
