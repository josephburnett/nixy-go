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
	c, err := s.simulation.GetComputer(s.hostname)
	if err != nil {
		return nil, err
	}
	f, err := c.Filesystem.Navigate([]string{"bin", seg.Name})
	if err != nil {
		return nil, fmt.Errorf("command not found: %v", seg.Name)
	}
	if f.Type != file.Binary {
		return nil, fmt.Errorf("not executable: %v", seg.Name)
	}
	b, err := simulation.GetBinary(f.Data)
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
// something the shell would refuse to run cleanly. Enter and `|` are
// shell-level "execute this segment" keys, valid only when the current
// segment is complete.
//
// Pipes are handled structurally: command.Parse splits on `|` and we
// validate only the last segment. Earlier segments are committed —
// typing `|` was only allowed when the previous segment was complete.
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
	// inPipe: at least one earlier segment exists, so the current segment
	// runs as a pipe receiver. Commands marked PipeReceiver can execute
	// with fewer args when piped (cat with zero args, grep with just a
	// pattern) because their stdin comes from upstream rather than
	// terminal input.
	inPipe := p.InPipe()

	cmdNames := s.availableCommands()

	if last.Name == "" {
		// Fresh prompt or fresh segment after a pipe.
		firstChars := map[byte]bool{}
		for _, name := range cmdNames {
			if len(name) > 0 {
				firstChars[name[0]] = true
			}
		}
		for ch := range firstChars {
			valid = append(valid, process.Chars(string(ch)))
		}
		return valid
	}

	if last.HasArgs {
		// Args mode for the current segment.
		argValidator := s.getArgValidator(last.Name)
		if argValidator != nil {
			argValid := argValidator(s.simulation, s.hostname, s.currentDirectory, last.PartialArgs)
			valid = append(valid, argValid...)
			// `|` is shell-level — it's valid wherever Enter would be (i.e.
			// when the validator considers the arg complete).
			if datumsContainEnter(argValid) {
				valid = append(valid, process.Chars("|"))
			}
			// Commands with optional args (ls, pwd) can run as-is when
			// the partial arg is empty — allow Enter and `|` here too.
			if last.PartialArgs == "" && commandOptionalArgs(last.Name) {
				valid = append(valid, process.TermEnter)
				valid = append(valid, process.Chars("|"))
			}
			// Pipe-receiver position: command reads from upstream pipe, so
			// fewer args are needed. e.g. `ls | grep target<Enter>` runs
			// stdinGrep on ls's output.
			if inPipe && commandReadyInPipe(last.Name, last.PartialArgs) {
				valid = append(valid, process.TermEnter)
				valid = append(valid, process.Chars("|"))
			}
		} else {
			// No validator: any printable + Enter + |
			for c := byte(32); c < 127; c++ {
				valid = append(valid, process.Chars(string(c)))
			}
			valid = append(valid, process.TermEnter)
			valid = append(valid, process.Chars("|"))
		}
		return valid
	}

	// Mid-command-name for the current segment.
	continuations := map[byte]bool{}
	exactMatch := false
	for _, name := range cmdNames {
		if strings.HasPrefix(name, last.Name) {
			if name == last.Name {
				exactMatch = true
			}
			if len(name) > len(last.Name) {
				continuations[name[len(last.Name)]] = true
			}
		}
	}
	for ch := range continuations {
		valid = append(valid, process.Chars(string(ch)))
	}
	if exactMatch {
		valid = append(valid, process.Chars(" ")) // can always type args
		// Enter and `|` only when the segment is executable as-is. Two cases:
		// - OptionalArgs (e.g. `ls<Enter>` standalone)
		// - PipeReceiver in pipe position (e.g. `ls | cat<Enter>` — cat
		//   runs as stdinCat reading ls's output).
		// Otherwise the binary itself would reject zero args ("missing
		// operand"), violating the "no mistakes" invariant.
		if commandOptionalArgs(last.Name) {
			valid = append(valid, process.TermEnter)
			valid = append(valid, process.Chars("|"))
		} else if inPipe && commandReadyInPipe(last.Name, "") {
			valid = append(valid, process.TermEnter)
			valid = append(valid, process.Chars("|"))
		}
	}
	return valid
}

// commandReadyInPipe reports whether a command segment is ready to
// execute as a pipe receiver, given its current partial args.
//
//   - cat: always ready in pipe context — bare cat reads stdin from
//     upstream.
//   - grep: ready when at least one non-space char of pattern has been
//     typed; without a pattern, grep errors ("missing pattern").
//   - others: not ready in pipe context.
//
// This is the per-command rule the keyboard uses to decide when Enter
// is valid in pipe-receiver position.
func commandReadyInPipe(cmdName, partialArgs string) bool {
	b, err := simulation.GetBinary(cmdName)
	if err != nil || !b.PipeReceiver {
		return false
	}
	switch cmdName {
	case "cat":
		return true // any args (or none) are fine in pipe context
	case "grep":
		// Need at least one pattern char (non-space).
		return strings.TrimSpace(partialArgs) != ""
	}
	return false
}

// commandOptionalArgs reports whether the named command (binary or
// builtin) can run with zero args. Used to gate Enter/| at exact-match
// points so the player can never execute a command that would error on
// zero args.
func commandOptionalArgs(name string) bool {
	switch name {
	case "cd", "exit": // builtins that take no args (cd → /, exit terminates)
		return true
	case "nx": // requires subcommand
		return false
	}
	if b, err := simulation.GetBinary(name); err == nil {
		return b.OptionalArgs
	}
	return false
}

func datumsContainEnter(ds []process.Datum) bool {
	for _, d := range ds {
		if t, ok := d.(process.TermCode); ok && t == process.TermEnter {
			return true
		}
	}
	return false
}

func (s *shell) getArgValidator(cmdName string) simulation.ValidArgs {
	// Check builtins first
	switch cmdName {
	case "cd":
		return command.ValidArgsCd
	}
	// Check binary registry
	c, err := s.simulation.GetComputer(s.hostname)
	if err != nil {
		return nil
	}
	f, err := c.Filesystem.Navigate([]string{"bin", cmdName})
	if err != nil {
		return nil
	}
	b, err := simulation.GetBinary(f.Data)
	if err != nil {
		return nil
	}
	return b.ValidArgs
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
