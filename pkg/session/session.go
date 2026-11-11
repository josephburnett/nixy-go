package session

import (
	"strings"

	shellpkg "github.com/josephburnett/nixy-go/pkg/command/shell"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/quests"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
	"github.com/josephburnett/nixy-go/pkg/guide"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

// ShellInfo provides context about the current shell for hints and display.
type ShellInfo interface {
	Hostname() string
	CurrentDirectory() []string
	CurrentCommand() string
}

// Mode is the session's current phase.
type Mode int

const (
	ModeLogin   Mode = iota // user is choosing their username
	ModePlaying             // game is live; shell/guide/game are populated
)

// Session holds session-wide state. In ModeLogin, only Username is being
// built; Game/Guide/Shell are nil. In ModePlaying they are all set.
//
// Logging out (exit on the laptop) returns to ModeLogin. Each username
// keeps its own game, so logging back in resumes where that player left
// off, like logging back into a machine that kept running.
type Session struct {
	Mode     Mode
	Username string
	Game     *game.Game
	Guide    *guide.G
	Shell    ShellInfo

	games    map[string]*game.Game
	lastUser string    // who logged in last, to tell a resume from a switch
	loginSh  process.P // the laptop login shell; Guide wraps it
}

const (
	loginMaxLen = 8
	loginPrompt = "login: "
)

// New creates a new session in ModeLogin. The game is not initialized
// until login completes — we don't want quest dialog firing before the
// player has a name.
func New() (*Session, error) {
	return &Session{Mode: ModeLogin}, nil
}

// InitTerminal sets up the terminal for whatever mode the session is in.
func (s *Session) InitTerminal(t *terminal.T) {
	switch s.Mode {
	case ModeLogin:
		s.updateLoginTerminal(t)
	case ModePlaying:
		dialog := s.Game.Manager.Dialog.Drain()
		if len(dialog) > 0 {
			t.SetDialog(dialog)
		}
		s.updatePlayingTerminal(t)
	}
}

// HandleKeystroke routes the keystroke based on session mode. The session
// never ends on its own: logging out returns to the login prompt, and
// quitting is up to the front end (double Ctrl+C in the CLI).
func (s *Session) HandleKeystroke(datum process.Datum, t *terminal.T) {
	if datum == process.TermClear {
		// Ctrl+L clears the screen and keeps what's typed, like readline.
		// It's the terminal's job, so the shell never sees it.
		t.State.Lines = nil
		return
	}
	if s.Mode == ModeLogin {
		s.handleLoginKeystroke(datum, t)
		return
	}
	s.handlePlayingKeystroke(datum, t)
}

// --- login mode ---------------------------------------------------------

// handleLoginKeystroke validates the keystroke against the login rules
// (lowercase a-z, length 1-8, Backspace when non-empty, Enter when valid)
// and bootstraps the game on a successful Enter.
func (s *Session) handleLoginKeystroke(datum process.Datum, t *terminal.T) {
	t.Notify("")
	switch d := datum.(type) {
	case process.Chars:
		if len(d) == 1 && isLoginChar(rune(d[0])) && len(t.State.Line) < loginMaxLen {
			t.State.Line += string(d)
		}
	case process.TermCode:
		switch d {
		case process.TermBackspace:
			if len(t.State.Line) > 0 {
				t.State.Line = t.State.Line[:len(t.State.Line)-1]
			}
		case process.TermEnter:
			if isValidUsername(t.State.Line) {
				name := t.State.Line
				// Echo the login line into the scrollback like a real
				// terminal does.
				t.Write(process.Data{process.TermEnter})
				if err := s.login(name, t); err != nil {
					t.Notify("login error: " + err.Error())
					s.updateLoginTerminal(t)
				}
				return
			}
		}
	}
	s.updateLoginTerminal(t)
}

func (s *Session) updateLoginTerminal(t *terminal.T) {
	t.SetPrompt(terminal.PromptInfo{Raw: loginPrompt})
	t.SetPromptTarget("") // no specific target during login
	t.SetKeyboard(loginValidKeys(t.State.Line), loginHintKey(t.State.Line))
	t.SetThought(loginThought(t.State.Line))
}

func isLoginChar(r rune) bool { return r >= 'a' && r <= 'z' }

func isValidUsername(s string) bool {
	if len(s) < 1 || len(s) > loginMaxLen {
		return false
	}
	for _, r := range s {
		if !isLoginChar(r) {
			return false
		}
	}
	return true
}

// loginValidKeys returns the keys the player may press right now during
// login, given the current buffer.
func loginValidKeys(buf string) []process.Datum {
	var valid []process.Datum
	if len(buf) < loginMaxLen {
		for r := 'a'; r <= 'z'; r++ {
			valid = append(valid, process.Chars(string(r)))
		}
	}
	if len(buf) > 0 {
		valid = append(valid, process.TermBackspace)
	}
	if isValidUsername(buf) {
		valid = append(valid, process.TermEnter)
	}
	return valid
}

// loginHintKey suggests the next keystroke. Once a valid username exists
// we hint Enter; if the buffer is full but invalid (shouldn't happen),
// we'd hint Backspace. With an empty buffer we have no specific hint.
func loginHintKey(buf string) process.Datum {
	switch {
	case isValidUsername(buf):
		return process.TermEnter
	case len(buf) >= loginMaxLen:
		return process.TermBackspace
	default:
		return nil
	}
}

func loginThought(buf string) string {
	if isValidUsername(buf) {
		return "I can press enter to log in"
	}
	if len(buf) == 0 {
		return "I need to pick a username (lowercase letters, up to 8)"
	}
	return "I can keep typing or press enter when ready"
}

// login starts a login shell on the laptop for name, resuming that
// player's game if they have one, and switches to ModePlaying.
//
// A different username gets its own game rather than a new home in the
// current one: quest progress and Nixy's dialog belong to one player, and
// sharing them would hand the second player the first one's achievements.
func (s *Session) login(name string, t *terminal.T) error {
	if s.games == nil {
		s.games = map[string]*game.Game{}
	}
	g, resumed := s.games[name]
	if !resumed {
		var err error
		g, err = newGame(name)
		if err != nil {
			return err
		}
		s.games[name] = g
	}
	if name != s.lastUser {
		// Someone else's conversation with Nixy shouldn't linger on screen.
		t.State.Dialog = nil
	}
	s.lastUser = name
	shellpkg.DefaultNxHandler = g
	s.Username = name
	s.Game = g
	if err := s.startLoginShell(); err != nil {
		return err
	}
	s.Mode = ModePlaying

	// Drain pending dialog (e.g. the first quest's greeting).
	dialog := g.Manager.Dialog.Drain()
	if len(dialog) > 0 {
		t.SetDialog(dialog)
	}
	s.updatePlayingTerminal(t)
	return nil
}

// startLoginShell launches a fresh shell on the laptop for the current
// user, replacing (and killing) any previous one.
func (s *Session) startLoginShell() error {
	if s.loginSh != nil {
		s.loginSh.Kill()
	}
	proc, err := s.Game.Sim.Launch("laptop", s.Username, "shell", nil, []string{})
	if err != nil {
		return err
	}
	s.loginSh = proc
	s.Guide = guide.New(proc)
	s.Shell = proc.(ShellInfo)
	return nil
}

// logout ends the login shell and returns to the login prompt. The game
// stays in s.games for when this user logs back in.
func (s *Session) logout(t *terminal.T) {
	t.Write(process.Data{process.Chars("logout\n")})
	if s.loginSh != nil {
		s.loginSh.Kill()
	}
	s.loginSh = nil
	s.Mode = ModeLogin
	s.Username = ""
	s.Game = nil
	s.Guide = nil
	s.Shell = nil
	t.State.Line = ""
	s.updateLoginTerminal(t)
}

// newGame builds a fresh world and quest line for username.
func newGame(username string) (*game.Game, error) {
	allQuests := []game.Quest{
		&quests.Connect{},
		&quests.Orientation{},
		&quests.Inspection{},
		&quests.Modification{},
		&quests.Composition{},
		&quests.Permissions{},
	}
	machines := []game.MachineEntry{
		{Hostname: "laptop", Filesystem: worlds.Laptop},
		{Hostname: "nixy", Filesystem: worlds.Nixy},
		{Hostname: "server", Filesystem: worlds.Server, UnlockedBy: "server-unlocked"},
	}
	return game.NewGame(allQuests, machines, username)
}

// --- playing mode -------------------------------------------------------

func (s *Session) handlePlayingKeystroke(datum process.Datum, t *terminal.T) {
	t.SetPrompt(s.promptFor(s.Shell.Hostname(), s.Shell.CurrentDirectory()))

	// Capture command + host before Enter dispatches — ssh nixy must be
	// recorded against laptop, not the new nixy shell it spawns.
	var cmdLine, cmdHost string
	var cmdCwd []string
	if datum == process.TermEnter {
		cmdLine = strings.TrimSpace(t.State.Line)
		cmdHost = s.Shell.Hostname()
		cmdCwd = s.Shell.CurrentDirectory()
	}

	// Invalid keystrokes return an error from the guide; we silently swallow
	// it so the user gets no negative feedback.
	_, _ = s.Guide.Stdin(process.Data{datum})
	t.Notify("")

	loggedOut := false
	for range 50 {
		out, eof, _ := s.Guide.Stdout()
		if len(out) > 0 {
			t.Write(out)
		}
		if eof {
			loggedOut = true
			break
		}
		if len(out) == 0 {
			break
		}
	}

	for range 10 {
		errOut, _, _ := s.Guide.Stderr()
		if len(errOut) > 0 {
			t.Write(errOut)
		} else {
			break
		}
	}

	// nx panic disconnects the player back to the laptop: every nested
	// ssh shell goes, and so does the laptop shell (which the panic may
	// have exited), replaced by a fresh one. It never logs the player out.
	if s.Game.TakePanic() {
		if err := s.startLoginShell(); err != nil {
			t.Notify("nx panic: " + err.Error())
		}
		loggedOut = false
	}

	if _, ok := datum.(process.TermCode); ok && datum == process.TermEnter {
		if cmdLine != "" {
			// Same host? use post-execution cwd (so cd counts as a visit).
			// Different host? keep pre-command cwd (ssh/exit attribution).
			cwd := cmdCwd
			if s.Shell.Hostname() == cmdHost {
				cwd = s.Shell.CurrentDirectory()
			}
			s.Game.Manager.Tracker.Record(cmdHost, cwd, cmdLine)
		}
		s.Game.AfterCommand()
		dialog := s.Game.Manager.Dialog.Drain()
		if len(dialog) > 0 {
			t.SetDialog(dialog)
		}
	}

	if loggedOut {
		s.logout(t)
		return
	}
	s.updatePlayingTerminal(t)
}

func (s *Session) updatePlayingTerminal(t *terminal.T) {
	t.SetPrompt(s.promptFor(s.Shell.Hostname(), s.Shell.CurrentDirectory()))
	t.SetPromptTarget(s.Game.GetPlannedCommand(s.Shell.Hostname(), s.Shell.CurrentDirectory()))
	valid := s.Guide.Next()
	hint := s.Game.GetHint(s.Shell.Hostname(), s.Shell.CurrentDirectory(), s.Shell.CurrentCommand())
	t.SetKeyboard(valid, hint)
	t.SetThought(s.Game.GetThought(s.Shell.Hostname(), s.Shell.CurrentDirectory()))
}

func (s *Session) promptFor(hostname string, cwd []string) terminal.PromptInfo {
	path := "/"
	if len(cwd) > 0 {
		path = "/" + strings.Join(cwd, "/")
	}
	return terminal.PromptInfo{User: s.Username, Host: hostname, Path: path}
}
