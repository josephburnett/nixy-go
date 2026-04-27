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
	OptionalArgs bool      // true if the command can run with zero arguments (standalone, no pipe required)
	PipeReceiver bool      // true if the command can read stdin from an upstream pipe (cat with no args, grep with just a pattern)
}

// Suggestion is what an argument validator returns to the shell:
//   - Chars are the characters the player may type next as part of the
//     argument (e.g. continuations of a path prefix, including `/` to
//     descend, or printable bytes for a free-form pattern).
//   - Complete reports whether the current partial input forms a complete,
//     submittable argument (e.g. exact-match against an existing file).
//
// The shell composes Enter and `|` on top of Complete. Validators do not
// need to return TermEnter directly; they answer "what continuations?"
// and "is this submittable?" and the shell handles segment-completion
// keys.
type Suggestion struct {
	Chars    []rune
	Complete bool
}

// ValidArgs returns the set of valid continuations and whether the
// current partial input is a complete argument. If a binary's ValidArgs
// is nil, the shell allows any printable character as arguments.
type ValidArgs func(
	sim *S,
	hostname string,
	cwd []string,
	partialArgs string,
) Suggestion

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
