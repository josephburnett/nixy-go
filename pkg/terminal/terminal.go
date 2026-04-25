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

	// Hint slot above the box is reserved for errors / Ctrl+C confirmations.
	hintStr := ""
	if t.State.Hint != nil {
		hintStr = t.State.Hint.Error()
	}

	// Thought slot below the box.
	thoughtStr := ""
	if t.State.Thought != "" {
		thoughtStr = "(" + t.State.Thought + "...)"
	}

	// Dialog fills remaining space above the hint + terminal box + thought
	dialogSpace := t.ScreenHeight - termBoxHeight - hintLine - thoughtLine - keyboardLines
	if dialogSpace < 0 {
		dialogSpace = 0
	}

	// Show the last dialogSpace lines of accumulated dialog
	var dialogToShow []DialogLine
	if dialogSpace > 0 && len(t.State.Dialog) > 0 {
		start := len(t.State.Dialog) - dialogSpace
		if start < 0 {
			start = 0
		}
		dialogToShow = t.State.Dialog[start:]
	}

	displayLines := ReflowLines(t.State.Lines, contentWidth, termContentHeight)

	f := Frame{
		DisplayLines: displayLines,
		PromptPrefix: t.State.promptPrefix(),
		PromptInput:  t.State.Line,
		Dialog:       dialogToShow,
		DialogSpace:  dialogSpace,
		Hint:         hintStr,
		Thought:      thoughtStr,
		ValidKeys:    t.State.ValidKeys,
		HintKey:      t.State.HintKey,
		Width:        contentWidth,
		Height:       termContentHeight,
	}
	return t.renderer.Render(f)
}
