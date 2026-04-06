package terminal

import "github.com/josephburnett/nixy-go/pkg/process"

// Frame holds all data needed to render one screen.
type Frame struct {
	DisplayLines []string
	Prompt       string
	Dialog       []string
	Hint         string
	ValidKeys    []process.Datum
	HintKey      process.Datum
	Width        int
	Height       int
}

// Renderer produces platform-specific output from a Frame.
type Renderer interface {
	Render(f Frame) string
}
