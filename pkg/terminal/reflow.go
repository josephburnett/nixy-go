package terminal

import (
	"strings"
	"unicode/utf8"
)

// WrapLine breaks a single line into multiple display lines at the given width.
// If width <= 0, it returns the line as-is.
func WrapLine(line string, width int) []string {
	if width <= 0 {
		return []string{line}
	}
	if utf8.RuneCountInString(line) <= width {
		return []string{line}
	}
	var result []string
	runes := []rune(line)
	for len(runes) > 0 {
		end := width
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, string(runes[:end]))
		runes = runes[end:]
	}
	return result
}

// ReflowLines wraps all original lines to the given width, then returns
// the last height display lines (for scrolling).
func ReflowLines(lines []string, width, height int) []string {
	var display []string
	for _, line := range lines {
		display = append(display, WrapLine(line, width)...)
	}
	if len(display) > height {
		display = display[len(display)-height:]
	}
	return display
}

// DisplayLine is one rendered line in the terminal box. Prompt (if set)
// renders in the prompt-color spans (per-host) and only appears on the
// first line of a wrapped block; continuation lines have a zero Prompt.
type DisplayLine struct {
	Prompt PromptInfo
	Text   string
}

// ReflowHistory wraps each history entry to the given width and returns the
// last height display lines. The prompt prefix (if any) appears only on the
// first wrapped sub-line of each entry.
func ReflowHistory(lines []HistoryLine, width, height int) []DisplayLine {
	var display []DisplayLine
	for _, hl := range lines {
		display = append(display, wrapHistoryLine(hl, width)...)
	}
	if len(display) > height {
		display = display[len(display)-height:]
	}
	return display
}

func wrapHistoryLine(hl HistoryLine, width int) []DisplayLine {
	prefixText := hl.Prompt.promptPrefixText()
	if hl.Prompt.IsZero() {
		// Output line — no prompt prefix.
		prefixText = ""
	}
	if width <= 0 {
		return []DisplayLine{{Prompt: hl.Prompt, Text: hl.Input}}
	}
	prefixLen := utf8.RuneCountInString(prefixText)
	if prefixLen >= width {
		// Prefix alone overflows — show the prompt only and skip the input.
		return []DisplayLine{{Prompt: hl.Prompt}}
	}
	first := width - prefixLen
	inputRunes := []rune(hl.Input)
	if len(inputRunes) <= first {
		return []DisplayLine{{Prompt: hl.Prompt, Text: hl.Input}}
	}
	out := []DisplayLine{{Prompt: hl.Prompt, Text: string(inputRunes[:first])}}
	for _, w := range WrapLine(string(inputRunes[first:]), width) {
		out = append(out, DisplayLine{Text: w})
	}
	return out
}

// WrapWords wraps a line at word boundaries (spaces), keeping backtick-
// delimited spans atomic so command highlighting isn't split across lines.
// Falls back to char-level wrapping for tokens that exceed the width.
func WrapWords(line string, width int) []string {
	if width <= 0 {
		return []string{line}
	}
	tokens := tokenizeWords(line)
	var lines []string
	var cur strings.Builder
	curLen := 0
	for _, tok := range tokens {
		tokLen := utf8.RuneCountInString(tok)
		// Token longer than the whole width — break it raw.
		if tokLen > width {
			if curLen > 0 {
				lines = append(lines, cur.String())
				cur.Reset()
				curLen = 0
			}
			lines = append(lines, WrapLine(tok, width)...)
			continue
		}
		switch {
		case curLen == 0:
			cur.WriteString(tok)
			curLen = tokLen
		case curLen+1+tokLen <= width:
			cur.WriteString(" ")
			cur.WriteString(tok)
			curLen += 1 + tokLen
		default:
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(tok)
			curLen = tokLen
		}
	}
	if curLen > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

// tokenizeWords splits on spaces but treats backtick-delimited spans as
// single tokens, so wrapping never falls in the middle of a `command`.
func tokenizeWords(line string) []string {
	var tokens []string
	var cur strings.Builder
	inBack := false
	for _, r := range line {
		if r == '`' {
			inBack = !inBack
			cur.WriteRune(r)
			continue
		}
		if r == ' ' && !inBack {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// TruncateRunes cuts a string to at most width display columns.
func TruncateRunes(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	return string([]rune(s)[:width])
}
