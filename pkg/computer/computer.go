package computer

import (
	"github.com/josephburnett/nixy-go/pkg/model"
	"github.com/josephburnett/nixy-go/pkg/process"
)

type Computer struct {
	Filesystem *model.File
	Processes  *process.ProcessSpace
}

func NewComputer(filesystem *mode.File) *Computer {
	return &Computer{
		FileSystem: f,
		Processes:  process.NewProcessSpace(),
	}, 
}

func (c *Computer) Boot() error {
	return nil
}
