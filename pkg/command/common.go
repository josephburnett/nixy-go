package command

import "github.com/josephburnett/nixy-go/pkg/process"

// finished is a command that did all its work at launch: its output is
// written and it has exited.
type finished struct {
	process.Written
	owner string
}

var _ process.P = &finished{}

// Exited returns a process that wrote stdout and stderr (either may be
// empty) and exited with status.
func Exited(owner, stdout, stderr string, status int) process.P {
	p := &finished{owner: owner}
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

func (f *finished) Owner() string            { return f.owner }
func (f *finished) Stdin(process.Data) error { return process.ErrNoInput }
func (f *finished) Next() []process.Datum    { return nil }
