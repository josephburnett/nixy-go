package terminal

import (
	"fmt"
	"unicode/utf8"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// State holds the content state of the terminal, independent of rendering.
type State struct {
	Line      string
	Lines     []string
	Prompt    string // e.g. "user@nixy", updated by session on hostname changes
	Hint      error
	Dialog    []string
	ValidKeys []process.Datum
	HintKey   process.Datum
}

func (s *State) Write(in process.Data) error {
	for _, d := range in {
		switch d := d.(type) {
		case process.Chars:
			for _, c := range string(d) {
				if c == '\n' {
					s.Lines = append(s.Lines, s.Line)
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
				s.Lines = []string{}
			case process.TermEnter:
				s.Lines = append(s.Lines, s.promptPrefix()+s.Line)
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
