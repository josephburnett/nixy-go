package terminal

import (
	"fmt"
	"unicode/utf8"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// DialogLine is one line of dialog with its batch color index.
type DialogLine struct {
	Text     string
	ColorIdx int
}

// HistoryLine is a single line in the terminal scrollback. Prefix (when set)
// is the prompt prefix snapshot at the moment the user pressed Enter — it
// renders in the prompt color so old prompts stay visually distinct from
// command output.
type HistoryLine struct {
	Prefix string // e.g. "user@nixy:/home/nixy> ", or "" for command output
	Input  string // typed command, or output text
}

// State holds the content state of the terminal, independent of rendering.
type State struct {
	Line         string
	Lines        []HistoryLine
	Prompt       string // e.g. "user@nixy:/home/nixy", updated by session
	PromptTarget string // full planned command, used to highlight on-path typing
	Notice       string // shown above the box (errors, Ctrl+C confirmations)
	Thought      string // natural-language hint shown below the terminal
	Dialog       []DialogLine
	NextColorIdx int
	ValidKeys    []process.Datum
	HintKey      process.Datum
}

func (s *State) Write(in process.Data) error {
	for _, d := range in {
		switch d := d.(type) {
		case process.Chars:
			for _, c := range string(d) {
				if c == '\n' {
					s.Lines = append(s.Lines, HistoryLine{Input: s.Line})
					s.Line = ""
				} else {
					s.Line += string(c)
				}
			}
		case process.TermCode:
			switch d {
			case process.TermBackspace:
				if utf8.RuneCountInString(s.Line) > 0 {
					_, size := utf8.DecodeLastRuneInString(s.Line)
					s.Line = s.Line[:len(s.Line)-size]
				}
			case process.TermClear:
				s.Line = ""
				s.Lines = nil
			case process.TermEnter:
				s.Lines = append(s.Lines, HistoryLine{Prefix: s.promptPrefix(), Input: s.Line})
				s.Line = ""
			default:
				return fmt.Errorf("unknown term code: %v", d)
			}
		default:
			return fmt.Errorf("unsupported data type: %T", d)
		}
	}
	return nil
}

func (s *State) promptPrefix() string {
	if s.Prompt != "" {
		return s.Prompt + "> "
	}
	return "> "
}
