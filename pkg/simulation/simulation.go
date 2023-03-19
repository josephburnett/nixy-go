package simulation

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/computer"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
)

type State struct {
	FileSystems map[string]*file.F
}

type S struct {
	computers map[string]*computer.C

	state State
}

func New() *S {
	return &S{
		computers: map[string]*computer.C{},

		state: State{
			FileSystems: map[string]*file.F{},
		},
	}
}

func (s *S) Launch(hostname, owner, binaryName string, args string, input process.P) (process.P, error) {
	c, ok := s.computers[hostname]
	if !ok {
		return nil, fmt.Errorf("hostname %v not found", hostname)
	}
	b, ok := registry[binaryName]
	if !ok {
		return nil, fmt.Errorf("binary %v not found", binaryName)
	}
	p, err := b.Launch(s, owner, hostname, []string{}, args, input)
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

type Launch func(
	s *S,
	owner string,
	hostname string,
	cwd []string,
	args string,
	input process.P,
) (process.P, error)

type Test func(
	s *S,
	owner string,
	hostname string,
	cwd []string,
	args []string,
) []error

var registry = map[string]*Binary{}

func Register(name string, b *Binary) error {
	if _, registered := registry[name]; registered {
		return fmt.Errorf("binary %v already registered", name)
	}
	registry[name] = b
	return nil
}

func GetBinary(name string) (*Binary, error) {
	if b, ok := registry[name]; ok {
		return b, nil
	}
	return nil, fmt.Errorf("binary %v not registered", name)
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
	s.state.FileSystems[hostname] = filesystem
	return nil
}

func (s *S) GetComputer(hostname string) (*computer.C, error) {
	if c, ok := s.computers[hostname]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("host %v not found", hostname)
}
