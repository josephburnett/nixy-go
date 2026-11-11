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
		Launch: launch,
		// `ls` lists the cwd; `ls PATH` a folder, or names a file.
		ValidArgs: command.Positional{Args: []command.Arg{command.AnyPath}, Writes: true}.Validate,
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

	target, operand := cwd, "."
	if len(args) > 0 {
		target, operand = file.Resolve(cwd, args[0]), args[0]
	}

	f, err := c.Filesystem.Lookup(target, owner)
	if err != nil {
		return command.Fail(owner, "ls", fmt.Sprintf("cannot access '%v'", operand), err), nil
	}
	if f.Type != file.Folder {
		// ls on a file names it, as typed.
		return command.NewSingleValueProcess(owner, operand+"\n"), nil
	}
	if !f.CanRead(owner) {
		return command.Fail(owner, "ls", fmt.Sprintf("cannot open directory '%v'", operand), file.ErrPermission), nil
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
