package environment

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/computer"
	"github.com/josephburnett/nixy-go/pkg/file"
)

type Environment struct {
	computers map[string]*computer.Computer
}

func NewEnvironment(filesystems map[string]*file.File) (*Environment, error) {
	e := &Environment{
		computers: map[string]*computer.Computer{},
	}
	for hostname, f := range filesystems {
		if err := e.boot(f, hostname); err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e *Environment) Boot(filesystem *file.File, hostname string) error {
	if _, exists := e.computers[hostname]; exists {
		return fmt.Errorf("conflicting hostname: %v", hostname)
	}
	return e.boot(filesystem, hostname)
}

func (e *Environment) boot(filesystem *file.File, hostname string) error {
	c := computer.NewComputer(filesystem)
	if err := c.Boot(); err != nil {
		return fmt.Errorf("error booting %v: %v", hostname, err)
	}
	e.computers[hostname] = c
	return nil
}
