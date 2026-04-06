package terminal

import (
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
