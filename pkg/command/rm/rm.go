package rm

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("rm", &simulation.Binary{
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
		return command.NewErrorProcess(owner, "rm: missing operand\n"), nil
	}

	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}

	for _, arg := range args {
		path := resolvePath(cwd, arg)
		if len(path) == 0 {
			return command.NewErrorProcess(owner, "rm: cannot remove '/'\n"), nil
		}
		dir := path[:len(path)-1]
		name := path[len(path)-1]
		err := c.Filesystem.DeleteFile(dir, name, owner)
		if err != nil {
			return command.NewErrorProcess(owner, fmt.Sprintf("rm: %v\n", err)), nil
		}
	}

	return command.NewSingleValueProcess(owner, ""), nil
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
