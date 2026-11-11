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
		Launch: launch,
		// Files only: real rm refuses folders without -r.
		ValidArgs: command.Positional{Args: []command.Arg{command.FilePath}, Required: 1}.Validate,
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
		if err := remove(c.Filesystem, file.Resolve(cwd, arg), owner); err != nil {
			return command.Fail(owner, "rm", fmt.Sprintf("cannot remove '%v'", arg), err), nil
		}
	}

	return command.NewSingleValueProcess(owner, ""), nil
}

// remove deletes the file at path. Folders need -r, which rm doesn't
// have.
func remove(root *file.F, path []string, owner string) error {
	if len(path) == 0 {
		return file.ErrIsDir
	}
	f, err := root.Lookup(path, owner)
	if err != nil {
		return err
	}
	if f.Type == file.Folder {
		return file.ErrIsDir
	}
	return root.DeleteFile(path[:len(path)-1], path[len(path)-1], owner)
}
