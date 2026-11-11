package process

import (
	"errors"
	"strings"
)

// Stream is one of a process's two output streams.
type Stream int

const (
	Stdout Stream = 1
	Stderr Stream = 2
)

// Output is one write by a process to stdout or stderr.
type Output struct {
	Stream Stream
	Data   Data
}

// P is a running process.
//
// Processes act only when called: a command does its work at launch or
// when it gets input, and what it writes waits in order until Read. A
// process ends by exiting with a status (0 is success), as in Unix.
type P interface {
	// Read returns what the process wrote since the last Read, both
	// streams, in the order it wrote them. Reading is also what moves a
	// pipeline's data along, so a reader keeps calling Read until the
	// process exits or has nothing to say.
	Read() []Output
	// Stdin sends input to the process. It fails when the process takes
	// no input.
	Stdin(in Data) error
	// CloseStdin is end of input: whoever was writing to the process's
	// stdin is done, like the upstream end of a pipe closing. A process
	// that doesn't read stdin ignores it.
	CloseStdin()
	// Next returns the set of valid next inputs for this process.
	Next() []Datum
	// Owner returns the user who owns this process.
	Owner() string
	// Kill terminates the process: it exits with KilledStatus, and
	// output it hasn't been read is lost. It is SIGKILL, which no
	// process can catch.
	Kill()
	// Signal delivers sig. Unless the process handles it, it dies of
	// it (exit status sig.Status()); SigKill is Kill. Interactive
	// shells ignore SIGINT and SIGTERM, as bash does.
	Signal(sig Signal)
	// Exit reports whether the process has exited, and its exit status.
	// Read once more after exiting to collect what it wrote last.
	Exit() (status int, exited bool)
}

// KilledStatus is the exit status of a killed process: 128 + SIGKILL,
// as a shell reports it.
const KilledStatus = 128 + 9 // SigKill.Status()

// Text returns the characters in outs written to stream.
func Text(outs []Output, stream Stream) string {
	var sb strings.Builder
	for _, o := range outs {
		if o.Stream == stream {
			sb.WriteString(o.Data.String())
		}
	}
	return sb.String()
}

// Written is the part of a process that keeps its output and exit
// status. Embed it and call Write and Finish; it provides Read, Exit
// and an ignoring CloseStdin.
type Written struct {
	out    []Output
	status int
	exited bool
}

// Write queues data on stream.
func (w *Written) Write(stream Stream, data Data) {
	if len(data) > 0 && !w.exited {
		w.out = append(w.out, Output{Stream: stream, Data: data})
	}
}

// Finish exits with status. Later writes are dropped.
func (w *Written) Finish(status int) {
	if !w.exited {
		w.status, w.exited = status, true
	}
}

// Read returns and clears the queued output.
func (w *Written) Read() []Output {
	out := w.out
	w.out = nil
	return out
}

// Exit reports the exit status once finished.
func (w *Written) Exit() (int, bool) { return w.status, w.exited }

// CloseStdin is ignored: override it in a process that reads stdin.
func (w *Written) CloseStdin() {}

// Kill exits with KilledStatus and drops unread output.
func (w *Written) Kill() {
	w.out = nil
	w.Finish(KilledStatus)
}

// Signal is the default action for sig: die of it, keeping what was
// written before. SigKill is Kill.
func (w *Written) Signal(sig Signal) {
	if sig == SigKill {
		w.Kill()
		return
	}
	w.Finish(sig.Status())
}

// ErrNoInput is what Stdin returns for a process that takes no input.
var ErrNoInput = errors.New("process takes no input")
