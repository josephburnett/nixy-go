package terminal

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/process"
)

type T struct {
	x, y   int
	line   string
	lines  []string
	hint   error
	dialog []string
}

func New() *T {
	return &T{
		x:     55,
		y:     20,
		lines: []string{},
	}
}

func (t *T) Write(in process.Data) error {
	for _, d := range in {
		switch d := d.(type) {
		case process.Chars:
			for _, c := range string(d) {
				if c == '\n' {
					t.lines = append(t.lines, t.line)
					t.line = ""
				} else {
					t.line += string(c)
				}
			}
		case process.TermCode:
			switch d {
			case process.TermBackspace:
				if len(t.line) > 0 {
					t.line = t.line[:len(t.line)-1]
				}
			case process.TermClear:
				t.line = ""
				t.lines = []string{}
			case process.TermEnter:
				t.lines = append(t.lines, t.line)
				t.line = ""
			default:
				return fmt.Errorf("unknown term code: %v", d)
			}
		default:
			return fmt.Errorf("unsupported data type: %T", d)
		}
	}
	return nil
}

func (t *T) Render() string {
	var sb strings.Builder
	border := strings.Repeat("─", t.x)

	sb.WriteString("┌" + border + "┐\n")

	// Show last y lines of history
	start := len(t.lines) - t.y
	for i := 0; i < t.y; i++ {
		idx := start + i
		if idx < 0 || idx >= len(t.lines) {
			sb.WriteString("│" + strings.Repeat(" ", t.x) + "│\n")
		} else {
			line := t.lines[idx]
			if len(line) > t.x {
				line = line[:t.x]
			}
			padding := t.x - len(line)
			sb.WriteString("│" + line + strings.Repeat(" ", padding) + "│\n")
		}
	}

	// Current input line
	prompt := "> " + t.line
	if len(prompt) > t.x {
		prompt = prompt[:t.x]
	}
	padding := t.x - len(prompt)
	sb.WriteString("│" + prompt + strings.Repeat(" ", padding) + "│\n")

	sb.WriteString("└" + border + "┘\n")

	// Dialog (yellow)
	for _, line := range t.dialog {
		sb.WriteString("\033[33m" + line + "\033[0m\n")
	}
	t.dialog = nil

	// Hint
	if t.hint != nil {
		sb.WriteString("\033[2m" + t.hint.Error() + "\033[0m\n")
	}

	return sb.String()
}

func (t *T) Hint(err error) {
	t.hint = err
}

func (t *T) SetDialog(lines []string) {
	t.dialog = lines
}
