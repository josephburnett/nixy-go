package term

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/process"
)

type Term struct {
	x, y  int
	line  string
	lines []string
}

func NewTerm() *Term {
	return &Term{
		x:     40,
		y:     20,
		lines: []string{},
	}
}

func (t *Term) Write(in process.Data) error {
	for _, d := range in {
		switch d := d.(type) {
		case process.Chars:
			t.line += string(d)
		case process.TermCode:
			switch d {
			case process.TermBackspace:
				if len(t.line) > 0 {
					t.line = t.line[:len(t.line)-2]
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

func (t *Term) Render() string {
	out := ""
	lineCount := len(t.lines)
	if lineCount > t.y {
		lineCount = t.y
	}
	for _, line := range t.lines[len(t.lines)-(lineCount+1) : len(t.lines)-1] {
		columnCount := len(line)
		if columnCount > t.x {
			columnCount = t.x
		}
		out += line[:columnCount-1]
		out += "\n"
	}
	return out
}
