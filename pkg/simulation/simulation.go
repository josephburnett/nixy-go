package simulation

import (
	"fmt"
	"sort"

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
	Launch Launch
	// ValidArgs is required: it is the only description of what the
	// player may type after the command name (see ValidArgs).
	ValidArgs ValidArgs
	// Interactive commands take over the terminal (ssh opens a remote
	// shell). They can't be part of a pipeline or run under sudo.
	Interactive bool
}

// Suggestion is what an argument validator returns to the shell:
//   - Chars are the characters the player may type next (continuations
//     of a path, `/` to descend, the space that starts the next
//     argument, ...). Every one must lead toward a Complete state.
//   - Complete reports whether the command can run, as typed, without a
//     usage error. The shell offers Enter and `|` only then.
type Suggestion struct {
	Chars    []rune
	Complete bool
}

// ArgContext is what a validator sees besides the typed text.
type ArgContext struct {
	Sim      *S
	Hostname string
	Cwd      []string
	// User is who the command runs as: the shell's owner, or root
	// under sudo. Completion only looks inside folders User can read.
	User string
	// Home is the shell owner's home folder ($HOME). It stays the
	// player's under sudo, as the shell expands ~ before sudo runs.
	Home []string
	// InPipe is true when the command reads stdin from an upstream
	// pipe, so it may need fewer arguments (`ls | grep x`).
	InPipe bool
}

// ValidArgs validates everything typed after the command name and the
// space that follows it ("" right after `ls `). It decides both which
// keys come next and whether the command line is complete, so zero-arg
// commands answer Complete for "" and pipe receivers answer Complete
// for fewer args when ctx.InPipe.
type ValidArgs func(ctx ArgContext, partialArgs string) Suggestion

type Launch func(
	s *S,
	owner string,
	hostname string,
	cwd []string,
	args []string,
) (process.P, error)

var registry = map[string]*Binary{}

// Register adds a binary to the registry. Launch and ValidArgs are
// required; a missing validator would leave the keyboard with no way to
// know what may be typed.
func Register(name string, b *Binary) {
	if b.Launch == nil || b.ValidArgs == nil {
		panic(fmt.Sprintf("simulation.Register(%q): Launch and ValidArgs are required", name))
	}
	registry[name] = b
}

func GetBinary(name string) (*Binary, error) {
	if b, ok := registry[name]; ok {
		return b, nil
	}
	return nil, fmt.Errorf("binary %v not registered", name)
}

// LookupBinary resolves a command name the way the shell runs it:
// through the executable at /bin/<name> on the host.
func (s *S) LookupBinary(hostname, name string) (*Binary, error) {
	c, err := s.GetComputer(hostname)
	if err != nil {
		return nil, err
	}
	f, err := c.Filesystem.Navigate([]string{"bin", name})
	if err != nil {
		return nil, fmt.Errorf("command not found: %v", name)
	}
	if f.Type != file.Binary {
		return nil, fmt.Errorf("not executable: %v", name)
	}
	return GetBinary(f.Data)
}

// Commands lists the names in /bin on the host, sorted.
func (s *S) Commands(hostname string) []string {
	c, err := s.GetComputer(hostname)
	if err != nil {
		return nil
	}
	bin, err := c.Filesystem.Navigate([]string{"bin"})
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(bin.Files))
	for name, f := range bin.Files {
		if f.Type == file.Binary {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
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
