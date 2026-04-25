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

// PromptInfo describes a shell prompt's components so the renderer can
// color the host (and only the host) per-host. When Raw is set it's used
// verbatim — used for special prompts like "login: " that don't follow
// the user@host:path shape.
type PromptInfo struct {
	User string // e.g. "alice"
	Host string // e.g. "nixy"
	Path string // e.g. "/home/nixy"
	Raw  string // when set, used as-is and User/Host/Path are ignored
}

// IsZero reports whether this PromptInfo has no content (used by
// HistoryLine to mark output rows vs. entered commands).
func (p PromptInfo) IsZero() bool {
	return p.User == "" && p.Host == "" && p.Path == "" && p.Raw == ""
}

// HistoryLine is a single line in the terminal scrollback. When Prompt is
// non-zero, the line represents an entered command and the prompt is
// rendered in its colors; otherwise it's command output.
type HistoryLine struct {
	Prompt PromptInfo
	Input  string
}

// State holds the content state of the terminal, independent of rendering.
type State struct {
	Line         string
	Lines        []HistoryLine
	Prompt       PromptInfo // structured: user, host, path (or Raw for special prompts)
	PromptTarget string     // full planned command, used to highlight on-path typing
	Notice       string     // shown above the box (errors, Ctrl+C confirmations)
	Thought      string     // natural-language hint shown below the terminal
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
				s.Lines = append(s.Lines, HistoryLine{Prompt: s.Prompt, Input: s.Line})
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

// promptPrefixText returns the prompt as a single string, used for
// width/truncation arithmetic. The renderer still emits the structured
// PromptInfo as multi-style segments — this is just the length-equivalent.
func (p PromptInfo) promptPrefixText() string {
	if p.Raw != "" {
		return p.Raw
	}
	if p.IsZero() {
		return "> "
	}
	return p.User + "@" + p.Host + ":" + p.Path + "> "
}
