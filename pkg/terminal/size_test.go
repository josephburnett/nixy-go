package terminal

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/josephburnett/nixy-go/pkg/process"
)

// sizeStressTerm returns a terminal with every region overfull: long
// dialog, deep path, long typed input, long history and a notice.
func sizeStressTerm(r Renderer) *T {
	t := New(r)
	t.State.Prompt = PromptInfo{User: "abcdefgh", Host: "server", Path: "/home/nixy/projects/very/deep/path"}
	t.State.PromptTarget = "cat readme.txt | grep something"
	t.State.Line = "cat readme.txt | grep somethingelse entirely"
	for i := 0; i < 40; i++ {
		t.State.Lines = append(t.State.Lines,
			HistoryLine{Prompt: t.State.Prompt, Input: strings.Repeat("x", i*3)},
			HistoryLine{Input: strings.Repeat("output ", i)})
	}
	t.SetDialog([]string{strings.Repeat("Nixy talks a lot `ls -la` ", 10)})
	t.SetDialog([]string{"`averyveryverylongcommandthatcannotwrap`"})
	t.Notify(strings.Repeat("notice ", 20))
	t.SetKeyboard([]process.Datum{process.Chars("a"), process.TermEnter}, process.TermEnter)
	return t
}

// TestRenderFitsScreenAtAllSizes: at every screen size the terminal accepts,
// no rendered line is wider than the screen and the frame height is fixed.
// A line wider than the screen wraps in a real terminal (shifting the whole
// layout) and is clipped in the web build (hiding keys).
func TestRenderFitsScreenAtAllSizes(t *testing.T) {
	for _, r := range []struct {
		name  string
		r     Renderer
		strip func(string) string
	}{
		{"ANSI", NewANSI(), stripANSI},
		{"HTML", NewHTML(), func(s string) string {
			s = strings.TrimPrefix(s, "<pre>")
			s = strings.TrimSuffix(s, "</pre>")
			return stripHTML(s)
		}},
	} {
		term := sizeStressTerm(r.r)
		var lineCount = -1
		for w := 22; w <= 130; w++ {
			for _, h := range []int{15, 16, 24, 31, 50} {
				term.Resize(w, h)
				out := strings.TrimSuffix(r.strip(term.Render()), "\n")
				lines := strings.Split(out, "\n")
				for i, l := range lines {
					if n := utf8.RuneCountInString(l); n > w {
						t.Fatalf("%s %dx%d: line %d is %d wide: %q", r.name, w, h, i, n, l)
					}
				}
				if lineCount == -1 {
					lineCount = len(lines) - h
				}
				if len(lines)-h != lineCount {
					t.Fatalf("%s %dx%d: %d lines, want screen height %+d", r.name, w, h, len(lines), lineCount)
				}
				if len(lines) > h {
					t.Fatalf("%s %dx%d: %d lines overflow the screen", r.name, w, h, len(lines))
				}
			}
		}
	}
}

func TestSpecialRowLabelsShrinkToFit(t *testing.T) {
	for _, c := range []struct {
		width int
		want  string
	}{
		{80, "[space]"},
		{39, "[space]"},
		{38, "[spc]"}, // a phone at 16px is ~35 columns
		{33, "[spc]"},
		{32, "spc"},
		{22, "spc"},
	} {
		if got := specialRowLabels(c.width)[0]; got != c.want {
			t.Errorf("width %d: space label %q, want %q", c.width, got, c.want)
		}
	}
}

// TestResizeBelowMinimumClamps: sizes below the minimum (including zero
// and negative, which the web build can send while the page lays out)
// render at the minimum instead of panicking.
func TestResizeBelowMinimumClamps(t *testing.T) {
	term := sizeStressTerm(NewANSI())
	for _, wh := range [][2]int{{0, 0}, {-5, -5}, {1, 100}, {100, 1}} {
		term.Resize(wh[0], wh[1])
		_ = term.Render()
	}
}
