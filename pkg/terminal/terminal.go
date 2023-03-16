package terminal

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/process"
)

type T struct {
	x, y  int
	line  string
	lines []string
}

func New() *T {
	return &T{
		x:     40,
		y:     20,
		lines: []string{},
	}
}

func (t *T) Write(in process.Data) error {
	for _, d := range in {
		switch d := d.(type) {
		case process.Chars:
			t.line += string(d)
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
	border := strings.Repeat("=", 55) + "\n"
	var buf [20]string
	i := len(t.lines) - 20
	for j := range buf {
		if i < 0 {
			buf[j] = "|"
			i++
			continue
		}
		buf[j] = "| " + t.lines[i]
		i++
	}
	out := strings.Repeat("\n", 100)
	out += "\n"
	out += " Term codes: Enter = '>', Backspace = '<', Clear = '^'"
	out += "\n\n"
	out += border
	out += strings.Join(buf[:], "\n") + "\n"
	out += "> " + t.line + "\n"
	out += border
	out += "\n"
	return out
}
