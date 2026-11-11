package grep

import (
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("grep", &simulation.Binary{
		Launch: launch,
		// `grep PATTERN FILE`, or `grep PATTERN` reading an upstream
		// pipe. Standalone `grep PATTERN` would hang: no terminal stdin.
		ValidArgs: command.Positional{
			Args:         []command.Arg{command.Pattern, command.FilePath},
			Required:     2,
			ReadsStdin:   true,
			PipeRequired: 1,
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
		return command.NewErrorProcess(owner, "Usage: grep PATTERN [FILE]\n"), nil
	}

	pattern := args[0]

	if len(args) >= 2 {
		// grep pattern file
		c, err := sim.GetComputer(hostname)
		if err != nil {
			return nil, err
		}
		data, err := c.Filesystem.ReadFile(file.Resolve(cwd, args[1]), owner)
		if err != nil {
			return command.Fail(owner, "grep", args[1], err), nil
		}
		return result(owner, filterLines(data, pattern)), nil
	}

	// grep pattern (reads from stdin)
	return &stdinGrep{owner: owner, pattern: pattern}, nil
}

func filterLines(data, pattern string) string {
	var matches []string
	for _, line := range strings.Split(data, "\n") {
		if strings.Contains(line, pattern) {
			matches = append(matches, line)
		}
	}
	if len(matches) == 0 {
		return ""
	}
	return strings.Join(matches, "\n") + "\n"
}

// result is grep's output and exit status: 1 when nothing matched.
func result(owner, matches string) process.P {
	status := 0
	if matches == "" {
		status = 1
	}
	return command.Exited(owner, matches, "", status)
}

// stdinGrep filters stdin, matching once its input closes.
type stdinGrep struct {
	process.Written
	owner   string
	pattern string
	buf     strings.Builder
}

func (g *stdinGrep) Stdin(in process.Data) error {
	g.buf.WriteString(in.String())
	return nil
}

func (g *stdinGrep) CloseStdin() {
	matches := filterLines(g.buf.String(), g.pattern)
	if matches != "" {
		g.Write(process.Stdout, process.CharsData(matches))
		g.Finish(0)
	}
	g.Finish(1)
}

func (g *stdinGrep) Next() []process.Datum { return nil }
func (g *stdinGrep) Owner() string         { return g.owner }
