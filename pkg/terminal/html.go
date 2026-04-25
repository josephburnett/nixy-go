package terminal

import (
	"fmt"
	"html"
	"strings"
)

// dialogColorCount must match the number of dialog-N CSS classes in style.css.
const dialogColorCount = 4

// HTMLRenderer renders frames as HTML with CSS classes for styling.
type HTMLRenderer struct{}

func NewHTML() *HTMLRenderer { return &HTMLRenderer{} }

func (h *HTMLRenderer) Render(f Frame) string {
	var sb strings.Builder
	sb.WriteString("<pre>")
	for _, line := range Layout(f) {
		for _, seg := range line {
			class := htmlClass(seg.Style, seg.BatchIdx)
			text := html.EscapeString(seg.Text)
			if class != "" {
				sb.WriteString(`<span class="`)
				sb.WriteString(class)
				sb.WriteString(`">`)
				sb.WriteString(text)
				sb.WriteString(`</span>`)
			} else {
				sb.WriteString(text)
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("</pre>")
	return sb.String()
}

// htmlClass returns the CSS class name for a span of the given style.
// Empty string means "emit raw text without a span wrapper."
func htmlClass(style Style, batchIdx int) string {
	switch style {
	case StyleBox:
		return "box"
	case StylePrompt:
		return "prompt"
	case StylePromptOff:
		return "prompt-off"
	case StyleOnPath:
		return "key-hint"
	case StyleDim:
		return "hint"
	case StyleNotice:
		return "notice"
	case StyleCursorOn:
		return "cursor-on"
	case StyleCursorOff:
		return "cursor-off"
	case StyleKeyValid:
		return "key-valid"
	case StyleKeyDim:
		return "key-dim"
	case StyleDialog:
		return fmt.Sprintf("dialog dialog-%d", batchIdx%dialogColorCount)
	}
	return ""
}
