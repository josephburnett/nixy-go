package rm

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("rm", &simulation.Binary{
		Launch:    launch,
		ValidArgs: command.ValidArgsFile,
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
		path := file.Resolve(cwd, arg)
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
