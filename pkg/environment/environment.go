package environment

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/computer"
	"github.com/josephburnett/nixy-go/pkg/process"
)

type Environment struct {
	computers map[string]*computer.Computer
}

func NewEnvironment() (*Environment, error) {
	return &Environment{
		computers: map[string]*computer.Computer{},
	}, nil
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

func (e *Environment) AddComputer(hostname string, c *computer.Computer) error {
	if _, present := e.computers[hostname]; present {
		return fmt.Errorf("host %v already present", hostname)
	}
	e.computers[hostname] = c
	return nil
}
