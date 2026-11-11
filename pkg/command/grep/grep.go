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
		output := filterLines(data, pattern)
		return command.NewSingleValueProcess(owner, output), nil
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

type stdinGrep struct {
	owner   string
	pattern string
	buf     string
	done    bool
}

func (g *stdinGrep) Stdout() (process.Data, bool, error) {
	if !g.done {
		return nil, false, nil
	}
	output := filterLines(g.buf, g.pattern)
	g.buf = ""
	return process.CharsData(output), true, nil
}

func (g *stdinGrep) Stderr() (process.Data, bool, error) { return nil, true, nil }

func (g *stdinGrep) Stdin(in process.Data) (bool, error) {
	for _, d := range in {
		if c, ok := d.(process.Chars); ok {
			g.buf += string(c)
		}
	}
	return false, nil
}

func (g *stdinGrep) Next() []process.Datum { return nil }
func (g *stdinGrep) Owner() string         { return g.owner }

func (g *stdinGrep) Kill() error {
	g.done = true
	return nil
}
