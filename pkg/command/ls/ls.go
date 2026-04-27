package ls

import (
	"fmt"
	"sort"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("ls", &simulation.Binary{
		Launch:       launch,
		ValidArgs:    command.ValidArgsFolder,
		OptionalArgs: true, // ls with no args lists cwd
	})
}

func launch(
	sim *simulation.S,
	owner string,
	hostname string,
	cwd []string,
	args []string,
) (process.P, error) {
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}

	target := cwd
	if len(args) > 0 {
		target = file.Resolve(cwd, args[0])
	}

	f, err := c.Filesystem.Navigate(target)
	if err != nil {
		return command.NewErrorProcess(owner, fmt.Sprintf("ls: %v\n", err)), nil
	}
	if !f.CanRead(owner) {
		return command.NewErrorProcess(owner, "ls: permission denied\n"), nil
	}
	if f.Type != file.Folder {
		// ls on a file shows the file name (basename of the resolved path).
		name := ""
		if len(target) > 0 {
			name = target[len(target)-1]
		}
		return command.NewSingleValueProcess(owner, name+"\n"), nil
	}

	var names []string
	for name := range f.Files {
		names = append(names, name)
	}
	sort.Strings(names)

	output := strings.Join(names, "\n")
	if len(names) > 0 {
		output += "\n"
	}
	return command.NewSingleValueProcess(owner, output), nil
}
