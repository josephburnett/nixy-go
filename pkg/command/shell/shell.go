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
		Launch:      launch,
		ValidArgs:   command.NoArgs,
		Interactive: true,
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
			// Drain any pending stderr into echoErr before reaping the
			// child. Without this, an errorProcess's message — which lives
			// only on stderr — is lost because the session loop calls
			// Stdout before Stderr, and Kill drops the child.
			for {
				errOut, _, _ := s.childProcess.Stderr()
				if len(errOut) == 0 {
					break
				}
				s.echoErr = append(s.echoErr, errOut...)
			}
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
	p := command.Parse(cmd)
	if len(p.Segments) == 0 {
		return nil
	}

	if !p.IsPipeline() {
		return s.executeSingle(p.Segments[0])
	}

	// Pipeline: launch each segment, wire stdout->stdin
	var processes []process.P
	for _, seg := range p.Segments {
		if seg.Name == "" {
			return fmt.Errorf("empty pipe segment")
		}
		p, err := s.launchBinary(seg)
		if err != nil {
			for _, proc := range processes {
				proc.Kill()
			}
			return err
		}
		processes = append(processes, p)
	}

	s.childProcess = &pipeline{processes: processes}
	return nil
}

func (s *shell) executeSingle(seg command.Segment) error {
	if seg.Name == "" {
		return nil
	}

	switch seg.Name {
	case "cd":
		return s.builtinCd(seg.Args)
	case "exit":
		s.exited = true
		return nil
	case "nx":
		return s.builtinNx(seg.Args)
	}

	p, err := s.launchBinary(seg)
	if err != nil {
		return err
	}
	s.childProcess = p
	return nil
}

func (s *shell) launchBinary(seg command.Segment) (process.P, error) {
	b, err := s.simulation.LookupBinary(s.hostname, seg.Name)
	if err != nil {
		return nil, err
	}
	return b.Launch(s.simulation, s.owner, s.hostname, s.currentDirectory, seg.Args)
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
	newDir := file.Resolve(s.currentDirectory, target)

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

// Next returns the keystrokes that lead to a correct, executable action
// at the current cursor position. The invariant: a player can never type
// something the shell would refuse to run cleanly.
//
// Pipes are handled structurally: command.Parse splits on `|` and only
// the last segment is validated. Earlier segments are committed — `|`
// was only offered when the previous segment was complete. Everything
// about the segment itself (which names, which args, when it's complete)
// comes from the command's ValidArgs, found the same way the shell runs
// it. The shell adds only the segment-completion keys (Enter, `|`) and
// the optional space after `|`.
func (s *shell) Next() []process.Datum {
	if s.exited {
		return nil
	}
	if s.childProcess != nil {
		return s.childProcess.Next()
	}

	var valid []process.Datum
	if len(s.currentCommand) > 0 {
		valid = append(valid, process.TermBackspace)
	}

	p := command.Parse(s.currentCommand)
	last := p.Last()
	ctx := simulation.ArgContext{
		Sim:      s.simulation,
		Hostname: s.hostname,
		Cwd:      s.currentDirectory,
		InPipe:   p.InPipe(),
	}
	text := last.Name
	if last.HasArgs {
		text += " " + last.PartialArgs
	}
	sug := command.ValidCommand(ctx, text, s.commandNames(ctx.InPipe), s.validator)
	for _, r := range sug.Chars {
		valid = append(valid, process.Chars(string(r)))
	}
	if last.Name == "" && strings.HasSuffix(s.currentCommand, "|") {
		valid = append(valid, process.Chars(" ")) // `ls | grep`
	}
	if sug.Complete {
		valid = append(valid, process.TermEnter)
		if s.pipeable(last.Name) {
			valid = append(valid, process.Chars("|"))
		}
	}
	return valid
}

// builtinArgs are the validators for the shell's builtins.
var builtinArgs = map[string]simulation.ValidArgs{
	"cd":   command.Positional{Args: []command.Arg{command.CdTarget}}.Validate,
	"exit": command.NoArgs,
	"nx": command.Subcommands(map[string]simulation.ValidArgs{
		"quest": command.NoArgs,
		"log":   command.NoArgs,
		"panic": command.NoArgs,
	}),
}

// commandNames lists what may be typed in command position: builtins
// and /bin, or in a pipe segment only the binaries that can be piped.
func (s *shell) commandNames(inPipe bool) []string {
	var names []string
	if !inPipe {
		for name := range builtinArgs {
			names = append(names, name)
		}
	}
	for _, name := range s.simulation.Commands(s.hostname) {
		if !inPipe || s.pipeable(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// validator finds a command's args validator: builtins first, then the
// binary at /bin/<name>.
func (s *shell) validator(name string) (simulation.ValidArgs, bool) {
	if va, ok := builtinArgs[name]; ok {
		return va, true
	}
	b, err := s.simulation.LookupBinary(s.hostname, name)
	if err != nil {
		return nil, false
	}
	return b.ValidArgs, true
}

// pipeable reports whether a command can sit on either side of `|`: a
// binary that doesn't take over the terminal. Builtins change the shell
// itself, which a pipeline segment can't do.
func (s *shell) pipeable(name string) bool {
	if _, ok := builtinArgs[name]; ok {
		return false
	}
	b, err := s.simulation.LookupBinary(s.hostname, name)
	return err == nil && !b.Interactive
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
