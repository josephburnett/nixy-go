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
)

// dialogColorsANSI cycles for each new dialog batch so the user can see when
// a fresh message has arrived.
var dialogColorsANSI = []string{
	"\033[31m", // red
	"\033[33m", // yellow
	"\033[32m", // green
	"\033[34m", // blue
	"\033[35m", // purple
}

// ANSIRenderer renders frames using ANSI escape codes and box-drawing characters.
type ANSIRenderer struct{}

func NewANSI() *ANSIRenderer {
	return &ANSIRenderer{}
}

func (a *ANSIRenderer) Render(f Frame) string {
	var sb strings.Builder
	border := strings.Repeat("─", f.Width)

	// Dialog area — padded to fixed height so terminal stays put
	blankLines := f.DialogSpace - len(f.Dialog)
	for i := 0; i < blankLines; i++ {
		sb.WriteString("\n")
	}
	for _, line := range f.Dialog {
		sb.WriteString(renderDialogLineANSI(line) + "\n")
	}

	// Hint line — always occupies 1 line (blank if no hint)
	if f.Hint != "" {
		sb.WriteString(colorDim + f.Hint + colorReset + "\n")
	} else {
		sb.WriteString("\n")
	}

	sb.WriteString("┌" + border + "┐\n")

	// Display lines (bottom-aligned: blank lines at top, content above prompt)
	blankCount := f.Height - len(f.DisplayLines)
	for i := 0; i < blankCount; i++ {
		sb.WriteString("│" + strings.Repeat(" ", f.Width) + "│\n")
	}
	for _, line := range f.DisplayLines {
		runeLen := utf8.RuneCountInString(line)
		if runeLen > f.Width {
			line = string([]rune(line)[:f.Width])
			runeLen = f.Width
		}
		padding := f.Width - runeLen
		sb.WriteString("│" + line + strings.Repeat(" ", padding) + "│\n")
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

	// Keyboard
	sb.WriteString("\n")
	sb.WriteString(renderANSIKeyboard(f.ValidKeys, f.HintKey))

	return sb.String()
}

// renderDialogLineANSI emits a dialog line with backtick-marked spans
// highlighted in bold green (matching keyboard hints).
func renderDialogLineANSI(line DialogLine) string {
	baseColor := dialogColorsANSI[line.ColorIdx%len(dialogColorsANSI)]
	parts := strings.Split(line.Text, "`")
	var sb strings.Builder
	for i, p := range parts {
		if i%2 == 1 {
			sb.WriteString(colorGreen + p + colorReset)
		} else {
			sb.WriteString(baseColor + p + colorReset)
		}
	}
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
