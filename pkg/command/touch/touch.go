package touch

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("touch", &simulation.Binary{
		Launch:    launch,
		ValidArgs: command.Positional{Args: []command.Arg{command.NewPath}, Required: 1}.Validate,
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
		return command.NewErrorProcess(owner, "touch: missing operand\n"), nil
	}

	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}

	for _, arg := range args {
		path := file.Resolve(cwd, arg)
		if len(path) == 0 {
			return command.NewErrorProcess(owner, "touch: cannot touch '/'\n"), nil
		}
		dir := path[:len(path)-1]
		name := path[len(path)-1]

		// Check if file already exists - if so, no-op
		parent, err := c.Filesystem.Navigate(dir)
		if err != nil {
			return command.NewErrorProcess(owner, fmt.Sprintf("touch: %v\n", err)), nil
		}
		if _, exists := parent.Files[name]; exists {
			continue
		}

		newFile := &file.F{
			Type:             file.Text,
			Owner:            owner,
			OwnerPermission:  file.Write,
			CommonPermission: file.Read,
			Data:             "",
		}
		err = c.Filesystem.CreateFile(dir, name, newFile, owner)
		if err != nil {
			return command.NewErrorProcess(owner, fmt.Sprintf("touch: %v\n", err)), nil
		}
	}

	return command.NewSingleValueProcess(owner, ""), nil
}
