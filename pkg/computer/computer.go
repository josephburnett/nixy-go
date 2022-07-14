package computer

import (
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
)

type Computer struct {
	filesystem *file.File
	processes  *process.ProcessSpace
}

func NewComputer(filesystem *file.File) *Computer {
	return &Computer{
		filesystem: filesystem,
		processes:  process.NewProcessSpace(),
	}
}

func (c *Computer) Boot() error {
	return nil
}
