package terminal

import (
	"regexp"
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
)

var htmlTag = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	s = htmlTag.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	return s
}

// loadedFrame returns a populated terminal exercising every visible feature.
func loadedTerm(r Renderer) *T {
	t := New(r)
	t.Resize(80, 24)
	t.State.Prompt = "user@nixy:/home/nixy"
	t.State.PromptTarget = "pwd"
	// History: one prompted command, one output line
	t.State.Lines = []HistoryLine{
		{Prefix: "user@laptop:/> ", Input: "ssh nixy"},
		{Input: "Last login: Sat Jan 1"},
	}
	// Active typed input — partial on-path with one off-path char
	t.State.Line = "px"
	t.State.HintKey = process.TermBackspace
	t.State.ValidKeys = []process.Datum{process.TermBackspace}
	// Two dialog batches; second has a backtick command span
	t.SetDialog([]string{"first batch line"})
	t.SetDialog([]string{"second batch with `pwd` highlight"})
	t.State.Hint = errInvalidParity("a notice")
	t.State.Thought = "I need to print the current working directory"
	return t
}

type errInvalidParity string

func (e errInvalidParity) Error() string { return string(e) }

// TestRenderersProduceParallelStructure feeds an identical Frame to ANSI
// and HTML and asserts the visible text is structurally equivalent.
// This is the load-bearing invariant guarding renderer DRY-ing.
func TestRenderersProduceParallelStructure(t *testing.T) {
	ansiTerm := loadedTerm(NewANSI())
	htmlTerm := loadedTerm(NewHTML())

	ansiOut := stripANSI(ansiTerm.Render())
	htmlOut := stripHTML(htmlTerm.Render())

	ansiLines := strings.Split(strings.TrimRight(ansiOut, "\n"), "\n")
	htmlLines := strings.Split(strings.TrimRight(htmlOut, "\n"), "\n")

	if len(ansiLines) != len(htmlLines) {
		t.Fatalf("line count mismatch: ANSI=%d HTML=%d\nANSI:\n%s\nHTML:\n%s",
			len(ansiLines), len(htmlLines), ansiOut, htmlOut)
	}

	// Every line should match exactly after stripping styling.
	for i := range ansiLines {
		a := strings.TrimRight(ansiLines[i], " ")
		h := strings.TrimRight(htmlLines[i], " ")
		if a != h {
			t.Errorf("line %d differs:\nANSI: %q\nHTML: %q", i, a, h)
		}
	}
}

// TestRenderersBothShowKeyContent sanity-checks that both surfaces include
// the headline pieces of a fully loaded frame.
func TestRenderersBothShowKeyContent(t *testing.T) {
	expectations := []string{
		"first batch line",
		"second batch with",
		"pwd",                      // the highlighted command
		"a notice",                 // hint slot
		"current working directory", // thought
		"user@nixy:/home/nixy>",    // active prompt
		"ssh nixy",                 // history input
		"Last login: Sat Jan 1",    // history output
		"q", "w", "e",              // keyboard letters
		"[space]", "[enter]", "[bksp]", // keyboard specials
	}

	for _, r := range []struct {
		name string
		out  string
	}{
		{"ANSI", stripANSI(loadedTerm(NewANSI()).Render())},
		{"HTML", stripHTML(loadedTerm(NewHTML()).Render())},
	} {
		for _, want := range expectations {
			if !strings.Contains(r.out, want) {
				t.Errorf("%s output missing %q", r.name, want)
			}
		}
	}
}
