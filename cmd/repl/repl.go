package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

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

	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/guide"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/session"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

type model struct {
	game     *game.Game
	guide    *guide.G
	shell    session.ShellInfo
	terminal *terminal.T
	quitting bool
}

func initialModel() (model, error) {
	sess, err := session.New()
	if err != nil {
		return model{}, err
	}

	t := terminal.New(terminal.NewANSI())

	return model{
		game:     sess.Game,
		guide:    sess.Guide,
		shell:    sess.Shell,
		terminal: t,
	}, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminal.Resize(msg.Width, msg.Height)

	case tea.KeyMsg:
		var datum process.Datum

		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			datum = process.TermEnter
		case tea.KeyBackspace:
			datum = process.TermBackspace
		case tea.KeyCtrlL:
			datum = process.TermClear
		case tea.KeyRunes:
			if len(msg.Runes) == 1 {
				datum = process.Chars(string(msg.Runes[0]))
			}
		case tea.KeySpace:
			datum = process.Chars(" ")
		}

		if datum == nil {
			return m, nil
		}

		// Write to shell through guide
		_, err := m.guide.Stdin(process.Data{datum})
		m.terminal.Hint(err)

		// Drain stdout
		for i := 0; i < 50; i++ {
			out, eof, _ := m.guide.Stdout()
			if eof {
				m.quitting = true
				return m, tea.Quit
			}
			if len(out) > 0 {
				m.terminal.Write(out)
			} else {
				break
			}
		}

		// Drain stderr
		for i := 0; i < 10; i++ {
			errOut, _, _ := m.guide.Stderr()
			if len(errOut) > 0 {
				m.terminal.Write(errOut)
			} else {
				break
			}
		}

		// After Enter, check quest state and dialog
		if _, ok := datum.(process.TermCode); ok && datum == process.TermEnter {
			m.game.AfterCommand()
			dialog := m.game.Manager.Dialog.Drain()
			if len(dialog) > 0 {
				m.terminal.SetDialog(dialog)
			}
		}

		// Update keyboard display
		valid := m.guide.Next()
		hint := m.game.GetHint(m.shell.Hostname(), m.shell.CurrentDirectory(), m.shell.CurrentCommand())
		m.terminal.SetKeyboard(valid, hint)
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}
	return m.terminal.Render()
}

func main() {
	m, err := initialModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
