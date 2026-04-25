package terminal

import (
	"github.com/josephburnett/nixy-go/pkg/process"
)

const (
	// Fixed layout constants
	boxBorders    = 3 // top border + bottom border + prompt line
	hintLine      = 1 // always reserved, even when empty
	thoughtLine   = 1 // always reserved, below the terminal box
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

func (t *T) Hint(err error) {
	t.State.Hint = err
}

func (t *T) SetThought(s string) {
	t.State.Thought = s
}

func (t *T) SetDialog(lines []string) {
	if len(lines) == 0 {
		return
	}
	idx := t.State.NextColorIdx
	for _, l := range lines {
		t.State.Dialog = append(t.State.Dialog, DialogLine{Text: l, ColorIdx: idx})
	}
	t.State.NextColorIdx++
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

	// Hint and thought are single-line slots — truncate to fit so wrapping
	// doesn't push the rest of the layout around.
	hintStr := ""
	if t.State.Hint != nil {
		hintStr = TruncateRunes(t.State.Hint.Error(), contentWidth)
	}
	thoughtStr := ""
	if t.State.Thought != "" {
		thoughtStr = TruncateRunes("("+t.State.Thought+"...)", contentWidth)
	}

	// Dialog fills remaining space above the hint + terminal box + thought
	dialogSpace := t.ScreenHeight - termBoxHeight - hintLine - thoughtLine - keyboardLines
	if dialogSpace < 0 {
		dialogSpace = 0
	}

	// Wrap each accumulated dialog line to the screen width, preserving
	// the batch color across wrapped fragments. Then take the last
	// dialogSpace lines to fit the slot.
	var wrappedDialog []DialogLine
	for _, dl := range t.State.Dialog {
		for _, w := range WrapWords(dl.Text, t.ScreenWidth) {
			wrappedDialog = append(wrappedDialog, DialogLine{Text: w, ColorIdx: dl.ColorIdx})
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
		PromptPrefix:   t.State.promptPrefix(),
		PromptInputOn:  onPath,
		PromptInputOff: offPath,
		CursorOnPath:   cursorOnPath,
		Dialog:         dialogToShow,
		DialogSpace:    dialogSpace,
		Hint:           hintStr,
		Thought:        thoughtStr,
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
