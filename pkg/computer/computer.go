package computer

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
)

type C struct {
	filesystem *file.F
	processes  *process.Space
}

func New(filesystem *file.F) *C {
	return &C{
		filesystem: filesystem,
		processes:  process.NewSpace(),
	}
}

func (c *C) Boot() error {
	return nil
}

func (c *C) Add(p process.P) {
	c.processes.Add(p)
}

func (c *C) GetFile(path []string) (*file.F, error) {
	currentFile := c.filesystem
	for _, p := range path {
		if currentFile.Type != file.Folder {
			return nil, fmt.Errorf("%v is not a folder", p)
		}
		f, ok := currentFile.Files[p]
		if !ok {
			return nil, fmt.Errorf("file %v not found", p)
		}
		currentFile = f
	}
	return currentFile, nil
}
