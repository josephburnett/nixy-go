package mkdir

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("mkdir", &simulation.Binary{
		Launch: launch,
		// `mkdir DIR`: a new name in a folder that exists.
		ValidArgs: command.Positional{Args: []command.Arg{command.NewName}, Required: 1}.Validate,
	})
}

func launch(sim *simulation.S, owner, hostname string, cwd, args []string) (process.P, error) {
	if len(args) == 0 {
		return command.NewErrorProcess(owner, "mkdir: missing operand\n"), nil
	}
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}
	for _, arg := range args {
		if err := mkdir(c.Filesystem, file.Resolve(cwd, arg), owner); err != nil {
			return command.Fail(owner, "mkdir", fmt.Sprintf("cannot create directory '%v'", arg), err), nil
		}
	}
	return command.NewSingleValueProcess(owner, ""), nil
}

// mkdir makes an empty folder at path, owned by owner, which anyone may
// list (like the usual umask's 755).
func mkdir(root *file.F, path []string, owner string) error {
	if len(path) == 0 {
		return file.ErrExist // `/`
	}
	return root.CreateFile(path[:len(path)-1], path[len(path)-1], &file.F{
		Type:             file.Folder,
		Owner:            owner,
		OwnerPermission:  file.Write,
		CommonPermission: file.Read,
		Files:            map[string]*file.F{},
	}, owner)
}
