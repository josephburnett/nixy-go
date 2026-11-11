package session

import (
	"slices"
	"strings"

	shellpkg "github.com/josephburnett/nixy-go/pkg/command/shell"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/quests"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
	"github.com/josephburnett/nixy-go/pkg/guide"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

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
	Shell    shellpkg.Info

	games    map[string]*game.Game
	accounts []string  // existing users, which the player can't log in as
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
	return &Session{Mode: ModeLogin, accounts: game.Accounts(worldMachines())}, nil
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
	if !slices.Contains(s.loginValidKeys(t.State.Line), datum) {
		return // like the shell: a key not offered does nothing
	}
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
			if s.canLogin(t.State.Line) {
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
	t.SetKeyboard(s.loginValidKeys(t.State.Line), s.loginHintKey(t.State.Line))
	t.SetThought(s.loginThought(t.State.Line))
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

// canLogin reports whether name is a username the player may take: well
// formed and not an existing account. Real login would ask for that
// account's password, and the game has none, so root and nixy are taken.
func (s *Session) canLogin(name string) bool {
	return isValidUsername(name) && !slices.Contains(s.accounts, name)
}

// canReach reports whether typing more letters after buf can still
// reach a name the player may take.
func (s *Session) canReach(buf string) bool {
	if s.canLogin(buf) {
		return true
	}
	if len(buf) >= loginMaxLen {
		return false
	}
	for r := 'a'; r <= 'z'; r++ {
		if s.canReach(buf + string(r)) {
			return true
		}
	}
	return false
}

// loginValidKeys returns the keys the player may press right now during
// login, given the current buffer: letters that can still lead to a
// name the player may take, and Enter on one.
func (s *Session) loginValidKeys(buf string) []process.Datum {
	var valid []process.Datum
	if len(buf) < loginMaxLen {
		for r := 'a'; r <= 'z'; r++ {
			if s.canReach(buf + string(r)) {
				valid = append(valid, process.Chars(string(r)))
			}
		}
	}
	if len(buf) > 0 {
		valid = append(valid, process.TermBackspace)
	}
	if s.canLogin(buf) {
		valid = append(valid, process.TermEnter)
	}
	return valid
}

// loginHintKey suggests the next keystroke: Enter once the name can be
// taken, Backspace when nothing else is left. With an empty buffer we
// have no specific hint.
func (s *Session) loginHintKey(buf string) process.Datum {
	switch {
	case s.canLogin(buf):
		return process.TermEnter
	case !s.canReach(buf):
		return process.TermBackspace
	default:
		return nil
	}
}

func (s *Session) loginThought(buf string) string {
	if s.canLogin(buf) {
		return "I can press enter to log in"
	}
	if len(buf) == 0 {
		return "I need to pick a username (lowercase letters, up to 8)"
	}
	if slices.Contains(s.accounts, buf) {
		return buf + " is someone else's account. I need a name of my own"
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
	proc, err := s.Game.Sim.Launch("laptop", s.Username, "shell", []string{"-l"}, nil)
	if err != nil {
		return err
	}
	s.loginSh = proc
	s.Guide = guide.New(proc)
	s.Shell = proc.(shellpkg.Info)
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

// worldMachines are the game's machines, each built fresh by its
// Filesystem.
func worldMachines() []game.MachineEntry {
	return []game.MachineEntry{
		{Hostname: "laptop", Filesystem: worlds.Laptop},
		{Hostname: "nixy", Filesystem: worlds.Nixy},
		{Hostname: "server", Filesystem: worlds.Server, UnlockedBy: "server-unlocked"},
	}
}

// newGame builds a fresh world and quest line for username.
func newGame(username string) (*game.Game, error) {
	return game.NewGame(quests.All(), worldMachines(), username)
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

	// A key the keyboard didn't offer has no effect at all: nothing is
	// recorded or run, and the screen stays as it was. There is no
	// negative feedback either; the key just does nothing.
	typedIn := s.Shell.Active()
	if err := s.Guide.Stdin(process.Data{datum}); err != nil {
		return
	}
	t.Notify("")

	// Show what the shell writes, stdout and stderr in the order
	// written, as a terminal does. The cap keeps a command that never
	// stops writing from hanging the keystroke.
	loggedOut := false
	for range 50 {
		outs := s.Guide.Read()
		for _, o := range outs {
			t.Write(o.Data)
		}
		if _, exited := s.Guide.Exit(); exited {
			loggedOut = true
			break
		}
		// A quiet read means nothing more to show, unless a command is
		// still running: a pipeline can work for a while before it
		// prints (`yes | head | wc -l`).
		if _, done := s.Shell.Active().LastStatus(); len(outs) == 0 && done {
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
			status, done := typedIn.LastStatus()
			s.Game.Manager.Tracker.Record(game.CommandRecord{
				Hostname: cmdHost, User: s.Username, Dir: cmdCwd, Cwd: cwd, Command: cmdLine,
				Status: status, Running: !done,
			})
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
	if !slices.Contains(valid, hint) {
		// The planner plans command lines; while one runs (only ^C
		// is offered), or if its plan has gone stale, a hint on a key
		// that does nothing is worse than none.
		hint = nil
	}
	t.SetKeyboard(valid, hint)
	t.SetThought(s.Game.GetThought(s.Shell.Hostname(), s.Shell.CurrentDirectory()))
}

func (s *Session) promptFor(hostname string, cwd []string) terminal.PromptInfo {
	return terminal.PromptInfo{User: s.Username, Host: hostname, Path: promptPath(cwd, file.Home(s.Username))}
}

// promptPath is the cwd as bash's \w shows it: home and paths below it
// start with `~`, everything else is absolute.
func promptPath(cwd, home []string) string {
	if len(cwd) >= len(home) && slices.Equal(cwd[:len(home)], home) {
		if rest := cwd[len(home):]; len(rest) > 0 {
			return "~/" + strings.Join(rest, "/")
		}
		return "~"
	}
	return "/" + strings.Join(cwd, "/")
}
