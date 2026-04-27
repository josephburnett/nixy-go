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
		target, err = resolvePath(cwd, args[0])
		if err != nil {
			return nil, err
		}
	}

	f, err := c.Filesystem.Navigate(target)
	if err != nil {
		return command.NewErrorProcess(owner, fmt.Sprintf("ls: %v\n", err)), nil
	}
	if !f.CanRead(owner) {
		return command.NewErrorProcess(owner, "ls: permission denied\n"), nil
	}
	if f.Type != file.Folder {
		// ls on a file just shows the file name
		parts := strings.Split(args[0], "/")
		return command.NewSingleValueProcess(owner, parts[len(parts)-1]+"\n"), nil
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

func resolvePath(cwd []string, path string) ([]string, error) {
	if strings.HasPrefix(path, "/") {
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		return filterEmpty(parts), nil
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
	return result, nil
}

func filterEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
