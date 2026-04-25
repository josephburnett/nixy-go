package terminal

import "github.com/josephburnett/nixy-go/pkg/process"

// Frame holds all data needed to render one screen.
type Frame struct {
	DisplayLines   []DisplayLine
	PromptPrefix   string // colored portion (e.g. "user@nixy:/home/nixy> ")
	PromptInputOn  string // typed input that matches the planner's path
	PromptInputOff string // typed input that has gone off-path
	CursorOnPath   bool   // green block cursor when true, white otherwise
	Dialog         []DialogLine
	DialogSpace    int // total lines allocated for dialog (pad with blank lines)
	Status         string // single status line below the box
	StatusIsNotice bool   // true: notice (errors/Ctrl+C); false: thought
	ValidKeys      []process.Datum
	HintKey        process.Datum
	Width          int
	Height         int
}

// Renderer produces platform-specific output from a Frame.
type Renderer interface {
	Render(f Frame) string
}

// Style identifies the visual treatment of a Segment. Renderers map
// Style values to ANSI escape codes or CSS classes.
type Style int

const (
	StyleDefault Style = iota
	StyleBox            // box-drawing borders
	StylePrompt         // bold blue prompt prefix
	StylePromptOff      // white off-path input
	StyleOnPath         // bold green: on-path input, hint key, command markers
	StyleDialog         // dialog batch color (uses BatchIdx)
	StyleDim            // thought (faded grey)
	StyleNotice         // notice line (errors, Ctrl+C) — slightly more prominent
	StyleCursorOn       // green cursor block
	StyleCursorOff      // white cursor block
	StyleKeyValid       // bold white keyboard key
	StyleKeyDim         // dim keyboard key
)

// Segment is one styled span of text within a rendered line.
type Segment struct {
	Text     string
	Style    Style
	BatchIdx int // for StyleDialog: selects palette index
}

// RenderedLine is one output line composed of styled segments. Renderers
// emit a newline after each line.
type RenderedLine []Segment
