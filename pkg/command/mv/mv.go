package mv

import (
	"fmt"
	"slices"
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

	if err := move(c.Filesystem, cwd, args[0], args[1], owner); err != nil {
		return command.NewErrorProcess(owner, "mv: "+err.Error()+"\n"), nil
	}
	return command.NewSingleValueProcess(owner, ""), nil
}

// move moves the file src to dst (as typed) for owner. Errors read like
// GNU mv's, without the "mv: " prefix.
func move(root *file.F, cwd []string, src, dst, owner string) error {
	cannot := func(err error) error {
		return fmt.Errorf("cannot move '%v' to '%v': %w", src, dst, err)
	}
	srcPath := file.Resolve(cwd, src)
	dstPath := file.Resolve(cwd, dst)
	srcFile, err := root.Lookup(srcPath, owner)
	if err != nil {
		return fmt.Errorf("cannot stat '%v': %w", src, err)
	}
	if len(srcPath) == 0 || srcFile.Type == file.Folder {
		return cannot(file.ErrIsDir) // folders need more than this mv does
	}
	srcName := srcPath[len(srcPath)-1]

	// Moving onto a folder moves into it, keeping the name.
	if f, err := root.Lookup(dstPath, owner); err == nil && f.Type == file.Folder {
		dstPath = append(dstPath, srcName)
	} else if strings.HasSuffix(dst, "/") {
		return cannot(file.ErrNotDir)
	}
	if slices.Equal(dstPath, srcPath) {
		return fmt.Errorf("'%v' and '%v' are the same file", src, dst)
	}
	srcParent, err := root.LookupDir(srcPath[:len(srcPath)-1], owner)
	if err != nil {
		return cannot(err)
	}
	dstDir, dstName := dstPath[:len(dstPath)-1], dstPath[len(dstPath)-1]
	dstParent, err := root.LookupDir(dstDir, owner)
	if err != nil {
		return cannot(err)
	}
	if !srcParent.CanWrite(owner) || !dstParent.CanWrite(owner) {
		return cannot(file.ErrPermission)
	}
	if existing, ok := dstParent.Files[dstName]; ok {
		if existing.Type == file.Folder {
			return fmt.Errorf("cannot overwrite directory '%v' with non-directory", dst)
		}
		// Like real mv, an existing file is replaced.
		delete(dstParent.Files, dstName)
	}
	if err := root.CreateFile(dstDir, dstName, srcFile, owner); err != nil {
		return cannot(err)
	}
	delete(srcParent.Files, srcName)
	return nil
}
