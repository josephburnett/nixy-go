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
	job process.P
	// jobInteractive: the job takes over the terminal (ssh), so every
	// key, Ctrl+C included, is its to handle.
	jobInteractive   bool
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
			if msg := signalMessage(status); msg != "" {
				s.Write(process.Stderr, process.CharsData(msg+"\n"))
			}
			s.setStatus(status)
		}
	}
	return s.Written.Read()
}

// signalMessage is what bash reports when a foreground job dies of a
// signal. SIGINT (the ^C is already on screen) and SIGPIPE are silent.
func signalMessage(status int) string {
	switch status {
	case process.SigKill.Status():
		return "Killed"
	case process.SigTerm.Status():
		return "Terminated"
	case process.SigHup.Status():
		return "Hangup"
	}
	return ""
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
		if len(d) == 1 && d[0] == process.SigInt && !s.jobInteractive {
			// Ctrl+C: the terminal echoes ^C and the job gets SIGINT.
			s.Write(process.Stdout, process.CharsData("^C\n"))
			s.job.Signal(process.SigInt)
			return nil
		}
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

	// A redirect is opened first, as the shell itself: a command that
	// then fails to start still leaves the file created or emptied.
	if r := p.Last().Redirect; r != nil {
		f, err := s.openRedirect(file.ExpandHome(r.Target(), s.home), r.Append)
		if err != nil {
			s.fail(1, fmt.Sprintf("%s: %v", r.Target(), err))
			return
		}
		defer func() {
			if s.job != nil {
				s.job = &redirected{P: s.job, f: f}
			}
		}()
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
		p, err := s.simulation.Start(s.hostname, s.owner, binaries[i], seg.Argv(), s.currentDirectory)
		if err != nil {
			for _, proc := range stages {
				proc.Kill()
			}
			s.fail(1, err.Error())
			return
		}
		stages = append(stages, p)
	}
	s.job, s.jobInteractive = &pipeline{stages: stages, closed: make([]bool, len(stages))}, false
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
	case "help":
		s.builtinHelp(seg.Args)
		return
	}

	b, err := s.simulation.LookupBinary(s.hostname, s.owner, seg.Name)
	if err != nil {
		s.fail(lookupStatus(err), err.Error())
		return
	}
	p, err := s.simulation.Start(s.hostname, s.owner, b, seg.Argv(), s.currentDirectory)
	if err != nil {
		s.fail(1, err.Error())
		return
	}
	s.job, s.jobInteractive = p, b.Interactive
}

func (s *shell) builtinNx(args []string) {
	nx := s.simulation.Nx
	if nx == nil {
		s.fail(127, "nx: command not found")
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
		output = nx.NxQuest()
	case "log":
		output = nx.NxLog()
	case "panic":
		output = nx.NxPanic()
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
		if !exited && i < last {
			if _, readerDone := p.stages[i+1].Exit(); readerDone {
				// Writing to a pipe nobody reads: SIGPIPE (`yes | head`).
				st.Signal(process.SigPipe)
			}
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

// Signal goes to every stage, as Ctrl+C reaches the whole foreground
// process group.
func (p *pipeline) Signal(sig process.Signal) {
	for _, st := range p.stages {
		st.Signal(sig)
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
// it, including whether it writes stdout and may feed `|`. The shell
// adds only the segment-completion keys (Enter, `|`) and the optional
// space after `|`.
func (s *shell) Next() []process.Datum {
	if _, exited := s.Exit(); exited {
		return nil
	}
	if s.job != nil {
		next := s.job.Next()
		if !s.jobInteractive {
			// Ctrl+C is always a way out of a running command.
			next = append(next, process.SigInt)
		}
		return next
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
	sug := command.ValidCommand(ctx, last.Text(), s.commandNames(ctx.InPipe), s.validator)
	printsOut := sug.Complete && sug.Writes && s.pipeable(last.Name)
	if r := last.Redirect; r != nil {
		// The command before `>` was committed when `>` was typed.
		if !printsOut {
			return valid
		}
		return append(valid, s.redirectKeys(ctx, p, *r)...)
	}
	for _, r := range sug.Chars {
		valid = append(valid, process.Chars(string(r)))
	}
	if last.Name == "" && strings.HasSuffix(s.currentCommand, "|") {
		valid = append(valid, process.Chars(" ")) // `ls | grep`
	}
	if sug.Complete {
		valid = append(valid, process.TermEnter)
		if printsOut {
			valid = append(valid, process.Chars("|"))
			if live(redirectTarget(ctx, p, "")) {
				valid = append(valid, process.Chars(">"))
			}
		}
	}
	return valid
}

// redirectKeys are the keys after `>`: a second `>` (append), one
// optional space, then the target file, and Enter on a valid one.
func (s *shell) redirectKeys(ctx simulation.ArgContext, p command.Pipeline, r command.Redirect) []process.Datum {
	var keys []process.Datum
	tok := strings.TrimPrefix(r.Raw, " ")
	if strings.Contains(tok, " ") {
		return nil
	}
	if r.Raw == "" {
		if !r.Append {
			keys = append(keys, process.Chars(">"))
		}
		keys = append(keys, process.Chars(" "))
	}
	sug := redirectTarget(ctx, p, tok)
	for _, c := range sug.Chars {
		keys = append(keys, process.Chars(string(c)))
	}
	if sug.Complete {
		keys = append(keys, process.TermEnter)
	}
	return keys
}

// redirectTarget validates the file after `>`. The shell opens it as
// itself (not as root under sudo), and it can't be a file the line
// reads.
func redirectTarget(ctx simulation.ArgContext, p command.Pipeline, tok string) simulation.Suggestion {
	ctx.InPipe = false
	var operands []string
	for _, seg := range p.Segments {
		operands = append(operands, seg.Args...)
	}
	return command.RedirectTarget(ctx, operands, tok)
}

func live(s simulation.Suggestion) bool { return s.Complete || len(s.Chars) > 0 }

// openRedirect opens the target of `>` (truncating it) or `>>` as the
// shell's owner, creating it if need be, before the command runs.
func (s *shell) openRedirect(target string, appendTo bool) (*file.F, error) {
	c, err := s.simulation.GetComputer(s.hostname)
	if err != nil {
		return nil, err
	}
	path := file.Resolve(s.currentDirectory, target)
	if len(path) == 0 {
		return nil, file.ErrIsDir
	}
	f, err := c.Filesystem.Lookup(path, s.owner)
	switch {
	case err == nil:
		if f.Type == file.Folder {
			return nil, file.ErrIsDir
		}
		if !f.CanWrite(s.owner) {
			return nil, file.ErrPermission
		}
		if !appendTo {
			f.Data = ""
		}
		return f, nil
	case errors.Is(err, file.ErrNotExist):
		f = &file.F{
			Type:             file.Text,
			Owner:            s.owner,
			OwnerPermission:  file.Write,
			CommonPermission: file.Read,
		}
		if err := c.Filesystem.CreateFile(path[:len(path)-1], path[len(path)-1], f, s.owner); err != nil {
			return nil, err
		}
		return f, nil
	}
	return nil, err
}

// redirected is a job whose stdout goes to a file. It holds the file
// itself, like an open file descriptor: moving or deleting the file
// doesn't stop the writes.
type redirected struct {
	process.P
	f *file.F
}

func (r *redirected) Read() []process.Output {
	var out []process.Output
	for _, o := range r.P.Read() {
		if o.Stream == process.Stdout {
			r.f.Data += o.Data.String()
			continue
		}
		out = append(out, o)
	}
	return out
}

// builtinHelp is bash's help: what the shell's own commands do. Other
// commands have manual pages (man).
var builtinHelp = map[string]string{
	"cd":   "cd [DIR]\n    Change the current folder to DIR. With no DIR, go home (~).\n",
	"exit": "exit\n    Leave this shell: log out, or end an ssh session.\n",
	"help": "help [BUILTIN]\n    Describe the shell's builtins. For other commands, use man.\n",
	"nx": "nx quest|log|panic\n    The game's own command: the current quest, what you've done,\n" +
		"    or panic to put every machine's /bin back and return to the laptop.\n",
}

func (s *shell) builtinHelp(args []string) {
	if len(args) == 0 {
		var names []string
		for name := range builtinArgs {
			if _, ok := s.builtin(name); ok {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		var sb strings.Builder
		sb.WriteString("Shell builtins (help NAME for more; man COMMAND for the rest):\n")
		for _, n := range names {
			sb.WriteString("    " + n + "\n")
		}
		s.Write(process.Stdout, process.CharsData(sb.String()))
	} else {
		s.Write(process.Stdout, process.CharsData(builtinHelp[args[0]]))
	}
	s.setStatus(0)
}

// helpArgs is `help [BUILTIN]`.
func helpArgs(ctx simulation.ArgContext, partial string) simulation.Suggestion {
	topic := command.OneOf(func(ctx simulation.ArgContext) []string {
		var names []string
		for name := range builtinHelp {
			if name != "nx" || ctx.Sim.Nx != nil {
				names = append(names, name)
			}
		}
		return names
	})
	return command.Positional{Args: []command.Arg{topic}}.Validate(ctx, partial)
}

// builtinArgs are the validators for the shell's builtins.
var builtinArgs = map[string]simulation.ValidArgs{
	"cd":   cdArgs,
	"exit": command.NoArgs,
	"help": helpArgs,
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
			if _, ok := s.builtin(name); ok {
				names = append(names, name)
			}
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
	if va, ok := s.builtin(name); ok {
		return va, true
	}
	b, err := s.simulation.LookupBinary(s.hostname, s.owner, name)
	if err != nil {
		return nil, false
	}
	return b.ValidArgs, true
}

// builtin returns a builtin's validator. nx exists only in a game:
// a simulation without one has no nx to offer.
func (s *shell) builtin(name string) (simulation.ValidArgs, bool) {
	if name == "nx" && s.simulation.Nx == nil {
		return nil, false
	}
	va, ok := builtinArgs[name]
	return va, ok
}

// IsBuiltin reports whether name is one of the shell's own commands
// (cd, exit, help, nx), which it runs itself rather than from /bin.
func IsBuiltin(name string) bool {
	_, ok := builtinArgs[name]
	return ok
}

// pipeable reports whether a command can sit on either side of `|` at
// all: a binary that doesn't take over the terminal. Builtins change
// the shell itself, which a pipeline segment can't do. Whether it
// prints or reads stdin is up to its validator.
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

// Signal: an interactive shell ignores SIGINT and SIGTERM, as bash
// does (Ctrl+C at a prompt, or `kill` of the shell, does nothing).
// SIGHUP ends it and its job; SIGKILL kills it.
func (s *shell) Signal(sig process.Signal) {
	switch sig {
	case process.SigKill:
		s.Kill()
	case process.SigHup:
		if s.job != nil {
			s.job.Signal(process.SigHup)
			s.job = nil
		}
		s.Finish(sig.Status())
	}
}

// Kill ends the shell and its job.
func (s *shell) Kill() {
	if s.job != nil {
		s.job.Kill()
		s.job = nil
	}
	s.Written.Kill()
}
