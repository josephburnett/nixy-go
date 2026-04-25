package terminal

import (
	"strings"
	"unicode/utf8"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// Layout converts a Frame into a sequence of styled lines. This is the
// single source of layout truth — both renderers consume its output and
// only differ in how they map Style values to escape codes / CSS classes.
func Layout(f Frame) []RenderedLine {
	var lines []RenderedLine

	// Dialog area — padded above so the terminal box position is stable.
	blanks := f.DialogSpace - len(f.Dialog)
	for i := 0; i < blanks; i++ {
		lines = append(lines, nil)
	}
	for _, dl := range f.Dialog {
		lines = append(lines, layoutDialogLine(dl))
	}

	// Box top border.
	lines = append(lines, RenderedLine{{Text: "┌" + strings.Repeat("─", f.Width) + "┐", Style: StyleBox}})

	// Display content (history): bottom-aligned with blank padding above.
	blankCount := f.Height - len(f.DisplayLines)
	for i := 0; i < blankCount; i++ {
		lines = append(lines, layoutBoxBlank(f.Width))
	}
	for _, dl := range f.DisplayLines {
		lines = append(lines, layoutHistoryLine(dl, f.Width))
	}

	// Active prompt with cursor.
	lines = append(lines, layoutPromptLine(
		f.Prompt, f.PromptInputOn, f.PromptInputOff, f.CursorOnPath, f.Width,
	))

	// Box bottom.
	lines = append(lines, RenderedLine{{Text: "└" + strings.Repeat("─", f.Width) + "┘", Style: StyleBox}})

	// Status slot below the box: notice if set, otherwise thought.
	if f.Status != "" {
		style := StyleDim
		if f.StatusIsNotice {
			style = StyleNotice
		}
		lines = append(lines, RenderedLine{{Text: f.Status, Style: style}})
	} else {
		lines = append(lines, nil)
	}

	// Blank gutter then keyboard.
	lines = append(lines, nil)
	lines = append(lines, layoutKeyboard(f.ValidKeys, f.HintKey)...)

	return lines
}

// layoutBoxBlank returns the empty content row inside the terminal box.
func layoutBoxBlank(width int) RenderedLine {
	return RenderedLine{
		{Text: "│", Style: StyleBox},
		{Text: strings.Repeat(" ", width)},
		{Text: "│", Style: StyleBox},
	}
}

// layoutDialogLine emits a dialog line with backtick-marked spans
// highlighted in the on-path color. Empty Text yields nil (a blank
// paragraph-separator line).
func layoutDialogLine(dl DialogLine) RenderedLine {
	if dl.Text == "" {
		return nil
	}
	var line RenderedLine
	parts := strings.Split(dl.Text, "`")
	for i, p := range parts {
		if i%2 == 1 {
			line = append(line, Segment{Text: p, Style: StyleOnPath})
		} else {
			line = append(line, Segment{Text: p, Style: StyleDialog})
		}
	}
	return line
}

// layoutHistoryLine emits a single history row inside the box, with the
// prompt prefix (if any) styled per-host and any padding to fill the row.
func layoutHistoryLine(dl DisplayLine, width int) RenderedLine {
	var promptSegs []Segment
	// Zero PromptInfo means "this is command output, not an entered command".
	if !dl.Prompt.IsZero() {
		promptSegs = promptSegments(dl.Prompt)
	}
	prefixLen := segmentsLen(promptSegs)
	text := dl.Text
	textLen := utf8.RuneCountInString(text)
	total := prefixLen + textLen
	if total > width {
		excess := total - width
		if textLen >= excess {
			text = string([]rune(text)[:textLen-excess])
			textLen -= excess
		} else {
			// Prompt alone overflows — drop the input and truncate the prompt
			// segments collectively.
			text = ""
			textLen = 0
			promptSegs = truncateSegments(promptSegs, width)
			prefixLen = segmentsLen(promptSegs)
		}
		total = prefixLen + textLen
	}
	line := RenderedLine{{Text: "│", Style: StyleBox}}
	line = append(line, promptSegs...)
	if textLen > 0 {
		line = append(line, Segment{Text: text})
	}
	line = append(line, Segment{Text: strings.Repeat(" ", width-total)})
	line = append(line, Segment{Text: "│", Style: StyleBox})
	return line
}

// layoutPromptLine emits the active prompt: host-colored prompt frame,
// green on-path input, white off-path input, then a colored cursor block.
// Truncates from the right (off-path first, then on-path, then prompt) to
// fit within width.
func layoutPromptLine(p PromptInfo, onPath, offPath string, cursorOnPath bool, width int) RenderedLine {
	promptSegs := promptSegments(p)
	prefixLen := segmentsLen(promptSegs)
	onLen := utf8.RuneCountInString(onPath)
	offLen := utf8.RuneCountInString(offPath)
	const cursorWidth = 1
	total := prefixLen + onLen + offLen + cursorWidth
	if total > width {
		excess := total - width
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
				promptSegs = truncateSegments(promptSegs, width-cursorWidth)
				prefixLen = segmentsLen(promptSegs)
			}
		}
		total = prefixLen + onLen + offLen + cursorWidth
	}
	cursorStyle := StyleCursorOff
	if cursorOnPath {
		cursorStyle = StyleCursorOn
	}
	line := RenderedLine{{Text: "│", Style: StyleBox}}
	line = append(line, promptSegs...)
	if onLen > 0 {
		line = append(line, Segment{Text: onPath, Style: StyleOnPath})
	}
	if offLen > 0 {
		line = append(line, Segment{Text: offPath, Style: StylePromptOff})
	}
	line = append(line, Segment{Text: "█", Style: cursorStyle})
	line = append(line, Segment{Text: strings.Repeat(" ", width-total)})
	line = append(line, Segment{Text: "│", Style: StyleBox})
	return line
}

// promptSegments turns a PromptInfo into styled segments. The host name
// gets StyleHost (per-host color); everything else gets StylePrompt.
// For Raw prompts (e.g. "login: ") the whole thing is StylePrompt.
// A zero PromptInfo yields a minimal "> " default.
func promptSegments(p PromptInfo) []Segment {
	if p.IsZero() {
		return []Segment{{Text: "> ", Style: StylePrompt}}
	}
	if p.Raw != "" {
		return []Segment{{Text: p.Raw, Style: StylePrompt}}
	}
	return []Segment{
		{Text: p.User + "@", Style: StylePrompt},
		{Text: p.Host, Style: StyleHost, Host: p.Host},
		{Text: ":" + p.Path + "> ", Style: StylePrompt},
	}
}

// segmentsLen returns the visible rune count across a slice of segments.
func segmentsLen(segs []Segment) int {
	n := 0
	for _, s := range segs {
		n += utf8.RuneCountInString(s.Text)
	}
	return n
}

// truncateSegments cuts a slice of segments to at most width display
// columns, dropping or truncating from the right.
func truncateSegments(segs []Segment, width int) []Segment {
	if width <= 0 {
		return nil
	}
	out := make([]Segment, 0, len(segs))
	used := 0
	for _, s := range segs {
		segLen := utf8.RuneCountInString(s.Text)
		if used+segLen <= width {
			out = append(out, s)
			used += segLen
			continue
		}
		// Take only as much as fits.
		remaining := width - used
		if remaining > 0 {
			s.Text = string([]rune(s.Text)[:remaining])
			out = append(out, s)
		}
		break
	}
	return out
}

// layoutKeyboard emits the four-row keyboard: three letter rows then a
// special-keys row. Each key carries StyleOnPath if it's the hint,
// StyleKeyValid if it's typeable now, StyleKeyDim otherwise.
func layoutKeyboard(valid []process.Datum, hint process.Datum) []RenderedLine {
	validSet := buildDatumSet(valid)
	indents := []string{" ", "  ", "   "}
	var lines []RenderedLine
	for row, keys := range keyboardRows {
		line := RenderedLine{{Text: indents[row]}}
		for i, key := range keys {
			if i > 0 {
				line = append(line, Segment{Text: " "})
			}
			line = append(line, layoutKey(key, process.Chars(key), validSet, hint))
		}
		lines = append(lines, line)
	}
	specialLine := RenderedLine{{Text: " "}}
	for i, sk := range specialKeys {
		if i > 0 {
			specialLine = append(specialLine, Segment{Text: " "})
		}
		specialLine = append(specialLine, layoutKey("["+sk.label+"]", sk.datum, validSet, hint))
	}
	lines = append(lines, specialLine)
	return lines
}

func layoutKey(label string, datum process.Datum, validSet datumSet, hint process.Datum) Segment {
	switch {
	case hint != nil && datumEqual(datum, hint):
		return Segment{Text: label, Style: StyleOnPath}
	case validSet.contains(datum):
		return Segment{Text: label, Style: StyleKeyValid}
	default:
		return Segment{Text: label, Style: StyleKeyDim}
	}
}
