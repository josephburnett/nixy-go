package computer

import (
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
