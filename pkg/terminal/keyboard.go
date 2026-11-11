package terminal

import (
	"unicode/utf8"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// keyboardRows defines the qwerty layout. Each key is its KeyName.
// Keys that don't fit the special row (it must fit a phone) go at the
// end of the shorter letter rows: ~, and ^C, which interrupts a
// running command.
var keyboardRows = [][]string{
	{"q", "w", "e", "r", "t", "y", "u", "i", "o", "p"},
	{"a", "s", "d", "f", "g", "h", "j", "k", "l"},
	{"z", "x", "c", "v", "b", "n", "m", "~", "^C"},
}

// specialKeys are shown on the bottom row. The row is wider than the
// letter rows, so narrow screens get shorter labels (see specialRowLabels).
var specialKeys = []struct {
	label string
	short string
	datum process.Datum
}{
	{"space", "spc", process.Chars(" ")},
	{".", ".", process.Chars(".")},
	{"/", "/", process.Chars("/")},
	{"|", "|", process.Chars("|")},
	{"-", "-", process.Chars("-")},
	{"enter", "ent", process.TermEnter},
	{"bksp", "bs", process.TermBackspace},
}

// specialRowLabels returns the widest special-key labels that fit in
// width columns (including the row's one-column indent): "[space]", then
// "[spc]", then bare "spc" at the smallest sizes.
func specialRowLabels(width int) []string {
	for _, tier := range []struct {
		short    bool
		brackets bool
	}{{false, true}, {true, true}, {true, false}} {
		labels := make([]string, len(specialKeys))
		n := 1 + len(specialKeys) - 1 // indent + separators
		for i, sk := range specialKeys {
			l := sk.label
			if tier.short {
				l = sk.short
			}
			if tier.brackets {
				l = "[" + l + "]"
			}
			labels[i] = l
			n += utf8.RuneCountInString(l)
		}
		if n <= width || !tier.brackets {
			return labels
		}
	}
	return nil // unreachable: the last tier always returns
}

// datumSet is a simple set for checking datum membership.
type datumSet struct {
	chars     map[string]bool
	termCodes map[process.TermCode]bool
	signals   map[process.Signal]bool
}

func buildDatumSet(datums []process.Datum) datumSet {
	s := datumSet{
		chars:     map[string]bool{},
		termCodes: map[process.TermCode]bool{},
		signals:   map[process.Signal]bool{},
	}
	for _, d := range datums {
		switch d := d.(type) {
		case process.Chars:
			s.chars[string(d)] = true
		case process.TermCode:
			s.termCodes[d] = true
		case process.Signal:
			s.signals[d] = true
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
	case process.Signal:
		return s.signals[d]
	}
	return false
}

// KeyName is the name a front end uses for a keystroke: "Enter",
// "Backspace", "^C" (Ctrl+C), or the character itself. Apart from ^C it
// matches DOM KeyboardEvent.key, so the web build sends typed and
// tapped keys the same way (it maps Ctrl+C to "^C" itself).
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
	case process.Signal:
		if d == process.SigInt {
			return "^C"
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
	case "^C":
		return process.SigInt
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
	case process.Signal:
		if b, ok := b.(process.Signal); ok {
			return a == b
		}
	}
	return false
}
