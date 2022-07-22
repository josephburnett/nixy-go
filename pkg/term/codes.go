package term

import (
	"strings"
)

const (
	Delete = "[DELETE]"
	Enter  = "[ENTER]"
	Clear  = "[CLEAR]"
)

type Term struct {
	x, y  int
	lines []string
}

func NewTerm() *Term {
	return &Term{
		x:     40,
		y:     20,
		lines: []string{},
	}
}

func (t *Term) Write(in string) error {

	// Replace nice term codes with single-char internal codes to
	// make parsing easier.
	internalDelete := 127 // ASCII DEL
	internalEnter := 10   // ASCII Line Feed
	internalClear := 12   // ASCII Form Feed ¯\_(ツ)_/¯
	raw := strings.ReplaceAll(in, Delete, string(internalDelete))
	raw = strings.ReplaceAll(raw, Enter, string(internalEnter))
	raw = strings.ReplaceAll(raw, Clear, string(internalClear))

	line := ""
	for _, c := range raw {
		switch c {
		case rune(internalDelete):
			// Drop last char on line.
			if len(line) > 0 {
				line = line[0 : len(line)-2]
			}
		case rune(internalEnter):
			// Carriage Return and Line Feed
			t.lines = append(t.lines, line)
			line = ""
		case rune(internalClear):
			// Clear the screen
			t.lines = []string{}
			line = ""
		default:
			line += string(c)
		}
	}
	return nil
}
