package session

import (
	shellpkg "github.com/josephburnett/nixy-go/pkg/command/shell"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/quests"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
	"github.com/josephburnett/nixy-go/pkg/guide"
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
