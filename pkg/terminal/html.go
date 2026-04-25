package terminal

import (
	"fmt"
	"html"
	"strings"
	"unicode/utf8"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// dialogColorCount must match the number of dialog-N CSS classes in style.css.
const dialogColorCount = 4

// HTMLRenderer renders frames as HTML with CSS classes for styling.
type HTMLRenderer struct{}

func NewHTML() *HTMLRenderer {
	return &HTMLRenderer{}
}

func (h *HTMLRenderer) Render(f Frame) string {
	var sb strings.Builder
	border := strings.Repeat("─", f.Width)

	sb.WriteString("<pre>")

	// Dialog area — padded to fixed height so terminal stays put
	blankLines := f.DialogSpace - len(f.Dialog)
	for i := 0; i < blankLines; i++ {
		sb.WriteString("\n")
	}
	for _, line := range f.Dialog {
		sb.WriteString(renderDialogLineHTML(line) + "\n")
	}

	// Hint line — always occupies 1 line (blank if no hint)
	if f.Hint != "" {
		sb.WriteString(`<span class="hint">` + html.EscapeString(f.Hint) + "</span>\n")
	} else {
		sb.WriteString("\n")
	}

	// Box top
	sb.WriteString(`<span class="box">┌` + border + "┐</span>\n")

	// Display lines (bottom-aligned: blank lines at top, content above prompt)
	blankCount := f.Height - len(f.DisplayLines)
	for i := 0; i < blankCount; i++ {
		sb.WriteString(`<span class="box">│</span>` + strings.Repeat(" ", f.Width) + `<span class="box">│</span>` + "\n")
	}
	for _, line := range f.DisplayLines {
		sb.WriteString(`<span class="box">│</span>`)
		runeLen := utf8.RuneCountInString(line)
		if runeLen > f.Width {
			line = string([]rune(line)[:f.Width])
			runeLen = f.Width
		}
		padding := f.Width - runeLen
		sb.WriteString(html.EscapeString(line) + strings.Repeat(" ", padding))
		sb.WriteString(`<span class="box">│</span>` + "\n")
	}

	// Prompt line — prefix (blue), on-path input (green), off-path input (white).
	prefix := f.PromptPrefix
	onPath := f.PromptInputOn
	offPath := f.PromptInputOff
	prefixLen := utf8.RuneCountInString(prefix)
	onLen := utf8.RuneCountInString(onPath)
	offLen := utf8.RuneCountInString(offPath)
	totalLen := prefixLen + onLen + offLen
	if totalLen > f.Width {
		excess := totalLen - f.Width
		if offLen >= excess {
			offPath = string([]rune(offPath)[:offLen-excess])
			offLen -= excess
		} else {
			excess -= offLen
			offPath = ""
			offLen = 0
			if onLen >= excess {
				onPath = string([]rune(onPath)[:onLen-excess])
				onLen -= excess
			} else {
				onPath = ""
				onLen = 0
				prefix = string([]rune(prefix)[:f.Width])
				prefixLen = f.Width
			}
		}
		totalLen = prefixLen + onLen + offLen
	}
	padding := f.Width - totalLen
	sb.WriteString(`<span class="box">│</span>`)
	sb.WriteString(`<span class="prompt">` + html.EscapeString(prefix) + "</span>")
	if onLen > 0 {
		sb.WriteString(`<span class="key-hint">` + html.EscapeString(onPath) + "</span>")
	}
	if offLen > 0 {
		sb.WriteString(`<span class="prompt-off">` + html.EscapeString(offPath) + "</span>")
	}
	sb.WriteString(strings.Repeat(" ", padding))
	sb.WriteString(`<span class="box">│</span>` + "\n")

	// Box bottom
	sb.WriteString(`<span class="box">└` + border + "┘</span>\n")

	// Thought line below the box.
	if f.Thought != "" {
		sb.WriteString(`<span class="hint">` + html.EscapeString(f.Thought) + "</span>\n")
	} else {
		sb.WriteString("\n")
	}

	// Keyboard
	sb.WriteString("\n")
	sb.WriteString(renderHTMLKeyboard(f.ValidKeys, f.HintKey))

	sb.WriteString("</pre>")
	return sb.String()
}

// renderDialogLineHTML emits a dialog line with backtick-marked spans
// highlighted in bright green (matching keyboard hints).
func renderDialogLineHTML(line DialogLine) string {
	class := fmt.Sprintf("dialog dialog-%d", line.ColorIdx%dialogColorCount)
	parts := strings.Split(line.Text, "`")
	var sb strings.Builder
	for i, p := range parts {
		if i%2 == 1 {
			sb.WriteString(`<span class="key-hint">` + html.EscapeString(p) + `</span>`)
		} else {
			sb.WriteString(`<span class="` + class + `">` + html.EscapeString(p) + `</span>`)
		}
	}
	return sb.String()
}

func renderHTMLKeyboard(valid []process.Datum, hint process.Datum) string {
	validSet := buildDatumSet(valid)
	var sb strings.Builder

	indents := []string{" ", "  ", "   "}
	for row, keys := range keyboardRows {
		sb.WriteString(indents[row])
		for i, key := range keys {
			if i > 0 {
				sb.WriteString(" ")
			}
			datum := process.Chars(key)
			sb.WriteString(colorKeyHTML(key, datum, validSet, hint))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(" ")
	for i, sk := range specialKeys {
		if i > 0 {
			sb.WriteString(" ")
		}
		label := "[" + sk.label + "]"
		sb.WriteString(colorKeyHTML(label, sk.datum, validSet, hint))
	}
	sb.WriteString("\n")

	return sb.String()
}

func colorKeyHTML(label string, datum process.Datum, validSet datumSet, hint process.Datum) string {
	class := "key-dim"
	if hint != nil && datumEqual(datum, hint) {
		class = "key-hint"
	} else if validSet.contains(datum) {
		class = "key-valid"
	}
	return `<span class="` + class + `">` + html.EscapeString(label) + "</span>"
}
