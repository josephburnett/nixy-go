package terminal

import (
	"strings"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorDim    = "\033[2m"    // dim grey
	colorWhite  = "\033[1;37m" // bold white
	colorGreen  = "\033[1;32m" // bold green
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

// RenderKeyboard renders a visual keyboard with color-coded keys.
// valid = keys from Next(), hint = the planner's recommended key.
func RenderKeyboard(valid []process.Datum, hint process.Datum) string {
	validSet := buildDatumSet(valid)

	var sb strings.Builder

	// Letter rows
	indents := []string{" ", "  ", "   "}
	for row, keys := range keyboardRows {
		sb.WriteString(indents[row])
		for i, key := range keys {
			if i > 0 {
				sb.WriteString(" ")
			}
			datum := process.Chars(key)
			sb.WriteString(colorKey(key, datum, validSet, hint))
		}
		sb.WriteString("\n")
	}

	// Special keys row
	sb.WriteString(" ")
	for i, sk := range specialKeys {
		if i > 0 {
			sb.WriteString(" ")
		}
		label := "[" + sk.label + "]"
		sb.WriteString(colorKey(label, sk.datum, validSet, hint))
	}
	sb.WriteString("\n")

	return sb.String()
}

func colorKey(label string, datum process.Datum, validSet datumSet, hint process.Datum) string {
	if hint != nil && datumEqual(datum, hint) {
		return colorGreen + label + colorReset
	}
	if validSet.contains(datum) {
		return colorWhite + label + colorReset
	}
	return colorDim + label + colorReset
}

// datumSet is a simple set for checking datum membership.
type datumSet struct {
	chars    map[string]bool
	termCodes map[process.TermCode]bool
}

func buildDatumSet(datums []process.Datum) datumSet {
	s := datumSet{
		chars:    map[string]bool{},
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
