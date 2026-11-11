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

// ReflowLines wraps lines to the given width and returns the last height
// display lines (for scrolling).
// Only the lines needed to fill height are wrapped.
func ReflowLines(lines []string, width, height int) []string {
	return tailWrap(lines, height, func(l string) []string { return WrapLine(l, width) })
}

// tailWrap wraps entries from the newest backwards until it has height
// display lines, then returns those lines oldest first. Rendering costs
// what's on screen, however long the history is.
func tailWrap[E, L any](entries []E, height int, wrap func(E) []L) []L {
	var rev [][]L
	n := 0
	for i := len(entries) - 1; i >= 0 && n < height; i-- {
		w := wrap(entries[i])
		rev = append(rev, w)
		n += len(w)
	}
	out := make([]L, 0, n)
	for i := len(rev) - 1; i >= 0; i-- {
		out = append(out, rev[i]...)
	}
	if len(out) > height {
		out = out[len(out)-height:]
	}
	return out
}

// DisplayLine is one rendered line in the terminal box. Prompt (if set)
// renders in the prompt-color spans (per-host). A wrapped entry carries
// its Prompt on every row that shows part of the prompt; PromptSkip is how
// many prompt runes earlier rows already showed. Rows holding only input
// have a zero Prompt.
type DisplayLine struct {
	Prompt     PromptInfo
	PromptSkip int
	Text       string
}

// ReflowHistory wraps each history entry to the given width and returns the
// last height display lines.
func ReflowHistory(lines []HistoryLine, width, height int) []DisplayLine {
	return tailWrap(lines, height, func(hl HistoryLine) []DisplayLine { return wrapHistoryLine(hl, width) })
}

// wrapHistoryLine wraps prompt and input as one run of text, the way a
// terminal does, so a prompt wider than the box still leaves the input
// visible on the rows below.
func wrapHistoryLine(hl HistoryLine, width int) []DisplayLine {
	if hl.Prompt.IsZero() {
		// Output line — no prompt prefix.
		var out []DisplayLine
		for _, w := range WrapLine(hl.Input, width) {
			out = append(out, DisplayLine{Text: w})
		}
		return out
	}
	if width <= 0 {
		return []DisplayLine{{Prompt: hl.Prompt, Text: hl.Input}}
	}
	prefixLen := utf8.RuneCountInString(hl.Prompt.promptPrefixText())
	input := []rune(hl.Input)
	var out []DisplayLine
	skip := 0
	for skip < prefixLen {
		room := width - (prefixLen - skip)
		if room < 0 {
			out = append(out, DisplayLine{Prompt: hl.Prompt, PromptSkip: skip})
			skip += width
			continue
		}
		if room > len(input) {
			room = len(input)
		}
		out = append(out, DisplayLine{Prompt: hl.Prompt, PromptSkip: skip, Text: string(input[:room])})
		input = input[room:]
		skip = prefixLen
	}
	if len(input) > 0 {
		for _, w := range WrapLine(string(input), width) {
			out = append(out, DisplayLine{Text: w})
		}
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
