package terminal

import (
	"github.com/josephburnett/nixy-go/pkg/process"
)

const (
	// Fixed layout constants
	boxBorders    = 3 // top border + bottom border + prompt line
	hintLine      = 1 // always reserved, even when empty
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

func (t *T) SetDialog(lines []string) {
	t.State.Dialog = append(t.State.Dialog, lines...)
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

	hintStr := ""
	if t.State.Hint != nil {
		hintStr = t.State.Hint.Error()
	}

	// Dialog fills remaining space above the hint + terminal box
	dialogSpace := t.ScreenHeight - termBoxHeight - hintLine - keyboardLines
	if dialogSpace < 0 {
		dialogSpace = 0
	}

	// Show the last dialogSpace lines of accumulated dialog
	var dialogToShow []string
	if dialogSpace > 0 && len(t.State.Dialog) > 0 {
		start := len(t.State.Dialog) - dialogSpace
		if start < 0 {
			start = 0
		}
		dialogToShow = t.State.Dialog[start:]
	}

	displayLines := ReflowLines(t.State.Lines, contentWidth, termContentHeight)
	prompt := "> " + t.State.Line

	f := Frame{
		DisplayLines: displayLines,
		Prompt:       prompt,
		Dialog:       dialogToShow,
		DialogSpace:  dialogSpace,
		Hint:         hintStr,
		ValidKeys:    t.State.ValidKeys,
		HintKey:      t.State.HintKey,
		Width:        contentWidth,
		Height:       termContentHeight,
	}
	return t.renderer.Render(f)
}
