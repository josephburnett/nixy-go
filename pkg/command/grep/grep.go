package grep

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("grep", &simulation.Binary{
		Launch: launch,
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
		return command.NewErrorProcess(owner, "grep: missing pattern\n"), nil
	}

	pattern := args[0]

	if len(args) >= 2 {
		// grep pattern file
		c, err := sim.GetComputer(hostname)
		if err != nil {
			return nil, err
		}
		path := resolvePath(cwd, args[1])
		f, err := c.Filesystem.Navigate(path)
		if err != nil {
			return command.NewErrorProcess(owner, fmt.Sprintf("grep: %v\n", err)), nil
		}
		if !f.CanRead(owner) {
			return command.NewErrorProcess(owner, "grep: permission denied\n"), nil
		}
		if f.Type == file.Folder {
			return command.NewErrorProcess(owner, fmt.Sprintf("grep: %v: Is a directory\n", args[1])), nil
		}
		output := filterLines(f.Data, pattern)
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

func resolvePath(cwd []string, path string) []string {
	if strings.HasPrefix(path, "/") {
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		var out []string
		for _, p := range parts {
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	result := append([]string{}, cwd...)
	for _, p := range strings.Split(path, "/") {
		if p == ".." {
			if len(result) > 0 {
				result = result[:len(result)-1]
			}
		} else if p != "" && p != "." {
			result = append(result, p)
		}
	}
	return result
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
