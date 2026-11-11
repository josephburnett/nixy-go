package command

import "github.com/josephburnett/nixy-go/pkg/process"

// Proc is the part every command's process shares: its output and exit
// status (process.Written), who it runs as, and no keys of its own. It
// takes no input; a command that reads a pipe defines Stdin and
// CloseStdin.
type Proc struct {
	process.Written
	owner string
}

var _ process.P = &Proc{}

// NewProc is a Proc running as owner.
func NewProc(owner string) Proc { return Proc{owner: owner} }

func (p *Proc) Owner() string            { return p.owner }
func (p *Proc) Stdin(process.Data) error { return process.ErrNoInput }
func (p *Proc) Next() []process.Datum    { return nil }

// Exited returns a process that wrote stdout and stderr (either may be
// empty) and exited with status.
func Exited(owner, stdout, stderr string, status int) process.P {
	p := &Proc{owner: owner}
	if stderr != "" {
		p.Write(process.Stderr, process.CharsData(stderr))
	}
	if stdout != "" {
		p.Write(process.Stdout, process.CharsData(stdout))
	}
	p.Finish(status)
	return p
}

// NewSingleValueProcess is a command that succeeded, printing value.
func NewSingleValueProcess(owner, value string) process.P {
	return Exited(owner, value, "", 0)
}

// NewErrorProcess is a command that failed, printing msg on stderr.
func NewErrorProcess(owner, msg string) process.P {
	return Exited(owner, "", msg, 1)
}
