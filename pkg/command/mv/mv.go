package mv

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("mv", &simulation.Binary{
		Launch: launch,
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

	srcPath := resolvePath(cwd, args[0])
	dstPath := resolvePath(cwd, args[1])

	if len(srcPath) == 0 {
		return command.NewErrorProcess(owner, "mv: cannot move '/'\n"), nil
	}

	// Get source file
	srcDir := srcPath[:len(srcPath)-1]
	srcName := srcPath[len(srcPath)-1]
	srcParent, err := c.Filesystem.Navigate(srcDir)
	if err != nil {
		return command.NewErrorProcess(owner, fmt.Sprintf("mv: %v\n", err)), nil
	}
	srcFile, ok := srcParent.Files[srcName]
	if !ok {
		return command.NewErrorProcess(owner, fmt.Sprintf("mv: %v: not found\n", args[0])), nil
	}
	if !srcParent.CanWrite(owner) {
		return command.NewErrorProcess(owner, "mv: permission denied\n"), nil
	}

	// Determine destination
	if len(dstPath) == 0 {
		return command.NewErrorProcess(owner, "mv: cannot move to '/'\n"), nil
	}
	dstDir := dstPath[:len(dstPath)-1]
	dstName := dstPath[len(dstPath)-1]

	// Create at destination
	err = c.Filesystem.CreateFile(dstDir, dstName, srcFile, owner)
	if err != nil {
		return command.NewErrorProcess(owner, fmt.Sprintf("mv: %v\n", err)), nil
	}

	// Remove from source
	delete(srcParent.Files, srcName)

	return command.NewSingleValueProcess(owner, ""), nil
}

func resolvePath(cwd []string, path string) []string {
	if strings.HasPrefix(path, "/") {
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		var out []string
		for _, p := range parts {
			if p != "" {
				out = append(out, p)
			}
		}
		return out
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
	return result
}
