package terminal

import (
	"github.com/josephburnett/nixy-go/pkg/process"
)

// T is the terminal controller. It composes content state with a renderer.
type T struct {
	State    State
	Width    int
	Height   int
	renderer Renderer
}

func New(r Renderer) *T {
	return &T{
		Width:    55,
		Height:   20,
		renderer: r,
	}
}

func (t *T) Resize(w, h int) {
	t.Width = w
	t.Height = h
}

func (t *T) Write(in process.Data) error {
	return t.State.Write(in)
}

func (t *T) Hint(err error) {
	t.State.Hint = err
}

func (t *T) SetDialog(lines []string) {
	t.State.Dialog = lines
}

func (t *T) SetKeyboard(valid []process.Datum, hint process.Datum) {
	t.State.ValidKeys = valid
	t.State.HintKey = hint
}

func (t *T) Render() string {
	displayLines := ReflowLines(t.State.Lines, t.Width, t.Height)
	prompt := "> " + t.State.Line

	hint := ""
	if t.State.Hint != nil {
		hint = t.State.Hint.Error()
	}

	f := Frame{
		DisplayLines: displayLines,
		Prompt:       prompt,
		Dialog:       t.State.Dialog,
		Hint:         hint,
		ValidKeys:    t.State.ValidKeys,
		HintKey:      t.State.HintKey,
		Width:        t.Width,
		Height:       t.Height,
	}
	t.State.Dialog = nil // clear after render
	return t.renderer.Render(f)
}
