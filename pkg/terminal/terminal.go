package terminal

import (
	"github.com/josephburnett/nixy-go/pkg/process"
)

const (
	// Fixed layout constants
	boxBorders    = 3 // top border + bottom border + prompt line
	statusLine    = 1 // below the box: notice (errors etc) or thought
	keyboardLines = 6 // 3 letter rows + 1 special row + 1 blank + 1 trailing
)

// T is the terminal controller. It composes content state with a renderer.
type T struct {
	State        State
	ScreenWidth  int
	ScreenHeight int
	renderer     Renderer
}

func New(r Renderer) *T {
	return &T{
		ScreenWidth:  57, // 55 content + 2 borders
		ScreenHeight: 30,
		renderer:     r,
	}
}

func (t *T) Resize(w, h int) {
	if w < 22 {
		w = 22
	}
	if h < 15 {
		h = 15
	}
	t.ScreenWidth = w
	t.ScreenHeight = h
}

func (t *T) Write(in process.Data) error {
	return t.State.Write(in)
}

// Notify sets the line shown above the terminal box. Empty string clears it.
// Used for errors (e.g. Ctrl+C confirmation), not for ongoing planner hints.
func (t *T) Notify(msg string) {
	t.State.Notice = msg
}

func (t *T) SetThought(s string) {
	t.State.Thought = s
}

func (t *T) SetPrompt(p PromptInfo) {
	t.State.Prompt = p
}

func (t *T) SetPromptTarget(s string) {
	t.State.PromptTarget = s
}

// SetDialog appends a batch of dialog lines. Successive calls are
// separated by a blank line so paragraphs are visually distinct.
func (t *T) SetDialog(lines []string) {
	if len(lines) == 0 {
		return
	}
	if len(t.State.Dialog) > 0 {
		t.State.Dialog = append(t.State.Dialog, DialogLine{}) // paragraph separator
	}
	for _, l := range lines {
		t.State.Dialog = append(t.State.Dialog, DialogLine{Text: l})
	}
}

func (t *T) SetKeyboard(valid []process.Datum, hint process.Datum) {
	t.State.ValidKeys = valid
	t.State.HintKey = hint
}

func (t *T) Render() string {
	contentWidth := t.ScreenWidth - 2 // subtract left+right borders

	// Terminal box gets at most 50% of screen
	termBoxHeight := t.ScreenHeight / 2
	if termBoxHeight < boxBorders+1 {
		termBoxHeight = boxBorders + 1
	}
	termContentHeight := termBoxHeight - boxBorders

	// Single status slot below the box: notice takes priority over thought.
	// Both truncate to width so wrapping never pushes the rest of the
	// layout around.
	statusStr := ""
	statusIsNotice := false
	switch {
	case t.State.Notice != "":
		statusStr = TruncateRunes(t.State.Notice, contentWidth)
		statusIsNotice = true
	case t.State.Thought != "":
		statusStr = TruncateRunes("("+t.State.Thought+"...)", contentWidth)
	}

	// Dialog fills remaining space above the terminal box and below the
	// status line + keyboard.
	dialogSpace := t.ScreenHeight - termBoxHeight - statusLine - keyboardLines
	if dialogSpace < 0 {
		dialogSpace = 0
	}

	// Wrap each accumulated dialog line to the screen width. Empty entries
	// are paragraph separators — preserve them as-is.
	var wrappedDialog []DialogLine
	for _, dl := range t.State.Dialog {
		if dl.Text == "" {
			wrappedDialog = append(wrappedDialog, DialogLine{})
			continue
		}
		for _, w := range WrapWords(dl.Text, t.ScreenWidth) {
			wrappedDialog = append(wrappedDialog, DialogLine{Text: w})
		}
	}
	var dialogToShow []DialogLine
	if dialogSpace > 0 && len(wrappedDialog) > 0 {
		start := len(wrappedDialog) - dialogSpace
		if start < 0 {
			start = 0
		}
		dialogToShow = wrappedDialog[start:]
	}

	displayLines := ReflowHistory(t.State.Lines, contentWidth, termContentHeight)
	onPath, offPath := splitOnPath(t.State.Line, t.State.PromptTarget)

	// Cursor follows the on-path/off-path semantics of the keyboard hint:
	// green when the planner suggests typing forward (a char or Enter),
	// white when it suggests backspace or has no suggestion.
	cursorOnPath := t.State.HintKey != nil && t.State.HintKey != process.TermBackspace

	f := Frame{
		DisplayLines:   displayLines,
		Prompt:         t.State.Prompt,
		PromptInputOn:  onPath,
		PromptInputOff: offPath,
		CursorOnPath:   cursorOnPath,
		Dialog:         dialogToShow,
		DialogSpace:    dialogSpace,
		Status:         statusStr,
		StatusIsNotice: statusIsNotice,
		ValidKeys:      t.State.ValidKeys,
		HintKey:        t.State.HintKey,
		Width:          contentWidth,
		Height:         termContentHeight,
	}
	return t.renderer.Render(f)
}

// splitOnPath returns the longest prefix of input that matches target,
// followed by whatever the user has typed beyond that prefix. Once the
// user diverges from target, everything after stays on the off-path side.
func splitOnPath(input, target string) (onPath, offPath string) {
	ir := []rune(input)
	tr := []rune(target)
	n := 0
	for n < len(ir) && n < len(tr) && ir[n] == tr[n] {
		n++
	}
	return string(ir[:n]), string(ir[n:])
}
