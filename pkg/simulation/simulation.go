package simulation

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/computer"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
)

type S struct {
	computers map[string]*computer.C
}

func New() *S {
	return &S{
		computers: map[string]*computer.C{},
	}
}

func (s *S) Launch(hostname, binaryName string, ctx Context, args string, input process.P) (process.P, error) {
	c, ok := s.computers[hostname]
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
	Launch Launch
	Test   Test
}

type Launch func(context Context, args string, input process.P) (process.P, error)

type Test func(context Context, args []string) []error

type Context struct {
	Simulation    *S
	ParentProcess process.P
	Hostname      string
	Directory     []string
}

var registry = map[string]Binary{}

func Register(name string, b Binary) error {
	if _, registered := registry[name]; registered {
		return fmt.Errorf("binary %v already registered", name)
	}
	registry[name] = b
	return nil
}

func (s *S) Boot(hostname string, filesystem *file.F) error {
	if _, present := s.computers[hostname]; present {
		return fmt.Errorf("host %v already present", hostname)
	}
	c := computer.New(filesystem)
	err := c.Boot()
	if err != nil {
		return err
	}
	s.computers[hostname] = c
	return nil
}
