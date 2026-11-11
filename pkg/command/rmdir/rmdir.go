package rmdir

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("rmdir", &simulation.Binary{
		Launch: launch,
		// `rmdir DIR`: only an empty folder, as real rmdir.
		ValidArgs: command.Positional{Args: []command.Arg{command.EmptyDir}, Required: 1}.Validate,
	})
}

func launch(sim *simulation.S, owner, hostname string, cwd, args []string) (process.P, error) {
	if len(args) == 0 {
		return command.NewErrorProcess(owner, "rmdir: missing operand\n"), nil
	}
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}
	for _, arg := range args {
		if err := rmdir(c.Filesystem, file.Resolve(cwd, arg), owner); err != nil {
			return command.Fail(owner, "rmdir", fmt.Sprintf("failed to remove '%v'", arg), err), nil
		}
	}
	return command.NewSingleValueProcess(owner, ""), nil
}

// rmdir deletes the empty folder at path. Like any delete, it takes
// write permission on the parent folder.
func rmdir(root *file.F, path []string, owner string) error {
	if len(path) == 0 {
		return file.ErrNotEmpty // `/`
	}
	f, err := root.Lookup(path, owner)
	if err != nil {
		return err
	}
	if f.Type != file.Folder {
		return file.ErrNotDir
	}
	if len(f.Files) > 0 {
		return file.ErrNotEmpty
	}
	return root.DeleteFile(path[:len(path)-1], path[len(path)-1], owner)
}
