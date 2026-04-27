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
type Session struct {
	Mode     Mode
	Username string
	Game     *game.Game
	Guide    *guide.G
	Shell    ShellInfo
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

// HandleKeystroke routes the keystroke based on session mode.
func (s *Session) HandleKeystroke(datum process.Datum, t *terminal.T) bool {
	if s.Mode == ModeLogin {
		return s.handleLoginKeystroke(datum, t)
	}
	return s.handlePlayingKeystroke(datum, t)
}

// --- login mode ---------------------------------------------------------

// handleLoginKeystroke validates the keystroke against the login rules
// (lowercase a-z, length 1-8, Backspace when non-empty, Enter when valid)
// and bootstraps the game on a successful Enter.
func (s *Session) handleLoginKeystroke(datum process.Datum, t *terminal.T) bool {
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
				s.Username = t.State.Line
				t.State.Line = ""
				if err := s.bootstrapGame(t); err != nil {
					t.Notify("login error: " + err.Error())
					return false
				}
				return false
			}
		}
	}
	s.updateLoginTerminal(t)
	return false
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

// bootstrapGame initializes the game world after a successful login and
// switches the session to ModePlaying.
func (s *Session) bootstrapGame(t *terminal.T) error {
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
	g, err := game.NewGame(allQuests, machines, s.Username)
	if err != nil {
		return err
	}
	shellpkg.DefaultNxHandler = g
	proc, err := g.Sim.Launch("laptop", s.Username, "shell", nil, []string{})
	if err != nil {
		return err
	}
	s.Game = g
	s.Guide = guide.New(proc)
	s.Shell = proc.(ShellInfo)
	s.Mode = ModePlaying

	// Drain initial dialog (e.g. first quest activation greeting).
	dialog := g.Manager.Dialog.Drain()
	if len(dialog) > 0 {
		t.SetDialog(dialog)
	}
	s.updatePlayingTerminal(t)
	return nil
}

// --- playing mode -------------------------------------------------------

func (s *Session) handlePlayingKeystroke(datum process.Datum, t *terminal.T) bool {
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

	for range 50 {
		out, eof, _ := s.Guide.Stdout()
		if eof {
			return true
		}
		if len(out) > 0 {
			t.Write(out)
		} else {
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

	s.updatePlayingTerminal(t)
	return false
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
