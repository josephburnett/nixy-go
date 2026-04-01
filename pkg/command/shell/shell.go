package shell

import (
	"fmt"
	"sort"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("shell", &simulation.Binary{
		Launch: launch,
	})
}

func launch(
	sim *simulation.S,
	owner string,
	hostname string,
	cwd []string,
	_ []string,
) (process.P, error) {
	return &shell{
		simulation:       sim,
		owner:            owner,
		hostname:         hostname,
		currentDirectory: cwd,
	}, nil
}

var builtins = []string{"cd", "exit", "nx"}

// NxHandler handles nx builtin subcommands. Set by the game layer.
type NxHandler interface {
	NxQuest() string
	NxLog() string
	NxPanic() string
}

// DefaultNxHandler is set by the game layer at startup.
var DefaultNxHandler NxHandler

var _ process.P = &shell{}

type shell struct {
	simulation *simulation.S
	owner      string
	hostname   string
	exited     bool

	childProcess     process.P
	currentDirectory []string
	currentCommand   string
	echoOut          process.Data
	echoErr          process.Data
}

func (s *shell) Stdout() (process.Data, bool, error) {
	if s.exited {
		return nil, true, nil
	}
	data := s.echoOut
	s.echoOut = nil
	if s.childProcess != nil {
		d, eof, err := s.childProcess.Stdout()
		if err != nil && err != command.ErrEndOfFile {
			return data, false, err
		}
		data = append(data, d...)
		if eof {
			s.childProcess.Kill()
			s.childProcess = nil
		}
	}
	return data, false, nil
}

func (s *shell) Stderr() (process.Data, bool, error) {
	if s.exited {
		return nil, true, nil
	}
	data := s.echoErr
	s.echoErr = nil
	if s.childProcess != nil {
		d, eof, err := s.childProcess.Stderr()
		if err != nil {
			return data, false, err
		}
		data = append(data, d...)
		_ = eof // stderr eof doesn't kill the child
	}
	return data, false, nil
}

func (s *shell) Stdin(d process.Data) (bool, error) {
	if s.exited {
		return true, nil
	}
	if s.childProcess != nil {
		return s.childProcess.Stdin(d)
	}

	if len(d) != 1 {
		return false, fmt.Errorf("shell only supports 1 datum at a time: %v", len(d))
	}
	in := d[0]
	s.echoOut = append(s.echoOut, in)

	switch in := in.(type) {
	case process.Chars:
		s.currentCommand += string(in)
		return false, nil
	case process.TermCode:
		switch in {
		case process.TermEnter:
			cmd := strings.TrimSpace(s.currentCommand)
			s.currentCommand = ""
			if cmd == "" {
				return false, nil
			}
			err := s.executeCommand(cmd)
			if err != nil {
				s.echoErr = append(s.echoErr, process.Chars(err.Error()+"\n"))
			}
		case process.TermBackspace:
			if len(s.currentCommand) > 0 {
				s.currentCommand = s.currentCommand[:len(s.currentCommand)-1]
			}
		case process.TermClear:
			s.currentCommand = ""
		}
		return s.exited, nil
	default:
		return false, fmt.Errorf("unhandled input type %T", in)
	}
}

func (s *shell) executeCommand(cmd string) error {
	// Split on pipes
	segments := strings.Split(cmd, "|")

	if len(segments) == 1 {
		return s.executeSingle(strings.TrimSpace(cmd))
	}

	// Pipeline: launch each segment, wire stdout->stdin
	var processes []process.P
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return fmt.Errorf("empty pipe segment")
		}
		p, err := s.launchBinary(seg)
		if err != nil {
			// Kill already-launched processes
			for _, proc := range processes {
				proc.Kill()
			}
			return err
		}
		processes = append(processes, p)
	}

	// Wire: pipe stdout of each process to stdin of the next
	s.childProcess = &pipeline{processes: processes}
	return nil
}

func (s *shell) executeSingle(cmd string) error {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return nil
	}
	name := fields[0]
	args := fields[1:]

	// Builtins
	switch name {
	case "cd":
		return s.builtinCd(args)
	case "exit":
		s.exited = true
		return nil
	case "nx":
		return s.builtinNx(args)
	}

	p, err := s.launchBinary(cmd)
	if err != nil {
		return err
	}
	s.childProcess = p
	return nil
}

func (s *shell) launchBinary(cmd string) (process.P, error) {
	fields := strings.Fields(cmd)
	name := fields[0]
	args := fields[1:]

	c, err := s.simulation.GetComputer(s.hostname)
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
	b, err := simulation.GetBinary(f.Data)
	if err != nil {
		return nil, err
	}
	return b.Launch(s.simulation, s.owner, s.hostname, s.currentDirectory, args)
}

func (s *shell) builtinNx(args []string) error {
	if DefaultNxHandler == nil {
		return fmt.Errorf("nx: game not initialized")
	}
	if len(args) == 0 {
		s.echoOut = append(s.echoOut, process.Chars("Usage: nx quest|log|panic\n"))
		return nil
	}
	var output string
	switch args[0] {
	case "quest":
		output = DefaultNxHandler.NxQuest()
	case "log":
		output = DefaultNxHandler.NxLog()
	case "panic":
		output = DefaultNxHandler.NxPanic()
		s.exited = true // Disconnect to laptop
	default:
		return fmt.Errorf("nx: unknown subcommand '%v'", args[0])
	}
	s.echoOut = append(s.echoOut, process.Chars(output))
	return nil
}

// Hostname returns the hostname of the innermost active shell.
func (s *shell) Hostname() string {
	if child, ok := s.childProcess.(*shell); ok {
		return child.Hostname()
	}
	return s.hostname
}

// CurrentDirectory returns the cwd of the innermost active shell.
func (s *shell) CurrentDirectory() []string {
	if child, ok := s.childProcess.(*shell); ok {
		return child.CurrentDirectory()
	}
	return s.currentDirectory
}

// CurrentCommand returns what the user has typed in the innermost active shell.
func (s *shell) CurrentCommand() string {
	if child, ok := s.childProcess.(*shell); ok {
		return child.CurrentCommand()
	}
	return s.currentCommand
}

// pipeline connects multiple processes: stdout of N feeds stdin of N+1.
type pipeline struct {
	processes []process.P
	piped     bool
}

func (p *pipeline) Stdout() (process.Data, bool, error) {
	if !p.piped {
		p.pipe()
	}
	// Read from last process
	last := p.processes[len(p.processes)-1]
	return last.Stdout()
}

func (p *pipeline) Stderr() (process.Data, bool, error) {
	// Collect stderr from all processes
	var all process.Data
	for _, proc := range p.processes {
		d, _, _ := proc.Stderr()
		all = append(all, d...)
	}
	return all, false, nil
}

func (p *pipeline) Stdin(in process.Data) (bool, error) {
	// Write to first process
	return p.processes[0].Stdin(in)
}

func (p *pipeline) Next() []process.Datum { return nil }
func (p *pipeline) Owner() string         { return p.processes[0].Owner() }

func (p *pipeline) Kill() error {
	for _, proc := range p.processes {
		proc.Kill()
	}
	return nil
}

func (p *pipeline) pipe() {
	p.piped = true
	for i := 0; i < len(p.processes)-1; i++ {
		src := p.processes[i]
		dst := p.processes[i+1]
		// Read all stdout from src and feed to dst
		for {
			data, eof, _ := src.Stdout()
			if len(data) > 0 {
				dst.Stdin(data)
			}
			if eof {
				dst.Kill() // Signal EOF to dst
				break
			}
		}
	}
}

func (s *shell) builtinCd(args []string) error {
	if len(args) == 0 {
		s.currentDirectory = []string{}
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("cd: too many arguments")
	}
	target := args[0]
	var newDir []string

	if strings.HasPrefix(target, "/") {
		// Absolute path
		parts := strings.Split(strings.TrimPrefix(target, "/"), "/")
		newDir = filterEmpty(parts)
	} else {
		// Relative path
		parts := strings.Split(target, "/")
		newDir = append([]string{}, s.currentDirectory...)
		for _, p := range parts {
			if p == ".." {
				if len(newDir) > 0 {
					newDir = newDir[:len(newDir)-1]
				}
			} else if p != "" && p != "." {
				newDir = append(newDir, p)
			}
		}
	}

	// Validate directory exists
	c, err := s.simulation.GetComputer(s.hostname)
	if err != nil {
		return err
	}
	f, err := c.Filesystem.Navigate(newDir)
	if err != nil {
		return fmt.Errorf("cd: %v", err)
	}
	if f.Type != file.Folder {
		return fmt.Errorf("cd: %v: not a directory", target)
	}
	s.currentDirectory = newDir
	return nil
}

func (s *shell) Next() []process.Datum {
	if s.exited {
		return nil
	}
	if s.childProcess != nil {
		return s.childProcess.Next()
	}

	var valid []process.Datum

	// Backspace if there's text
	if len(s.currentCommand) > 0 {
		valid = append(valid, process.TermBackspace)
	}

	// Collect all valid command names
	cmdNames := s.availableCommands()

	if s.currentCommand == "" {
		// At empty prompt: first char of any valid command
		firstChars := map[byte]bool{}
		for _, name := range cmdNames {
			if len(name) > 0 {
				firstChars[name[0]] = true
			}
		}
		for ch := range firstChars {
			valid = append(valid, process.Chars(string(ch)))
		}
	} else if strings.Contains(s.currentCommand, " ") {
		// After command name + space: we're in arguments.
		// For now, allow any printable character, space, enter.
		for c := byte(32); c < 127; c++ {
			valid = append(valid, process.Chars(string(c)))
		}
		valid = append(valid, process.TermEnter)
	} else {
		// Mid-command-name: chars that continue toward a valid command
		continuations := map[byte]bool{}
		exactMatch := false
		for _, name := range cmdNames {
			if strings.HasPrefix(name, s.currentCommand) {
				if name == s.currentCommand {
					exactMatch = true
				}
				if len(name) > len(s.currentCommand) {
					continuations[name[len(s.currentCommand)]] = true
				}
			}
		}
		for ch := range continuations {
			valid = append(valid, process.Chars(string(ch)))
		}
		if exactMatch {
			// Can press space to start args or enter to execute
			valid = append(valid, process.Chars(" "))
			valid = append(valid, process.TermEnter)
		}
	}

	return valid
}

func (s *shell) availableCommands() []string {
	names := append([]string{}, builtins...)
	c, err := s.simulation.GetComputer(s.hostname)
	if err != nil {
		return names
	}
	binDir, err := c.Filesystem.Navigate([]string{"bin"})
	if err != nil {
		return names
	}
	for name := range binDir.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *shell) Owner() string {
	return s.owner
}

func (s *shell) Kill() error {
	s.exited = true
	if s.childProcess != nil {
		s.childProcess.Kill()
		s.childProcess = nil
	}
	return nil
}

func filterEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
