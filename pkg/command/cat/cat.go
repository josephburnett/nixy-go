package cat

import (
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

	data, err := c.Filesystem.ReadFile(file.Resolve(cwd, args[0]), owner)
	if err != nil {
		return command.Fail(owner, "cat", args[0], err), nil
	}
	if len(data) > 0 && !strings.HasSuffix(data, "\n") {
		data += "\n"
	}
	return command.NewSingleValueProcess(owner, data), nil
}

// stdinCat copies stdin to stdout until its input closes.
type stdinCat struct {
	process.Written
	owner string
}

func (s *stdinCat) Stdin(in process.Data) error {
	s.Write(process.Stdout, in)
	return nil
}

func (s *stdinCat) CloseStdin()           { s.Finish(0) }
func (s *stdinCat) Next() []process.Datum { return nil }
func (s *stdinCat) Owner() string         { return s.owner }
