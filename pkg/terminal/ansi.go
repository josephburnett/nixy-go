package terminal

import (
	"strings"
	"unicode/utf8"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// ANSI color codes
const (
	colorReset = "\033[0m"
	colorDim   = "\033[2m"    // dim grey
	colorWhite = "\033[1;37m" // bold white
	colorGreen = "\033[1;32m" // bold green
	colorDialog = "\033[33m"  // yellow
)

// ANSIRenderer renders frames using ANSI escape codes and box-drawing characters.
type ANSIRenderer struct{}

func NewANSI() *ANSIRenderer {
	return &ANSIRenderer{}
}

func (a *ANSIRenderer) Render(f Frame) string {
	var sb strings.Builder
	border := strings.Repeat("─", f.Width)

	sb.WriteString("┌" + border + "┐\n")

	// Display lines (padded to fill viewport)
	for i := 0; i < f.Height; i++ {
		if i < len(f.DisplayLines) {
			line := f.DisplayLines[i]
			runeLen := utf8.RuneCountInString(line)
			if runeLen > f.Width {
				line = string([]rune(line)[:f.Width])
				runeLen = f.Width
			}
			padding := f.Width - runeLen
			sb.WriteString("│" + line + strings.Repeat(" ", padding) + "│\n")
		} else {
			sb.WriteString("│" + strings.Repeat(" ", f.Width) + "│\n")
		}
	}

	// Prompt line
	prompt := f.Prompt
	runeLen := utf8.RuneCountInString(prompt)
	if runeLen > f.Width {
		prompt = string([]rune(prompt)[:f.Width])
		runeLen = f.Width
	}
	padding := f.Width - runeLen
	sb.WriteString("│" + prompt + strings.Repeat(" ", padding) + "│\n")

	sb.WriteString("└" + border + "┘\n")

	// Dialog (yellow)
	for _, line := range f.Dialog {
		sb.WriteString(colorDialog + line + colorReset + "\n")
	}

	// Hint (dim)
	if f.Hint != "" {
		sb.WriteString(colorDim + f.Hint + colorReset + "\n")
	}

	// Keyboard
	sb.WriteString("\n")
	sb.WriteString(renderANSIKeyboard(f.ValidKeys, f.HintKey))

	return sb.String()
}

func renderANSIKeyboard(valid []process.Datum, hint process.Datum) string {
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
			sb.WriteString(colorKeyANSI(key, datum, validSet, hint))
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
		sb.WriteString(colorKeyANSI(label, sk.datum, validSet, hint))
	}
	sb.WriteString("\n")

	return sb.String()
}

func colorKeyANSI(label string, datum process.Datum, validSet datumSet, hint process.Datum) string {
	if hint != nil && datumEqual(datum, hint) {
		return colorGreen + label + colorReset
	}
	if validSet.contains(datum) {
		return colorWhite + label + colorReset
	}
	return colorDim + label + colorReset
}
