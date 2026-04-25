package terminal

import "github.com/josephburnett/nixy-go/pkg/process"

// Frame holds all data needed to render one screen.
type Frame struct {
	DisplayLines     []string
	PromptPrefix     string // colored portion (e.g. "user@nixy:/home/nixy> ")
	PromptInputOn    string // typed input that matches the planner's path
	PromptInputOff   string // typed input that has gone off-path
	Dialog           []DialogLine
	DialogSpace      int // total lines allocated for dialog (pad with blank lines)
	Hint             string
	Thought          string // shown on its own line below the terminal box
	ValidKeys        []process.Datum
	HintKey          process.Datum
	Width            int
	Height           int
}

// Renderer produces platform-specific output from a Frame.
type Renderer interface {
	Render(f Frame) string
}
