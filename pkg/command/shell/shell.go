package shell

import (
	"errors"
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

// launch starts a shell in cwd. With -l it is a login shell (the laptop
// login, ssh), which starts in the owner's home instead.
func launch(
	sim *simulation.S,
	owner string,
	hostname string,
	cwd []string,
	args []string,
) (process.P, error) {
	s := &shell{
		simulation:       sim,
		owner:            owner,
		hostname:         hostname,
		home:             file.Home(owner),
		currentDirectory: cwd,
	}
	if len(args) == 1 && args[0] == "-l" {
		// Like login(1): fall back to / when home can't be entered.
		s.currentDirectory = []string{}
		if s.canEnter(s.home) {
			s.currentDirectory = append([]string{}, s.home...)
		}
	}
	return s, nil
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
	process.Written // what the shell writes itself: echo, errors, nx

	simulation *simulation.S
	owner      string
	hostname   string
	home       []string // $HOME: where login and bare `cd` land

	// job is the command line running in the foreground: a command, a
	// pipeline, or the remote shell of an ssh. Keys go to it until it
	// exits.
	job              process.P
	currentDirectory []string
	currentCommand   string
	status           int  // $?: the last command line's exit status
	hasStatus        bool // a command line has finished
}

// Read returns the shell's own output and its job's, in order. When the
// job exits the shell reaps it, keeping its status as $?.
func (s *shell) Read() []process.Output {
	if s.job != nil {
		s.writeAll(s.job.Read())
		if status, done := s.job.Exit(); done {
			s.writeAll(s.job.Read())
			s.job = nil
			s.setStatus(status)
		}
	}
	return s.Written.Read()
}

func (s *shell) writeAll(outs []process.Output) {
	for _, o := range outs {
		s.Write(o.Stream, o.Data)
	}
}

func (s *shell) setStatus(status int) {
	s.status, s.hasStatus = status, true
}

// fail reports a command line that couldn't run.
func (s *shell) fail(status int, msg string) {
	s.Write(process.Stderr, process.CharsData(msg+"\n"))
	s.setStatus(status)
}

func (s *shell) Stdin(d process.Data) error {
	if _, exited := s.Exit(); exited {
		return process.ErrNoInput
	}
	if s.job != nil {
		return s.job.Stdin(d)
	}

	if len(d) != 1 {
		return fmt.Errorf("shell only supports 1 datum at a time: %v", len(d))
	}
	in := d[0]
	s.Write(process.Stdout, process.Data{in})

	switch in := in.(type) {
	case process.Chars:
		s.currentCommand += string(in)
	case process.TermCode:
		switch in {
		case process.TermEnter:
			cmd := strings.TrimSpace(s.currentCommand)
			s.currentCommand = ""
			if cmd != "" {
				s.hasStatus = false
				s.executeCommand(cmd)
			}
		case process.TermBackspace:
			if len(s.currentCommand) > 0 {
				s.currentCommand = s.currentCommand[:len(s.currentCommand)-1]
			}
		case process.TermClear:
			s.currentCommand = ""
		}
	default:
		return fmt.Errorf("unhandled input type %T", in)
	}
	return nil
}

// CloseStdin is end of input to the shell (Ctrl+D): its job's input
// ends, or with no job the shell exits, like `exit`.
func (s *shell) CloseStdin() {
	if s.job != nil {
		s.job.CloseStdin()
		return
	}
	s.Finish(s.status)
}

// executeCommand runs one command line. Whatever it starts becomes the
// job; a builtin or a failure to start sets $? right away.
func (s *shell) executeCommand(cmd string) {
	p := command.Parse(cmd)
	if len(p.Segments) == 0 {
		return
	}
	// Tilde expansion happens here, before any command sees its args,
	// as in a real shell (so `sudo cat ~/x` is still the player's ~).
	for i := range p.Segments {
		for j, a := range p.Segments[i].Args {
			p.Segments[i].Args[j] = file.ExpandHome(a, s.home)
		}
	}

	if !p.IsPipeline() {
		s.executeSingle(p.Segments[0])
		return
	}

	// Pipeline: find every binary before launching any, the way a real
	// shell starts all segments together. Our commands act at launch, so
	// `sudo rm /bin/ls | ls` would otherwise lose ls halfway through.
	binaries := make([]*simulation.Binary, len(p.Segments))
	for i, seg := range p.Segments {
		if seg.Name == "" {
			s.fail(2, "syntax error near unexpected token `|'")
			return
		}
		b, err := s.simulation.LookupBinary(s.hostname, s.owner, seg.Name)
		if err != nil {
			s.fail(lookupStatus(err), err.Error())
			return
		}
		binaries[i] = b
	}
	var stages []process.P
	for i, seg := range p.Segments {
		p, err := binaries[i].Launch(s.simulation, s.owner, s.hostname, s.currentDirectory, seg.Args)
		if err != nil {
			for _, proc := range stages {
				proc.Kill()
			}
			s.fail(1, err.Error())
			return
		}
		stages = append(stages, p)
	}
	s.job = &pipeline{stages: stages, closed: make([]bool, len(stages))}
}

// lookupStatus is the shell's status for a command it couldn't start:
// 126 when it isn't allowed to run, 127 when there's no such command.
func lookupStatus(err error) int {
	if errors.Is(err, file.ErrPermission) {
		return 126
	}
	return 127
}

func (s *shell) executeSingle(seg command.Segment) {
	switch seg.Name {
	case "":
		return
	case "cd":
		if err := s.builtinCd(seg.Args); err != nil {
			s.fail(1, err.Error())
			return
		}
		s.setStatus(0)
		return
	case "exit":
		s.setStatus(s.status)
		s.Finish(s.status)
		return
	case "nx":
		s.builtinNx(seg.Args)
		return
	}

	b, err := s.simulation.LookupBinary(s.hostname, s.owner, seg.Name)
	if err != nil {
		s.fail(lookupStatus(err), err.Error())
		return
	}
	p, err := b.Launch(s.simulation, s.owner, s.hostname, s.currentDirectory, seg.Args)
	if err != nil {
		s.fail(1, err.Error())
		return
	}
	s.job = p
}

func (s *shell) builtinNx(args []string) {
	if DefaultNxHandler == nil {
		s.fail(1, "nx: game not initialized")
		return
	}
	if len(args) == 0 {
		s.Write(process.Stdout, process.CharsData("Usage: nx quest|log|panic\n"))
		s.setStatus(0)
		return
	}
	var output string
	switch args[0] {
	case "quest":
		output = DefaultNxHandler.NxQuest()
	case "log":
		output = DefaultNxHandler.NxLog()
	case "panic":
		output = DefaultNxHandler.NxPanic()
	default:
		s.fail(1, fmt.Sprintf("nx: unknown subcommand '%v'", args[0]))
		return
	}
	s.Write(process.Stdout, process.CharsData(output))
	s.setStatus(0)
	if args[0] == "panic" {
		s.Finish(0) // disconnect to the laptop
	}
}

// Info is what the game reads from a shell. Hostname, CurrentDirectory
// and CurrentCommand describe the innermost shell (the one the player
// is typing into, at the end of any ssh sessions).
type Info interface {
	Hostname() string
	CurrentDirectory() []string
	CurrentCommand() string
	// Active is the innermost shell itself.
	Active() Info
	// LastStatus is $?: the exit status of the last command line this
	// shell ran (0 is success). ok is false before one has finished
	// and while one is still running, such as an ssh session.
	LastStatus() (status int, ok bool)
}

var _ Info = &shell{}

func (s *shell) innermost() *shell {
	if child, ok := s.job.(*shell); ok {
		return child.innermost()
	}
	return s
}

func (s *shell) Active() Info               { return s.innermost() }
func (s *shell) Hostname() string           { return s.innermost().hostname }
func (s *shell) CurrentDirectory() []string { return s.innermost().currentDirectory }
func (s *shell) CurrentCommand() string     { return s.innermost().currentCommand }

func (s *shell) LastStatus() (int, bool) {
	if s.job != nil || !s.hasStatus {
		return 0, false
	}
	return s.status, true
}

// pipeline connects its stages: the stdout of each feeds the stdin of
// the next, and when one exits the next one's input closes. Stderr of
// every stage, and the last stage's stdout, go to the terminal. Its
// exit status is the last stage's, as in a shell.
type pipeline struct {
	stages []process.P
	closed []bool // closed[i]: stage i+1's stdin was closed
}

var _ process.P = &pipeline{}

// Read moves data one step through the pipeline: each stage is read
// once, in order, so data from a stage reaches the next in the same
// call. A stage that never stops writing can't hang the caller.
func (p *pipeline) Read() []process.Output {
	var out []process.Output
	last := len(p.stages) - 1
	for i, st := range p.stages {
		outs := st.Read()
		_, exited := st.Exit()
		if exited {
			outs = append(outs, st.Read()...)
		}
		for _, o := range outs {
			if o.Stream == process.Stdout && i < last {
				// A reader that has exited drops it, as a closed pipe would.
				_ = p.stages[i+1].Stdin(o.Data)
			} else {
				out = append(out, o)
			}
		}
		if exited && i < last && !p.closed[i] {
			p.closed[i] = true
			p.stages[i+1].CloseStdin()
		}
	}
	return out
}

func (p *pipeline) Exit() (int, bool) {
	for _, st := range p.stages {
		if _, exited := st.Exit(); !exited {
			return 0, false
		}
	}
	return p.stages[len(p.stages)-1].Exit()
}

func (p *pipeline) Stdin(in process.Data) error { return p.stages[0].Stdin(in) }
func (p *pipeline) CloseStdin()                 { p.stages[0].CloseStdin() }
func (p *pipeline) Next() []process.Datum       { return nil }
func (p *pipeline) Owner() string               { return p.stages[0].Owner() }

func (p *pipeline) Kill() {
	for _, st := range p.stages {
		st.Kill()
	}
}

// builtinCd changes directory; bare `cd` goes home.
func (s *shell) builtinCd(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("cd: too many arguments")
	}
	target := "/" + strings.Join(s.home, "/")
	if len(args) == 1 {
		target = args[0]
	}
	newDir := file.Resolve(s.currentDirectory, target)
	if err := s.enter(newDir); err != nil {
		return fmt.Errorf("cd: %v: %v", target, err)
	}
	s.currentDirectory = newDir
	return nil
}

// enter checks that the owner can make dir the cwd: a reachable,
// readable folder.
func (s *shell) enter(dir []string) error {
	c, err := s.simulation.GetComputer(s.hostname)
	if err != nil {
		return err
	}
	_, err = c.Filesystem.LookupDir(dir, s.owner)
	return err
}

func (s *shell) canEnter(dir []string) bool { return s.enter(dir) == nil }

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
	if _, exited := s.Exit(); exited {
		return nil
	}
	if s.job != nil {
		return s.job.Next()
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
		User:     s.owner,
		Home:     s.home,
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
	"cd":   cdArgs,
	"exit": command.NoArgs,
	"nx": command.Subcommands(map[string]simulation.ValidArgs{
		"quest": command.NoArgs,
		"log":   command.NoArgs,
		"panic": command.NoArgs,
	}),
}

// cdArgs is `cd [DIR]`. Bare `cd` goes home, so like `cd DIR` it is
// offered only when it would move: not when already home.
func cdArgs(ctx simulation.ArgContext, partial string) simulation.Suggestion {
	sug := command.Positional{Args: []command.Arg{command.CdTarget}}.Validate(ctx, partial)
	if partial == "" {
		sug.Complete = command.CdTarget(ctx, nil, "/"+strings.Join(ctx.Home, "/")).Complete
	}
	return sug
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
	for _, name := range s.simulation.Commands(s.hostname, s.owner) {
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
	b, err := s.simulation.LookupBinary(s.hostname, s.owner, name)
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
	b, err := s.simulation.LookupBinary(s.hostname, s.owner, name)
	return err == nil && !b.Interactive
}

func (s *shell) Owner() string {
	return s.owner
}

// Kill ends the shell and its job.
func (s *shell) Kill() {
	if s.job != nil {
		s.job.Kill()
		s.job = nil
	}
	s.Written.Kill()
}
