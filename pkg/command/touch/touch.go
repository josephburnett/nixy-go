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
		if err := touch(c.Filesystem, file.Resolve(cwd, arg), owner); err != nil {
			return command.Fail(owner, "touch", fmt.Sprintf("cannot touch '%v'", arg), err), nil
		}
	}

	return command.NewSingleValueProcess(owner, ""), nil
}

// touch creates an empty file at path owned by owner, or leaves an
// existing one alone.
func touch(root *file.F, path []string, owner string) error {
	if len(path) == 0 {
		return nil // `/` exists
	}
	dir, name := path[:len(path)-1], path[len(path)-1]
	parent, err := root.LookupDir(dir, owner)
	if err != nil {
		return err
	}
	if _, exists := parent.Files[name]; exists {
		return nil
	}
	return root.CreateFile(dir, name, &file.F{
		Type:             file.Text,
		Owner:            owner,
		OwnerPermission:  file.Write,
		CommonPermission: file.Read,
	}, owner)
}
