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

func (s *S) Launch(hostname, owner, binaryName string, args []string, cwd []string) (process.P, error) {
	c, ok := s.computers[hostname]
	if !ok {
		return nil, fmt.Errorf("hostname %v not found", hostname)
	}
	b, ok := registry[binaryName]
	if !ok {
		return nil, fmt.Errorf("binary %v not found", binaryName)
	}
	p, err := b.Launch(s, owner, hostname, cwd, args)
	if err != nil {
		return nil, err
	}
	c.Add(p)
	return p, nil
}

type Binary struct {
	Launch       Launch
	ValidArgs    ValidArgs // optional: returns valid next inputs for arguments
	OptionalArgs bool      // true if the command can run with zero arguments
}

// ValidArgs returns valid next datum values given partial argument input.
// If nil, the shell allows any printable character as arguments.
type ValidArgs func(
	sim *S,
	hostname string,
	cwd []string,
	partialArgs string,
) []process.Datum

type Launch func(
	s *S,
	owner string,
	hostname string,
	cwd []string,
	args []string,
) (process.P, error)

var registry = map[string]*Binary{}

func Register(name string, b *Binary) error {
	// Idempotent: allow re-registration with the same binary.
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
	return nil
}

func (s *S) GetComputer(hostname string) (*computer.C, error) {
	if c, ok := s.computers[hostname]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("host %v not found", hostname)
}
