package mv

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("mv", &simulation.Binary{
		Launch: launch,
		// `mv FILE DEST`. Files only, like rm: moving folders could
		// leave the cwd behind.
		ValidArgs: command.Positional{
			Args:     []command.Arg{command.FilePath, command.MoveDest},
			Required: 2,
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
	if len(args) < 2 {
		return command.NewErrorProcess(owner, "mv: missing operand\n"), nil
	}

	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}

	srcPath := file.Resolve(cwd, args[0])
	dstPath := file.Resolve(cwd, args[1])
	if len(srcPath) == 0 {
		return command.NewErrorProcess(owner, "mv: cannot move '/'\n"), nil
	}
	srcName := srcPath[len(srcPath)-1]
	srcParent, err := c.Filesystem.Navigate(srcPath[:len(srcPath)-1])
	if err != nil {
		return command.NewErrorProcess(owner, fmt.Sprintf("mv: %v\n", err)), nil
	}
	srcFile, ok := srcParent.Files[srcName]
	if !ok {
		return command.NewErrorProcess(owner, fmt.Sprintf("mv: %v: not found\n", args[0])), nil
	}

	// Moving onto a folder moves into it, keeping the name.
	if f, err := c.Filesystem.Navigate(dstPath); err == nil && f.Type == file.Folder {
		dstPath = append(dstPath, srcName)
	} else if strings.HasSuffix(args[1], "/") {
		return command.NewErrorProcess(owner, fmt.Sprintf("mv: %v: not a directory\n", args[1])), nil
	}
	if len(dstPath) == 0 {
		return command.NewErrorProcess(owner, "mv: cannot move to '/'\n"), nil
	}
	if strings.Join(dstPath, "/") == strings.Join(srcPath, "/") {
		return command.NewErrorProcess(owner, fmt.Sprintf("mv: '%v' and '%v' are the same file\n", args[0], args[1])), nil
	}
	dstName := dstPath[len(dstPath)-1]
	dstParent, err := c.Filesystem.Navigate(dstPath[:len(dstPath)-1])
	if err != nil {
		return command.NewErrorProcess(owner, fmt.Sprintf("mv: %v\n", err)), nil
	}
	if dstParent.Type != file.Folder {
		return command.NewErrorProcess(owner, fmt.Sprintf("mv: %v: not a directory\n", args[1])), nil
	}
	if !srcParent.CanWrite(owner) || !dstParent.CanWrite(owner) {
		return command.NewErrorProcess(owner, "mv: permission denied\n"), nil
	}
	if existing, ok := dstParent.Files[dstName]; ok {
		if existing.Type == file.Folder {
			return command.NewErrorProcess(owner, fmt.Sprintf("mv: cannot overwrite directory '%v'\n", dstName)), nil
		}
		// Like real mv, an existing file is replaced.
		delete(dstParent.Files, dstName)
	}
	if err := c.Filesystem.CreateFile(dstPath[:len(dstPath)-1], dstName, srcFile, owner); err != nil {
		return command.NewErrorProcess(owner, fmt.Sprintf("mv: %v\n", err)), nil
	}
	delete(srcParent.Files, srcName)

	return command.NewSingleValueProcess(owner, ""), nil
}
