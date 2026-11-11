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

	// The box holds Height history rows plus one prompt row. A long
	// active line wraps onto more rows, like a real terminal, and pushes
	// history up instead of growing the box. If it needs more rows than
	// the box has, its top scrolls off so the cursor stays visible.
	boxRows := f.Height + 1
	prompt := layoutPromptLines(f.Prompt, f.PromptInputOn, f.PromptInputOff, f.CursorOnPath, f.Width)
	if len(prompt) > boxRows {
		prompt = prompt[len(prompt)-boxRows:]
	}
	history := f.DisplayLines
	if room := boxRows - len(prompt); len(history) > room {
		history = history[len(history)-room:]
	}

	// History is bottom-aligned with blank padding above.
	for i := len(history) + len(prompt); i < boxRows; i++ {
		lines = append(lines, layoutBoxBlank(f.Width))
	}
	for _, dl := range history {
		lines = append(lines, layoutHistoryLine(dl, f.Width))
	}
	lines = append(lines, prompt...)

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
	lines = append(lines, layoutKeyboard(f.ValidKeys, f.HintKey, f.Width+2)...) // full screen width, not the box interior

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
		_, promptSegs = splitSegments(promptSegments(dl.Prompt), dl.PromptSkip)
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

// layoutPromptLines emits the active prompt: host-colored prompt frame,
// green on-path input, white off-path input, then a colored cursor block.
// When that is wider than the box it wraps onto as many rows as it needs,
// each segment keeping its style on whichever row it lands.
func layoutPromptLines(p PromptInfo, onPath, offPath string, cursorOnPath bool, width int) []RenderedLine {
	cursorStyle := StyleCursorOff
	if cursorOnPath {
		cursorStyle = StyleCursorOn
	}
	segs := promptSegments(p)
	if onPath != "" {
		segs = append(segs, Segment{Text: onPath, Style: StyleOnPath})
	}
	if offPath != "" {
		segs = append(segs, Segment{Text: offPath, Style: StylePromptOff})
	}
	segs = append(segs, Segment{Text: "█", Style: cursorStyle})

	if width < 1 {
		width = 1
	}
	var rows []RenderedLine
	for len(segs) > 0 {
		var row []Segment
		row, segs = splitSegments(segs, width)
		line := RenderedLine{{Text: "│", Style: StyleBox}}
		line = append(line, row...)
		if pad := width - segmentsLen(row); pad > 0 {
			line = append(line, Segment{Text: strings.Repeat(" ", pad)})
		}
		line = append(line, Segment{Text: "│", Style: StyleBox})
		rows = append(rows, line)
	}
	return rows
}

// splitSegments cuts segs after the first n runes, splitting the segment
// that straddles the cut so both halves keep its style.
func splitSegments(segs []Segment, n int) (head, tail []Segment) {
	for i, s := range segs {
		l := utf8.RuneCountInString(s.Text)
		if n >= l {
			head = append(head, s)
			n -= l
			continue
		}
		if n > 0 {
			r := []rune(s.Text)
			left, right := s, s
			left.Text, right.Text = string(r[:n]), string(r[n:])
			head = append(head, left)
			tail = append(tail, right)
		} else {
			tail = append(tail, s)
		}
		return head, append(tail, segs[i+1:]...)
	}
	return head, nil
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
func layoutKeyboard(valid []process.Datum, hint process.Datum, width int) []RenderedLine {
	validSet := buildDatumSet(valid)
	indents := []string{" ", "  ", "   "}
	var lines []RenderedLine
	for row, keys := range keyboardRows {
		line := RenderedLine{{Text: indents[row]}}
		for i, key := range keys {
			if i > 0 {
				line = append(line, Segment{Text: " "})
			}
			line = append(line, layoutKey(key, KeyDatum(key), validSet, hint))
		}
		lines = append(lines, line)
	}
	labels := specialRowLabels(width)
	specialLine := RenderedLine{{Text: " "}}
	for i, sk := range specialKeys {
		if i > 0 {
			specialLine = append(specialLine, Segment{Text: " "})
		}
		specialLine = append(specialLine, layoutKey(labels[i], sk.datum, validSet, hint))
	}
	lines = append(lines, specialLine)
	return lines
}

// layoutKey styles one keyboard key. Keys the player can press right now
// also carry their KeyName so a front end can make them tappable; dim
// keys carry none, so tapping them does nothing.
func layoutKey(label string, datum process.Datum, validSet datumSet, hint process.Datum) Segment {
	seg := Segment{Text: label, Style: StyleKeyDim}
	switch {
	case hint != nil && datumEqual(datum, hint):
		seg.Style = StyleOnPath
	case validSet.contains(datum):
		seg.Style = StyleKeyValid
	}
	if validSet.contains(datum) {
		seg.Key = KeyName(datum)
	}
	return seg
}
