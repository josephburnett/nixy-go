package computer

import (
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
)

type C struct {
	Filesystem *file.F
	processes  *process.Space
}

func New(filesystem *file.F) *C {
	return &C{
		Filesystem: filesystem,
		processes:  process.NewSpace(),
	}
}

// Add puts a process started by command in the process table and
// returns its PID. It leaves the table when it exits.
func (c *C) Add(p process.P, command string) int {
	return c.processes.Add(p, command)
}

// Processes lists the running processes, by PID.
func (c *C) Processes() []process.Info {
	return c.processes.List()
}

// Signal sends sig to process pid on behalf of user, which only its
// owner and root may do, as with kill(2).
func (c *C) Signal(pid int, user string, sig process.Signal) error {
	p, ok := c.processes.Get(pid)
	if !ok {
		return process.ErrNoProcess
	}
	if user != file.OwnerRoot && user != p.Owner() {
		return process.ErrNotPermitted
	}
	return c.processes.Signal(pid, sig)
}

// Kill is Signal with SIGKILL.
func (c *C) Kill(pid int, user string) error {
	return c.Signal(pid, user, process.SigKill)
}
