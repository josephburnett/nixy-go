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

// Session holds the initialized game, guide, and shell.
type Session struct {
	Game  *game.Game
	Guide *guide.G
	Shell ShellInfo
}

// New creates a new game session with all quests and machines initialized.
// Callers must import command packages (via blank imports) before calling this.
func New() (*Session, error) {
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

	g, err := game.NewGame(allQuests, machines)
	if err != nil {
		return nil, err
	}

	shellpkg.DefaultNxHandler = g

	proc, err := g.Sim.Launch("laptop", "user", "shell", nil, []string{})
	if err != nil {
		return nil, err
	}

	gd := guide.New(proc)

	return &Session{
		Game:  g,
		Guide: gd,
		Shell: proc.(ShellInfo),
	}, nil
}

// InitTerminal drains initial quest dialog and sets keyboard state.
func (s *Session) InitTerminal(t *terminal.T) {
	dialog := s.Game.Manager.Dialog.Drain()
	if len(dialog) > 0 {
		t.SetDialog(dialog)
	}
	s.updateTerminal(t)
}

// HandleKeystroke processes a single input datum through the guide, drains
// output, checks quest state, and updates the terminal. Returns true if EOF.
func (s *Session) HandleKeystroke(datum process.Datum, t *terminal.T) bool {
	t.State.Prompt = "user@" + s.Shell.Hostname()

	// Capture command + host before Enter dispatches — ssh nixy must be
	// recorded against laptop, not the new nixy shell it spawns.
	var cmdLine, cmdHost string
	var cmdCwd []string
	if datum == process.TermEnter {
		cmdLine = strings.TrimSpace(t.State.Line)
		cmdHost = s.Shell.Hostname()
		cmdCwd = s.Shell.CurrentDirectory()
	}

	_, err := s.Guide.Stdin(process.Data{datum})
	t.Hint(err)

	// Drain stdout
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

	// Drain stderr
	for range 10 {
		errOut, _, _ := s.Guide.Stderr()
		if len(errOut) > 0 {
			t.Write(errOut)
		} else {
			break
		}
	}

	// After Enter, record the command, check quest state and dialog
	if _, ok := datum.(process.TermCode); ok && datum == process.TermEnter {
		if cmdLine != "" {
			s.Game.Manager.Tracker.Record(cmdHost, cmdCwd, cmdLine)
		}
		s.Game.AfterCommand()
		dialog := s.Game.Manager.Dialog.Drain()
		if len(dialog) > 0 {
			t.SetDialog(dialog)
		}
	}

	s.updateTerminal(t)
	return false
}

func (s *Session) updateTerminal(t *terminal.T) {
	t.State.Prompt = "user@" + s.Shell.Hostname()
	valid := s.Guide.Next()
	hint := s.Game.GetHint(s.Shell.Hostname(), s.Shell.CurrentDirectory(), s.Shell.CurrentCommand())
	t.SetKeyboard(valid, hint)
}
