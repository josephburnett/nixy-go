package terminal

import (
	"html"
	"strings"
	"unicode/utf8"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// HTMLRenderer renders frames as HTML with CSS classes for styling.
type HTMLRenderer struct{}

func NewHTML() *HTMLRenderer {
	return &HTMLRenderer{}
}

func (h *HTMLRenderer) Render(f Frame) string {
	var sb strings.Builder
	border := strings.Repeat("─", f.Width)

	sb.WriteString("<pre>")

	// Dialog — above the terminal box
	for _, line := range f.Dialog {
		sb.WriteString(`<span class="dialog">` + html.EscapeString(line) + "</span>\n")
	}

	// Hint — above the terminal box
	if f.Hint != "" {
		sb.WriteString(`<span class="hint">` + html.EscapeString(f.Hint) + "</span>\n")
	}

	// Box top
	sb.WriteString(`<span class="box">┌` + border + "┐</span>\n")

	// Display lines
	for i := 0; i < f.Height; i++ {
		sb.WriteString(`<span class="box">│</span>`)
		if i < len(f.DisplayLines) {
			line := f.DisplayLines[i]
			runeLen := utf8.RuneCountInString(line)
			if runeLen > f.Width {
				line = string([]rune(line)[:f.Width])
				runeLen = f.Width
			}
			padding := f.Width - runeLen
			sb.WriteString(html.EscapeString(line) + strings.Repeat(" ", padding))
		} else {
			sb.WriteString(strings.Repeat(" ", f.Width))
		}
		sb.WriteString(`<span class="box">│</span>` + "\n")
	}

	// Prompt line
	sb.WriteString(`<span class="box">│</span>`)
	prompt := f.Prompt
	runeLen := utf8.RuneCountInString(prompt)
	if runeLen > f.Width {
		prompt = string([]rune(prompt)[:f.Width])
		runeLen = f.Width
	}
	padding := f.Width - runeLen
	sb.WriteString(`<span class="prompt">` + html.EscapeString(prompt) + "</span>" + strings.Repeat(" ", padding))
	sb.WriteString(`<span class="box">│</span>` + "\n")

	// Box bottom
	sb.WriteString(`<span class="box">└` + border + "┘</span>\n")

	// Keyboard
	sb.WriteString("\n")
	sb.WriteString(renderHTMLKeyboard(f.ValidKeys, f.HintKey))

	sb.WriteString("</pre>")
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
