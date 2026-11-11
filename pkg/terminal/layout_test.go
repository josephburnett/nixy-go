package terminal

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// stripHTMLPre strips the HTML renderer's markup down to visible text.
func stripHTMLPre(s string) string {
	s = strings.TrimPrefix(s, "<pre>")
	s = strings.TrimSuffix(s, "</pre>")
	return stripHTML(s)
}

var renderers = []struct {
	name  string
	new   func() Renderer
	strip func(string) string
}{
	{"ANSI", func() Renderer { return NewANSI() }, stripANSI},
	{"HTML", func() Renderer { return NewHTML() }, stripHTMLPre},
}

// boxInterior returns the text inside the terminal box, rows joined in
// order. Rows are padded to the box width, so the join reads like the
// terminal's own character grid.
func boxInterior(screen string) string {
	var sb strings.Builder
	for _, l := range strings.Split(screen, "\n") {
		if strings.HasPrefix(l, "│") && strings.HasSuffix(l, "│") {
			sb.WriteString(strings.TrimSuffix(strings.TrimPrefix(l, "│"), "│"))
		}
	}
	return sb.String()
}

// TestLongInputVisibleAtAllWidths: whatever the width, the player sees the
// end of what they're typing and the cursor right after it, the frame keeps
// its height, and no line is wider than the screen. When prompt and input
// fit in the box, all of it is visible.
func TestLongInputVisibleAtAllWidths(t *testing.T) {
	prompt := PromptInfo{User: "player", Host: "nixy", Path: "/home/nixy/projects"}
	inputs := []string{"", "c", "cat /home/nixy/projects/target.txtZ"}
	inputs = append(inputs, strings.Repeat("abcdefghij", 8))
	for _, r := range renderers {
		for _, h := range []int{15, 24} {
			lineCount := -1
			for w := 22; w <= 130; w++ {
				for _, in := range inputs {
					term := New(r.new())
					term.Resize(w, h)
					term.SetPrompt(prompt)
					term.State.Lines = []HistoryLine{{Input: "some output"}, {Prompt: prompt, Input: "ls"}}
					term.State.Line = in
					screen := strings.TrimSuffix(r.strip(term.Render()), "\n")
					lines := strings.Split(screen, "\n")
					where := fmt.Sprintf("%s %dx%d input %q", r.name, w, h, in)
					for i, l := range lines {
						if n := utf8.RuneCountInString(l); n > w {
							t.Fatalf("%s: line %d is %d wide: %q", where, i, n, l)
						}
					}
					if lineCount == -1 {
						lineCount = len(lines)
					}
					if len(lines) != lineCount {
						t.Fatalf("%s: %d lines, want %d", where, len(lines), lineCount)
					}
					grid := boxInterior(screen)
					if strings.Count(grid, "█") != 1 {
						t.Fatalf("%s: want exactly one cursor in the box:\n%s", where, screen)
					}
					// At least the last row's worth of input is always visible.
					tail := []rune(in)
					if keep := w - 3; len(tail) > keep {
						tail = tail[len(tail)-keep:]
					}
					if !strings.Contains(grid, string(tail)+"█") {
						t.Fatalf("%s: end of input not visible before the cursor:\n%s", where, screen)
					}
					full := prompt.promptPrefixText() + in + "█"
					boxRows := h/2 - boxBorders + 1
					if utf8.RuneCountInString(full) <= boxRows*(w-2) && !strings.Contains(grid, full) {
						t.Fatalf("%s: prompt and input fit but aren't all visible:\n%s", where, screen)
					}
				}
			}
		}
	}
}

// TestRenderersAgreeOnLongInput: with a wrapped prompt and wrapped history
// both renderers show the same text at every width.
func TestRenderersAgreeOnLongInput(t *testing.T) {
	fill := func(term *T, w int) {
		term.Resize(w, 20)
		prompt := PromptInfo{User: "player", Host: "server", Path: "/home/nixy/projects/deep"}
		term.SetPrompt(prompt)
		term.State.PromptTarget = "cat /home/nixy/projects/target.txt"
		term.State.Line = "cat /home/nixy/projects/tarzzzzzzzzz"
		term.State.HintKey = process.TermBackspace
		term.State.Lines = []HistoryLine{{Prompt: prompt, Input: "ls /home/nixy/projects/deep"}, {Input: "target.txt"}}
		term.SetDialog([]string{"Use `cat` to read it."})
	}
	for w := 22; w <= 130; w++ {
		a, h := New(NewANSI()), New(NewHTML())
		fill(a, w)
		fill(h, w)
		if as, hs := stripANSI(a.Render()), stripHTMLPre(h.Render()); as != hs {
			t.Fatalf("width %d: renderers differ\nANSI:\n%s\nHTML:\n%s", w, as, hs)
		}
	}
}

// TestPromptWrapKeepsPathColors: when the active line wraps, every on-path
// and off-path rune keeps its style on whichever row it lands.
func TestPromptWrapKeepsPathColors(t *testing.T) {
	prompt := PromptInfo{User: "player", Host: "nixy", Path: "/home/nixy/projects"}
	on := "cat /home/nixy/projects/"
	off := "zzzzzzzzzzzzzzzzzzzz"
	for width := 20; width <= 80; width++ {
		f := Frame{Prompt: prompt, PromptInputOn: on, PromptInputOff: off, Width: width, Height: 6}
		var gotOn, gotOff strings.Builder
		for _, line := range Layout(f) {
			for _, s := range line {
				switch s.Style {
				case StyleOnPath:
					gotOn.WriteString(s.Text)
				case StylePromptOff:
					gotOff.WriteString(s.Text)
				}
			}
		}
		if gotOn.String() != on || gotOff.String() != off {
			t.Fatalf("width %d: on %q off %q, want %q %q", width, gotOn.String(), gotOff.String(), on, off)
		}
	}
}

// TestHistoryKeepsInputWhenPromptOverflows: a command entered at a prompt
// wider than the box still shows what was typed, on the following rows.
func TestHistoryKeepsInputWhenPromptOverflows(t *testing.T) {
	prompt := PromptInfo{User: "player", Host: "nixy", Path: "/home/nixy/projects"}
	for _, r := range renderers {
		term := New(r.new())
		term.Resize(22, 24)
		term.State.Lines = []HistoryLine{{Prompt: prompt, Input: "cat target.txt"}}
		grid := boxInterior(r.strip(term.Render()))
		if want := prompt.promptPrefixText() + "cat target.txt"; !strings.Contains(grid, want) {
			t.Fatalf("%s: history entry %q not visible in box %q", r.name, want, grid)
		}
	}
}

// TestRenderCostBoundedByScreen: rendering does work proportional to what's
// on screen, not to how much history or dialog has piled up. State is
// filled directly, past the retention caps, so this guards the reflow
// itself.
func TestRenderCostBoundedByScreen(t *testing.T) {
	term := New(NewANSI())
	term.Resize(80, 40)
	prompt := PromptInfo{User: "player", Host: "nixy", Path: "/"}
	term.SetPrompt(prompt)
	out := strings.Repeat("output ", 20)
	for i := 0; i < 100000; i++ {
		term.State.Lines = append(term.State.Lines, HistoryLine{Prompt: prompt, Input: "ls"}, HistoryLine{Input: out})
	}
	say := strings.Repeat("Nixy says something `ls` ", 6)
	for i := 0; i < 10000; i++ {
		term.State.Dialog = append(term.State.Dialog, DialogLine{}, DialogLine{Text: say})
	}
	start := time.Now()
	const renders = 100
	for i := 0; i < renders; i++ {
		_ = term.Render()
	}
	if per := time.Since(start) / renders; per > 5*time.Millisecond {
		t.Fatalf("render took %v with a long history; want it bounded by the screen", per)
	}
}

// TestScrollbackIsCapped: the terminal keeps a bounded scrollback, like a
// real terminal, and drops the oldest lines first.
func TestScrollbackIsCapped(t *testing.T) {
	term := New(NewANSI())
	for i := 0; i < maxScrollback+50; i++ {
		term.Write(process.Data{process.Chars(fmt.Sprintf("line %d\n", i))})
	}
	if len(term.State.Lines) != maxScrollback {
		t.Fatalf("kept %d lines, want %d", len(term.State.Lines), maxScrollback)
	}
	if last := term.State.Lines[len(term.State.Lines)-1].Input; last != fmt.Sprintf("line %d", maxScrollback+49) {
		t.Fatalf("newest line is %q", last)
	}
	for i := 0; i < maxDialog+50; i++ {
		term.SetDialog([]string{"a", "b"})
	}
	if n := len(term.State.Dialog); n > maxDialog {
		t.Fatalf("kept %d dialog lines, want at most %d", n, maxDialog)
	}
	if term.State.Dialog[0].Text == "" {
		t.Fatal("trimmed dialog starts with a paragraph separator")
	}
}

// TestRenderEmptyFrameAtMinimum: a terminal with nothing in it renders at
// the minimum size with the box, cursor and keyboard all present.
func TestRenderEmptyFrameAtMinimum(t *testing.T) {
	for _, r := range renderers {
		term := New(r.new())
		term.Resize(0, 0)
		screen := r.strip(term.Render())
		for _, want := range []string{"┌", "└", "█", "spc"} {
			if !strings.Contains(screen, want) {
				t.Fatalf("%s: empty frame missing %q:\n%s", r.name, want, screen)
			}
		}
	}
}
