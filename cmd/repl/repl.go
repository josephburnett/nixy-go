package main

import (
	"fmt"
	"os"
	"time"

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

	"github.com/josephburnett/nixy-go/pkg/debug"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/session"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

type model struct {
	sess      *session.Session
	terminal  *terminal.T
	recorder  *debug.Recorder
	quitting  bool
	lastCtrlC time.Time
}

const (
	ctrlCWindow = 2 * time.Second
	debugRing   = 10
)

// noticeExpireMsg is dispatched by a tea.Tick after a Ctrl+C so the
// "Press Ctrl+C again" notice clears once the window has elapsed.
type noticeExpireMsg struct{ at time.Time }

func initialModel() (model, error) {
	sess, err := session.New()
	if err != nil {
		return model{}, err
	}

	t := terminal.New(terminal.NewANSI())
	sess.InitTerminal(t)

	r := debug.NewRecorder(debugRing)
	r.Push(sess, t, nil)

	return model{
		sess:     sess,
		terminal: t,
		recorder: r,
	}, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminal.Resize(msg.Width, msg.Height)

	case noticeExpireMsg:
		// Clear only if no later Ctrl+C reset the window after this tick
		// was scheduled.
		if !msg.at.Before(m.lastCtrlC.Add(ctrlCWindow)) {
			m.terminal.Notify("")
		}
		return m, nil

	case tea.KeyMsg:
		var datum process.Datum

		switch msg.Type {
		case tea.KeyCtrlC:
			if time.Since(m.lastCtrlC) < ctrlCWindow {
				m.quitting = true
				return m, tea.Quit
			}
			m.lastCtrlC = time.Now()
			m.terminal.Notify("Press Ctrl+C again to exit")
			return m, tea.Tick(ctrlCWindow, func(t time.Time) tea.Msg {
				return noticeExpireMsg{at: t}
			})
		case tea.KeyCtrlBackslash:
			path, err := m.dumpSnapshot()
			if err != nil {
				m.terminal.Notify("debug dump failed: " + err.Error())
			} else {
				m.terminal.Notify("debug dump → " + path)
			}
			return m, nil
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
		m.recorder.Push(m.sess, m.terminal, datum)
	}

	return m, nil
}

func (m model) dumpSnapshot() (string, error) {
	path := fmt.Sprintf("nixy-debug-%s.txt", time.Now().Format("20060102-150405"))
	if err := os.WriteFile(path, []byte(m.recorder.Dump()), 0o644); err != nil {
		return "", err
	}
	return path, nil
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
