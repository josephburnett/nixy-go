package cat

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("cat", &simulation.Binary{
		Launch: launch,
		// `cat FILE`, or bare `cat` reading an upstream pipe. Standalone
		// bare cat would hang: the game has no terminal stdin.
		ValidArgs: command.Positional{
			Args:       []command.Arg{command.FilePath},
			Required:   1,
			ReadsStdin: true,
		}.Validate,
	})
}

func launch(
	sim *simulation.S,
	owner string,
	hostname string,
	cwd []string,
	args []string,
) (process.P, error) {
	if len(args) == 0 {
		// No args: read from stdin (for pipes). Return a passthrough process.
		return &stdinCat{owner: owner}, nil
	}

	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}

	f, err := c.Filesystem.Navigate(file.Resolve(cwd, args[0]))
	if err != nil {
		return command.NewErrorProcess(owner, fmt.Sprintf("cat: %v\n", err)), nil
	}
	if f.Type == file.Folder {
		return command.NewErrorProcess(owner, fmt.Sprintf("cat: %v: Is a directory\n", args[0])), nil
	}
	if !f.CanRead(owner) {
		return command.NewErrorProcess(owner, "cat: permission denied\n"), nil
	}
	data := f.Data
	if len(data) > 0 && !strings.HasSuffix(data, "\n") {
		data += "\n"
	}
	return command.NewSingleValueProcess(owner, data), nil
}

// stdinCat passes stdin through to stdout (for use in pipes).
type stdinCat struct {
	owner string
	buf   process.Data
	eof   bool
}

func (s *stdinCat) Stdout() (process.Data, bool, error) {
	d := s.buf
	s.buf = nil
	return d, s.eof, nil
}

func (s *stdinCat) Stderr() (process.Data, bool, error) { return nil, true, nil }

func (s *stdinCat) Stdin(in process.Data) (bool, error) {
	s.buf = append(s.buf, in...)
	return false, nil
}

func (s *stdinCat) Next() []process.Datum { return nil }
func (s *stdinCat) Owner() string         { return s.owner }
func (s *stdinCat) Kill() error           { s.eof = true; return nil }
