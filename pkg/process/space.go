package process

import "sort"

// Info describes an entry in a process table, as ps shows it.
type Info struct {
	PID     int
	Owner   string
	Command string // the command line that started it
}

// Errno is an error from the process table, with the C library's text
// for it, as kill(1) prints it.
type Errno string

func (e Errno) Error() string { return string(e) }

const (
	ErrNoProcess    Errno = "No such process"
	ErrNotPermitted Errno = "Operation not permitted"
)

// Space is a computer's process table. A process is in it from when it
// starts until it exits.
type Space struct {
	next      int
	processes map[int]entry
}

type entry struct {
	p       P
	command string
}

func NewSpace() *Space {
	return &Space{next: 1, processes: map[int]entry{}}
}

// Add registers p, started by command, and returns its PID.
func (ps *Space) Add(p P, command string) int {
	ps.prune()
	pid := ps.next
	ps.next++
	ps.processes[pid] = entry{p: p, command: command}
	return pid
}

// List returns the running processes, by PID.
func (ps *Space) List() []Info {
	ps.prune()
	out := make([]Info, 0, len(ps.processes))
	for pid, e := range ps.processes {
		out = append(out, Info{PID: pid, Owner: e.p.Owner(), Command: e.command})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PID < out[j].PID })
	return out
}

// Get returns the running process with pid.
func (ps *Space) Get(pid int) (P, bool) {
	ps.prune()
	e, ok := ps.processes[pid]
	return e.p, ok
}

// Signal sends sig to the process with pid. Who may do so is the
// caller's check (see computer.C.Signal). A process that dies of it
// leaves the table.
func (ps *Space) Signal(pid int, sig Signal) error {
	p, ok := ps.Get(pid)
	if !ok {
		return ErrNoProcess
	}
	p.Signal(sig)
	ps.prune()
	return nil
}

// prune removes processes that have exited.
func (ps *Space) prune() {
	for pid, e := range ps.processes {
		if _, exited := e.p.Exit(); exited {
			delete(ps.processes, pid)
		}
	}
}
