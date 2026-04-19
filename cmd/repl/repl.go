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

	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/session"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

type model struct {
	sess     *session.Session
	terminal *terminal.T
	quitting bool
}

func initialModel() (model, error) {
	sess, err := session.New()
	if err != nil {
		return model{}, err
	}

	t := terminal.New(terminal.NewANSI())
	sess.InitTerminal(t)

	return model{
		sess:     sess,
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

		if m.sess.HandleKeystroke(datum, m.terminal) {
			m.quitting = true
			return m, tea.Quit
		}
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
