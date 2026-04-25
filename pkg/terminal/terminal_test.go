package terminal

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/josephburnett/nixy-go/pkg/process"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

func TestWriteChars(t *testing.T) {
	term := New(NewANSI())
	term.Write(process.Data{process.Chars("hello")})
	if term.State.Line != "hello" {
		t.Fatalf("expected 'hello', got %q", term.State.Line)
	}
}

func TestWriteCharsWithNewlines(t *testing.T) {
	term := New(NewANSI())
	term.Write(process.Data{process.Chars("bin\netc\nhome\n")})
	if len(term.State.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(term.State.Lines), term.State.Lines)
	}
	if term.State.Lines[0].Input != "bin" || term.State.Lines[1].Input != "etc" || term.State.Lines[2].Input != "home" {
		t.Fatalf("expected [bin etc home], got %v", term.State.Lines)
	}
	if term.State.Line != "" {
		t.Fatalf("expected empty current line, got %q", term.State.Line)
	}
}

func TestWriteCharsPartialNewline(t *testing.T) {
	term := New(NewANSI())
	term.Write(process.Data{process.Chars("hello\nworld")})
	if len(term.State.Lines) != 1 || term.State.Lines[0].Input != "hello" {
		t.Fatalf("expected lines=['hello'], got %v", term.State.Lines)
	}
	if term.State.Line != "world" {
		t.Fatalf("expected current line 'world', got %q", term.State.Line)
	}
}

func TestWriteEnter(t *testing.T) {
	term := New(NewANSI())
	term.Write(process.Data{process.Chars("cmd")})
	term.Write(process.Data{process.TermEnter})
	if len(term.State.Lines) != 1 || term.State.Lines[0].Prefix != "> " || term.State.Lines[0].Input != "cmd" {
		t.Fatalf("expected lines=[{Prefix:'> ', Input:'cmd'}], got %v", term.State.Lines)
	}
	if term.State.Line != "" {
		t.Fatalf("expected empty line after enter, got %q", term.State.Line)
	}
}

func TestWriteBackspace(t *testing.T) {
	term := New(NewANSI())
	term.Write(process.Data{process.Chars("abc")})
	term.Write(process.Data{process.TermBackspace})
	if term.State.Line != "ab" {
		t.Fatalf("expected 'ab', got %q", term.State.Line)
	}
}

func TestWriteBackspaceEmpty(t *testing.T) {
	term := New(NewANSI())
	term.Write(process.Data{process.TermBackspace})
	if term.State.Line != "" {
		t.Fatalf("expected empty, got %q", term.State.Line)
	}
}

func TestWriteClear(t *testing.T) {
	term := New(NewANSI())
	term.Write(process.Data{process.Chars("hello")})
	term.Write(process.Data{process.TermEnter})
	term.Write(process.Data{process.Chars("world")})
	term.Write(process.Data{process.TermClear})
	if len(term.State.Lines) != 0 {
		t.Fatalf("expected no lines after clear, got %v", term.State.Lines)
	}
	if term.State.Line != "" {
		t.Fatalf("expected empty line after clear, got %q", term.State.Line)
	}
}

func TestWriteMultipleDatums(t *testing.T) {
	term := New(NewANSI())
	term.Write(process.Data{
		process.Chars("h"),
		process.Chars("i"),
		process.TermEnter,
		process.Chars("bye"),
	})
	if len(term.State.Lines) != 1 || term.State.Lines[0].Prefix != "> " || term.State.Lines[0].Input != "hi" {
		t.Fatalf("expected lines=[{Prefix:'> ', Input:'hi'}], got %v", term.State.Lines)
	}
	if term.State.Line != "bye" {
		t.Fatalf("expected 'bye', got %q", term.State.Line)
	}
}

func TestSplitOnPath(t *testing.T) {
	cases := []struct {
		input, target, on, off string
	}{
		{"", "pwd", "", ""},
		{"p", "pwd", "p", ""},
		{"pw", "pwd", "pw", ""},
		{"pwd", "pwd", "pwd", ""},
		{"px", "pwd", "p", "x"},
		{"hello", "", "", "hello"},
	}
	for _, c := range cases {
		on, off := splitOnPath(c.input, c.target)
		if on != c.on || off != c.off {
			t.Errorf("splitOnPath(%q, %q) = (%q, %q), want (%q, %q)",
				c.input, c.target, on, off, c.on, c.off)
		}
	}
}

func TestRenderEmptyTerminal(t *testing.T) {
	term := New(NewANSI())
	out := term.Render()
	if !strings.Contains(out, "┌") || !strings.Contains(out, "└") {
		t.Fatal("expected box drawing characters in render")
	}
	if !strings.Contains(out, "> ") {
		t.Fatal("expected prompt in render")
	}
}

func TestRenderWithContent(t *testing.T) {
	term := New(NewANSI())
	term.Write(process.Data{process.Chars("output line\n")})
	term.Write(process.Data{process.Chars("typing")})
	out := term.Render()
	if !strings.Contains(out, "output line") {
		t.Fatalf("expected 'output line' in render, got:\n%s", out)
	}
	if !strings.Contains(stripANSI(out), "> typing") {
		t.Fatalf("expected '> typing' in render, got:\n%s", out)
	}
}

func TestRenderDialog(t *testing.T) {
	term := New(NewANSI())
	term.SetDialog([]string{"Nixy: Hello!", "Nixy: Welcome."})
	out := term.Render()
	if !strings.Contains(out, "Nixy: Hello!") {
		t.Fatalf("expected dialog in render, got:\n%s", out)
	}
	if !strings.Contains(out, "Nixy: Welcome.") {
		t.Fatalf("expected second dialog line in render, got:\n%s", out)
	}
	// Dialog persists across renders
	out2 := term.Render()
	if !strings.Contains(out2, "Nixy: Hello!") {
		t.Fatal("dialog should persist across renders")
	}
}

func TestRenderNoticeEmpty(t *testing.T) {
	term := New(NewANSI())
	term.Notify("")
	out := term.Render()
	if strings.Contains(out, "invalid") {
		t.Fatal("should not show notice when empty")
	}
}

func TestRenderNoticeWithMessage(t *testing.T) {
	term := New(NewANSI())
	term.Notify("invalid input")
	out := term.Render()
	if !strings.Contains(out, "invalid input") {
		t.Fatalf("expected notice in render, got:\n%s", out)
	}
}

func TestRenderDialogAccumulates(t *testing.T) {
	term := New(NewANSI())
	term.SetDialog([]string{"First message"})
	term.Render() // should NOT clear
	term.SetDialog([]string{"Second message"})
	out := term.Render()
	if !strings.Contains(out, "First message") {
		t.Fatal("expected first message to persist")
	}
	if !strings.Contains(out, "Second message") {
		t.Fatal("expected second message to appear")
	}
	// First should appear before second
	idx1 := strings.Index(out, "First message")
	idx2 := strings.Index(out, "Second message")
	if idx1 > idx2 {
		t.Fatal("older dialog should appear above newer dialog")
	}
}

func TestRenderDialogBatchesGetDistinctColors(t *testing.T) {
	term := New(NewANSI())
	term.SetDialog([]string{"first"})
	term.SetDialog([]string{"second"})
	out := term.Render()
	// Each batch should be wrapped in a different color code.
	firstIdx := strings.Index(out, "first")
	secondIdx := strings.Index(out, "second")
	if firstIdx < 0 || secondIdx < 0 {
		t.Fatal("expected both messages in render")
	}
	// Find the ANSI color escape immediately preceding each text.
	firstPrefix := out[:firstIdx]
	secondPrefix := out[:secondIdx]
	firstColor := firstPrefix[strings.LastIndex(firstPrefix, "\033["):]
	secondColor := secondPrefix[strings.LastIndex(secondPrefix, "\033["):]
	if firstColor == secondColor {
		t.Fatalf("successive dialog batches should have different colors, both got %q", firstColor)
	}
}

func TestRenderDialogAboveBox(t *testing.T) {
	term := New(NewANSI())
	term.SetDialog([]string{"Nixy: Hello!"})
	out := term.Render()
	dialogIdx := strings.Index(out, "Nixy: Hello!")
	boxIdx := strings.Index(out, "┌")
	if dialogIdx < 0 || boxIdx < 0 {
		t.Fatalf("expected dialog and box in output, got:\n%s", out)
	}
	if dialogIdx > boxIdx {
		t.Fatal("dialog should appear above the terminal box")
	}
}

func TestRenderNoticeAboveBox(t *testing.T) {
	term := New(NewANSI())
	term.Notify("oops")
	out := term.Render()
	noticeIdx := strings.Index(out, "oops")
	boxIdx := strings.Index(out, "┌")
	if noticeIdx < 0 || boxIdx < 0 {
		t.Fatalf("expected notice and box in output, got:\n%s", out)
	}
	if noticeIdx > boxIdx {
		t.Fatal("notice should appear above the terminal box")
	}
}

func TestRenderScrolling(t *testing.T) {
	term := New(NewANSI())
	// Default screen 30 tall, terminal box = 50% = 15, content = 15 - 3 = 12
	for i := 0; i < 30; i++ {
		term.Write(process.Data{process.Chars("line\n")})
	}
	out := term.Render()
	if !strings.Contains(out, "line") {
		t.Fatal("expected lines in scrolled render")
	}
	// Terminal content is 50% of screen minus borders/prompt
	termContentHeight := term.ScreenHeight/2 - boxBorders
	count := strings.Count(out, "│line")
	if count > termContentHeight {
		t.Fatalf("expected at most %d visible lines, got %d", termContentHeight, count)
	}
}

// TestRenderLayoutHeightInvariant pins the "stable layout" property: for a
// given ScreenHeight, the rendered line count is constant regardless of
// frame content (dialog density, hint, thought, typed input). The terminal
// box must not bump around when dialog accumulates or hints appear.
func TestRenderLayoutHeightInvariant(t *testing.T) {
	const w, h = 80, 24
	configs := []struct {
		name string
		fill func(*T)
	}{
		{"empty", func(*T) {}},
		{"dialog heavy", func(t *T) {
			for i := 0; i < 10; i++ {
				t.SetDialog([]string{"line " + string(rune('a'+i))})
			}
		}},
		{"with notice and thought", func(t *T) {
			t.Notify("oops")
			t.SetThought("doing something")
		}},
		{"with input and history", func(t *T) {
			t.Write(process.Data{process.Chars("output\n")})
			t.Write(process.Data{process.Chars("typing")})
		}},
	}
	var baseline int
	for i, c := range configs {
		t.Run(c.name, func(t *testing.T) {
			term := New(NewANSI())
			term.Resize(w, h)
			c.fill(term)
			out := term.Render()
			gotLines := strings.Count(out, "\n")
			if i == 0 {
				baseline = gotLines
				return
			}
			if gotLines != baseline {
				t.Fatalf("layout shifted: %q rendered %d lines, baseline %d",
					c.name, gotLines, baseline)
			}
		})
	}
}

// TestRenderPromptCursorAndPath pins the cursor + on/off-path coloring so
// the upcoming layout refactor doesn't lose them.
func TestRenderPromptCursorAndPath(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*T)
		wantCursor  string // "green" or "white"
		wantOnGreen string // substring expected to be wrapped in green
		wantOffWhite string
	}{
		{
			name: "empty input white cursor",
			setup: func(t *T) {
				t.State.HintKey = nil
			},
			wantCursor: "white",
		},
		{
			name: "on path cursor green when hint is char",
			setup: func(t *T) {
				t.State.Line = "p"
				t.State.PromptTarget = "pwd"
				t.State.HintKey = process.Chars("w")
			},
			wantCursor:  "green",
			wantOnGreen: "p",
		},
		{
			name: "off path turns later input white",
			setup: func(t *T) {
				t.State.Line = "px"
				t.State.PromptTarget = "pwd"
				t.State.HintKey = process.TermBackspace
			},
			wantCursor:   "white",
			wantOnGreen:  "p",
			wantOffWhite: "x",
		},
		{
			name: "no hint cursor white",
			setup: func(t *T) {
				t.State.Line = "anything"
				t.State.HintKey = nil
			},
			wantCursor:   "white",
			wantOffWhite: "anything",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			term := New(NewANSI())
			term.Resize(80, 24)
			tc.setup(term)
			out := term.Render()

			// Find the cursor block and inspect the color escape preceding it.
			cursorIdx := strings.Index(out, "█")
			if cursorIdx < 0 {
				t.Fatalf("expected cursor in output, got:\n%s", out)
			}
			before := out[:cursorIdx]
			lastEsc := strings.LastIndex(before, "\033[")
			if lastEsc < 0 {
				t.Fatalf("expected color escape before cursor, got:\n%s", out)
			}
			cursorPrefix := before[lastEsc:]
			switch tc.wantCursor {
			case "green":
				if !strings.HasPrefix(cursorPrefix, colorGreen) {
					t.Errorf("expected green cursor, got escape %q", cursorPrefix)
				}
			case "white":
				if !strings.HasPrefix(cursorPrefix, colorWhite) {
					t.Errorf("expected white cursor, got escape %q", cursorPrefix)
				}
			}

			if tc.wantOnGreen != "" {
				want := colorGreen + tc.wantOnGreen + colorReset
				if !strings.Contains(out, want) {
					t.Errorf("expected on-path span %q in output", want)
				}
			}
			if tc.wantOffWhite != "" {
				want := colorWhite + tc.wantOffWhite + colorReset
				if !strings.Contains(out, want) {
					t.Errorf("expected off-path span %q in output", want)
				}
			}
		})
	}
}

// TestRenderActivePromptPrefixBlue pins the active prompt prefix color.
func TestRenderActivePromptPrefixBlue(t *testing.T) {
	term := New(NewANSI())
	term.Resize(80, 24)
	term.State.Prompt = "user@nixy:/"
	out := term.Render()
	want := colorPrompt + "user@nixy:/> " + colorReset
	if !strings.Contains(out, want) {
		t.Fatalf("expected blue active prompt %q in output", want)
	}
}

// TestRenderHistoryPromptBlue pins the historical prompt prefix color so
// old commands stay distinguishable from output.
func TestRenderHistoryPromptBlue(t *testing.T) {
	term := New(NewANSI())
	term.Resize(80, 24)
	term.State.Prompt = "user@nixy:/"
	term.Write(process.Data{process.Chars("ls")})
	term.Write(process.Data{process.TermEnter})
	out := term.Render()
	// History line should have the prefix in blue, then "ls" in plain.
	want := colorPrompt + "user@nixy:/> " + colorReset
	if !strings.Contains(out, want) {
		t.Fatalf("expected blue history prompt prefix in output, got:\n%s", out)
	}
}

// TestRenderThoughtBelowBox pins the thought line position between the
// terminal box and the keyboard.
func TestRenderThoughtBelowBox(t *testing.T) {
	term := New(NewANSI())
	term.Resize(80, 24)
	term.SetThought("here is a thought")
	out := term.Render()
	thoughtIdx := strings.Index(out, "here is a thought")
	boxBottomIdx := strings.Index(out, "└")
	keyboardIdx := strings.Index(out, "[space]")
	if thoughtIdx < 0 || boxBottomIdx < 0 || keyboardIdx < 0 {
		t.Fatalf("expected thought, box bottom, and keyboard in output, got:\n%s", out)
	}
	if !(boxBottomIdx < thoughtIdx && thoughtIdx < keyboardIdx) {
		t.Fatalf("thought should sit between box bottom (%d) and keyboard (%d), got at %d",
			boxBottomIdx, keyboardIdx, thoughtIdx)
	}
}

func TestRenderLineTruncation(t *testing.T) {
	term := New(NewANSI())
	long := strings.Repeat("x", 100)
	term.Write(process.Data{process.Chars(long + "\n")})
	out := term.Render()
	// Line should be truncated to fit within the box
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "│") && strings.HasSuffix(line, "│") {
			content := stripANSI(line[len("│") : len(line)-len("│")])
			contentWidth := term.ScreenWidth - 2
			if utf8.RuneCountInString(content) > contentWidth {
				t.Fatalf("line exceeds terminal width: %d > %d", utf8.RuneCountInString(content), contentWidth)
			}
		}
	}
}
