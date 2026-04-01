package terminal

import (
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
)

func TestWriteChars(t *testing.T) {
	term := New()
	term.Write(process.Data{process.Chars("hello")})
	if term.line != "hello" {
		t.Fatalf("expected 'hello', got %q", term.line)
	}
}

func TestWriteCharsWithNewlines(t *testing.T) {
	term := New()
	term.Write(process.Data{process.Chars("bin\netc\nhome\n")})
	if len(term.lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(term.lines), term.lines)
	}
	if term.lines[0] != "bin" || term.lines[1] != "etc" || term.lines[2] != "home" {
		t.Fatalf("expected [bin etc home], got %v", term.lines)
	}
	if term.line != "" {
		t.Fatalf("expected empty current line, got %q", term.line)
	}
}

func TestWriteCharsPartialNewline(t *testing.T) {
	term := New()
	term.Write(process.Data{process.Chars("hello\nworld")})
	if len(term.lines) != 1 || term.lines[0] != "hello" {
		t.Fatalf("expected lines=['hello'], got %v", term.lines)
	}
	if term.line != "world" {
		t.Fatalf("expected current line 'world', got %q", term.line)
	}
}

func TestWriteEnter(t *testing.T) {
	term := New()
	term.Write(process.Data{process.Chars("cmd")})
	term.Write(process.Data{process.TermEnter})
	if len(term.lines) != 1 || term.lines[0] != "cmd" {
		t.Fatalf("expected lines=['cmd'], got %v", term.lines)
	}
	if term.line != "" {
		t.Fatalf("expected empty line after enter, got %q", term.line)
	}
}

func TestWriteBackspace(t *testing.T) {
	term := New()
	term.Write(process.Data{process.Chars("abc")})
	term.Write(process.Data{process.TermBackspace})
	if term.line != "ab" {
		t.Fatalf("expected 'ab', got %q", term.line)
	}
}

func TestWriteBackspaceEmpty(t *testing.T) {
	term := New()
	term.Write(process.Data{process.TermBackspace})
	if term.line != "" {
		t.Fatalf("expected empty, got %q", term.line)
	}
}

func TestWriteClear(t *testing.T) {
	term := New()
	term.Write(process.Data{process.Chars("hello")})
	term.Write(process.Data{process.TermEnter})
	term.Write(process.Data{process.Chars("world")})
	term.Write(process.Data{process.TermClear})
	if len(term.lines) != 0 {
		t.Fatalf("expected no lines after clear, got %v", term.lines)
	}
	if term.line != "" {
		t.Fatalf("expected empty line after clear, got %q", term.line)
	}
}

func TestWriteMultipleDatums(t *testing.T) {
	term := New()
	term.Write(process.Data{
		process.Chars("h"),
		process.Chars("i"),
		process.TermEnter,
		process.Chars("bye"),
	})
	if len(term.lines) != 1 || term.lines[0] != "hi" {
		t.Fatalf("expected lines=['hi'], got %v", term.lines)
	}
	if term.line != "bye" {
		t.Fatalf("expected 'bye', got %q", term.line)
	}
}

func TestRenderEmptyTerminal(t *testing.T) {
	term := New()
	out := term.Render()
	if !strings.Contains(out, "┌") || !strings.Contains(out, "└") {
		t.Fatal("expected box drawing characters in render")
	}
	if !strings.Contains(out, "> ") {
		t.Fatal("expected prompt in render")
	}
}

func TestRenderWithContent(t *testing.T) {
	term := New()
	term.Write(process.Data{process.Chars("output line\n")})
	term.Write(process.Data{process.Chars("typing")})
	out := term.Render()
	if !strings.Contains(out, "output line") {
		t.Fatalf("expected 'output line' in render, got:\n%s", out)
	}
	if !strings.Contains(out, "> typing") {
		t.Fatalf("expected '> typing' in render, got:\n%s", out)
	}
}

func TestRenderDialog(t *testing.T) {
	term := New()
	term.SetDialog([]string{"Nixy: Hello!", "Nixy: Welcome."})
	out := term.Render()
	if !strings.Contains(out, "Nixy: Hello!") {
		t.Fatalf("expected dialog in render, got:\n%s", out)
	}
	if !strings.Contains(out, "Nixy: Welcome.") {
		t.Fatalf("expected second dialog line in render, got:\n%s", out)
	}
	// Dialog should be cleared after render
	out2 := term.Render()
	if strings.Contains(out2, "Nixy: Hello!") {
		t.Fatal("dialog should be cleared after first render")
	}
}

func TestRenderHintNil(t *testing.T) {
	term := New()
	term.Hint(nil)
	out := term.Render()
	if strings.Contains(out, "invalid") {
		t.Fatal("should not show hint when nil")
	}
}

func TestRenderHintWithError(t *testing.T) {
	term := New()
	term.Hint(errInvalid("invalid input"))
	out := term.Render()
	if !strings.Contains(out, "invalid input") {
		t.Fatalf("expected hint in render, got:\n%s", out)
	}
}

type errInvalid string

func (e errInvalid) Error() string { return string(e) }

func TestRenderScrolling(t *testing.T) {
	term := New()
	// Add more lines than the viewport (20 lines)
	for i := 0; i < 30; i++ {
		term.Write(process.Data{process.Chars("line\n")})
	}
	out := term.Render()
	// Should show the terminal without error
	if !strings.Contains(out, "line") {
		t.Fatal("expected lines in scrolled render")
	}
	// Count how many "│line" appear — should be at most 20
	count := strings.Count(out, "│line")
	if count > 20 {
		t.Fatalf("expected at most 20 visible lines, got %d", count)
	}
}

func TestRenderLineTruncation(t *testing.T) {
	term := New()
	long := strings.Repeat("x", 100)
	term.Write(process.Data{process.Chars(long + "\n")})
	out := term.Render()
	// Line should be truncated to fit within the box
	// Each content line is at most term.x chars
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "│") && strings.HasSuffix(line, "│") {
			content := line[len("│") : len(line)-len("│")]
			if len(content) > term.x {
				t.Fatalf("line exceeds terminal width: %d > %d", len(content), term.x)
			}
		}
	}
}
