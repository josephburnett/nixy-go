package environment

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/computer"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
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

func (e *Environment) Launch(hostname, binaryName string, ctx Context, args string, input process.Process) (process.Process, error) {
	c, ok := e.computers[hostname]
	if !ok {
		return nil, fmt.Errorf("hostname %v not found", hostname)
	}
	b, ok := registry[binaryName]
	if !ok {
		return nil, fmt.Errorf("binary %v not found", binaryName)
	}
	p, err := b.Launch(ctx, args, input)
	if err != nil {
		return nil, err
	}
	c.Add(p)
	return p, nil
}

type Binary struct {
	Launch   Launch
	Validate Validate
}

type Launch func(context Context, args string, input process.Process) (process.Process, error)

type Validate func(context Context, args []string) []error

type Context struct {
	Env           *Environment
	ParentProcess process.Process
	Hostname      string
	Directory     []string
}

var registry = map[string]Binary{}

func Register(name string, b Binary) {
	registry[name] = b
}

func (e *Environment) Add(hostname string, c *computer.Computer) {
	e.computers[hostname] = c
}
