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

// colorDialog is the single color used for all Nixy dialog (matches the
// nixy host color so it's clear when Nixy is speaking).
const colorDialog = "\033[38;5;209m" // salmon

// hostColorsANSI maps host names to their prompt color. Hosts not listed
// fall back to colorPrompt (laptop is the user's home — same color as the
// prompt frame).
var hostColorsANSI = map[string]string{
	"laptop": colorPrompt,        // bold blue
	"nixy":   "\033[38;5;209m",   // salmon
	"server": "\033[38;5;73m",    // teal / cadet
}

// ANSIRenderer renders frames using ANSI escape codes and box-drawing characters.
type ANSIRenderer struct{}

func NewANSI() *ANSIRenderer { return &ANSIRenderer{} }

func (a *ANSIRenderer) Render(f Frame) string {
	var sb strings.Builder
	for _, line := range Layout(f) {
		for _, seg := range line {
			open := ansiOpen(seg)
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
func ansiOpen(seg Segment) string {
	switch seg.Style {
	case StyleBox, StyleDim, StyleKeyDim:
		return colorDim
	case StyleNotice:
		return colorWhite
	case StylePrompt:
		return colorPrompt
	case StyleHost:
		if c, ok := hostColorsANSI[seg.Host]; ok {
			return c
		}
		return colorPrompt
	case StylePromptOff, StyleCursorOff, StyleKeyValid:
		return colorWhite
	case StyleOnPath, StyleCursorOn:
		return colorGreen
	case StyleDialog:
		return colorDialog
	}
	return ""
}
