package terminal

import "strings"

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorDim    = "\033[2m"    // dim grey
	colorWhite  = "\033[1;37m" // bold white
	colorGreen  = "\033[1;32m" // bold green
	colorPrompt = "\033[1;34m" // bold blue (active prompt only)
)

// dialogColorsANSI cycles for each new dialog batch so the user can see when
// a fresh message has arrived. Green is intentionally absent — it would
// clash with the bold-green highlight used for backtick command spans.
var dialogColorsANSI = []string{
	"\033[31m", // red
	"\033[33m", // yellow
	"\033[34m", // blue
	"\033[35m", // purple
}

// ANSIRenderer renders frames using ANSI escape codes and box-drawing characters.
type ANSIRenderer struct{}

func NewANSI() *ANSIRenderer { return &ANSIRenderer{} }

func (a *ANSIRenderer) Render(f Frame) string {
	var sb strings.Builder
	for _, line := range Layout(f) {
		for _, seg := range line {
			open := ansiOpen(seg.Style, seg.BatchIdx)
			if open != "" {
				sb.WriteString(open)
				sb.WriteString(seg.Text)
				sb.WriteString(colorReset)
			} else {
				sb.WriteString(seg.Text)
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// ansiOpen returns the ANSI escape that opens a span of the given style.
// Empty string means "no styling" — the renderer omits the escape and the
// trailing reset entirely.
func ansiOpen(style Style, batchIdx int) string {
	switch style {
	case StyleBox, StyleDim:
		return colorDim
	case StylePrompt:
		return colorPrompt
	case StylePromptOff, StyleCursorOff, StyleKeyValid:
		return colorWhite
	case StyleOnPath, StyleCursorOn:
		return colorGreen
	case StyleDialog:
		return dialogColorsANSI[batchIdx%len(dialogColorsANSI)]
	case StyleKeyDim:
		return colorDim
	}
	return ""
}
