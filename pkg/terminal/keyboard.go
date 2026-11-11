package terminal

import (
	"unicode/utf8"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// keyboardRows defines the qwerty layout.
var keyboardRows = [][]string{
	{"q", "w", "e", "r", "t", "y", "u", "i", "o", "p"},
	{"a", "s", "d", "f", "g", "h", "j", "k", "l"},
	{"z", "x", "c", "v", "b", "n", "m"},
}

// specialKeys are shown on the bottom row.
var specialKeys = []struct {
	label string
	datum process.Datum
}{
	{"space", process.Chars(" ")},
	{".", process.Chars(".")},
	{"/", process.Chars("/")},
	{"|", process.Chars("|")},
	{"-", process.Chars("-")},
	{"enter", process.TermEnter},
	{"bksp", process.TermBackspace},
}

// datumSet is a simple set for checking datum membership.
type datumSet struct {
	chars     map[string]bool
	termCodes map[process.TermCode]bool
}

func buildDatumSet(datums []process.Datum) datumSet {
	s := datumSet{
		chars:     map[string]bool{},
		termCodes: map[process.TermCode]bool{},
	}
	for _, d := range datums {
		switch d := d.(type) {
		case process.Chars:
			s.chars[string(d)] = true
		case process.TermCode:
			s.termCodes[d] = true
		}
	}
	return s
}

func (s datumSet) contains(d process.Datum) bool {
	switch d := d.(type) {
	case process.Chars:
		return s.chars[string(d)]
	case process.TermCode:
		return s.termCodes[d]
	}
	return false
}

// KeyName is the name a front end uses for a keystroke: "Enter",
// "Backspace", or the character itself. It matches DOM KeyboardEvent.key,
// so the web build sends typed and tapped keys the same way.
func KeyName(d process.Datum) string {
	switch d := d.(type) {
	case process.Chars:
		return string(d)
	case process.TermCode:
		switch d {
		case process.TermEnter:
			return "Enter"
		case process.TermBackspace:
			return "Backspace"
		}
	}
	return ""
}

// KeyDatum is the inverse of KeyName. It returns nil for names that
// aren't a keystroke the game handles (e.g. "Tab", "ArrowUp").
func KeyDatum(name string) process.Datum {
	switch name {
	case "Enter":
		return process.TermEnter
	case "Backspace":
		return process.TermBackspace
	}
	if utf8.RuneCountInString(name) == 1 {
		return process.Chars(name)
	}
	return nil
}

func datumEqual(a, b process.Datum) bool {
	switch a := a.(type) {
	case process.Chars:
		if b, ok := b.(process.Chars); ok {
			return a == b
		}
	case process.TermCode:
		if b, ok := b.(process.TermCode); ok {
			return a == b
		}
	}
	return false
}
