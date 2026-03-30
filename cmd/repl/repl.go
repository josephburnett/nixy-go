package main

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	_ "github.com/josephburnett/nixy-go/pkg/command/apt"
	_ "github.com/josephburnett/nixy-go/pkg/command/cat"
	_ "github.com/josephburnett/nixy-go/pkg/command/grep"
	_ "github.com/josephburnett/nixy-go/pkg/command/ls"
	_ "github.com/josephburnett/nixy-go/pkg/command/mv"
	_ "github.com/josephburnett/nixy-go/pkg/command/pwd"
	_ "github.com/josephburnett/nixy-go/pkg/command/rm"
	_ "github.com/josephburnett/nixy-go/pkg/command/shell"
	_ "github.com/josephburnett/nixy-go/pkg/command/ssh"
	_ "github.com/josephburnett/nixy-go/pkg/command/sudo"
	_ "github.com/josephburnett/nixy-go/pkg/command/touch"

	shellpkg "github.com/josephburnett/nixy-go/pkg/command/shell"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/quests"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
	"github.com/josephburnett/nixy-go/pkg/guide"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

func main() {
	g, t, gd, err := launch()
	if err != nil {
		panic(err)
	}
	in := make([]byte, 1)
	for {
		// Read shell stdout
		out, eof, err := gd.Stdout()
		if err != nil {
			panic(err)
		}
		if eof {
			return
		}
		// Read shell stderr
		errOut, _, _ := gd.Stderr()

		// Write to terminal
		_ = t.Write(out)
		_ = t.Write(errOut)

		// Show dialog if any
		dialog := g.Manager.Dialog.Drain()
		if len(dialog) > 0 {
			t.SetDialog(dialog)
		}

		// Render
		view := t.Render()
		fmt.Print(view)

		// Read keyboard
		count, err := os.Stdin.Read(in)
		if err == io.EOF {
			return
		}
		if err != nil {
			panic(err)
		}

		// Write to shell
		if count > 0 {
			r, _ := utf8.DecodeRune(in)
			s := string(r)
			data := process.CharsData(s)
			if s == ">" {
				data[0] = process.TermEnter
			}
			if s == "<" {
				data[0] = process.TermBackspace
			}
			if s == "^" {
				data[0] = process.TermClear
			}
			if s == "\n" {
				continue
			}
			_, err := gd.Stdin(data)
			t.Hint(err)

			// After Enter, check quest state
			if data[0] == process.TermEnter {
				g.AfterCommand()
			}
		}
	}
}

func launch() (*game.Game, *terminal.T, *guide.G, error) {
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
		return nil, nil, nil, err
	}

	// Wire nx handler
	shellpkg.DefaultNxHandler = g

	// Launch shell on laptop
	proc, err := g.Sim.Launch("laptop", "user", "shell", nil, []string{})
	if err != nil {
		return nil, nil, nil, err
	}
	gd := guide.New(proc)
	t := terminal.New()
	return g, t, gd, nil
}
