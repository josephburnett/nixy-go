package terminal

import (
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
)

func TestWrapLineShort(t *testing.T) {
	result := WrapLine("hello", 20)
	if len(result) != 1 || result[0] != "hello" {
		t.Fatalf("expected ['hello'], got %v", result)
	}
}

func TestWrapLineLong(t *testing.T) {
	result := WrapLine("abcdefghij", 4)
	expected := []string{"abcd", "efgh", "ij"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d parts, got %d: %v", len(expected), len(result), result)
	}
	for i, want := range expected {
		if result[i] != want {
			t.Fatalf("part %d: expected %q, got %q", i, want, result[i])
		}
	}
}

func TestWrapLineEmpty(t *testing.T) {
	result := WrapLine("", 20)
	if len(result) != 1 || result[0] != "" {
		t.Fatalf("expected [''], got %v", result)
	}
}

func TestWrapLineExactWidth(t *testing.T) {
	result := WrapLine("abcd", 4)
	if len(result) != 1 || result[0] != "abcd" {
		t.Fatalf("expected ['abcd'], got %v", result)
	}
}

func TestWrapLineZeroWidth(t *testing.T) {
	result := WrapLine("hello", 0)
	if len(result) != 1 || result[0] != "hello" {
		t.Fatalf("expected ['hello'] for zero width, got %v", result)
	}
}

func TestReflowLinesBasic(t *testing.T) {
	lines := []string{"hello", "world"}
	result := ReflowLines(lines, 20, 10)
	if len(result) != 2 || result[0] != "hello" || result[1] != "world" {
		t.Fatalf("expected [hello world], got %v", result)
	}
}

func TestReflowLinesWrapping(t *testing.T) {
	lines := []string{"abcdefghij"}
	result := ReflowLines(lines, 4, 10)
	if len(result) != 3 {
		t.Fatalf("expected 3 display lines, got %d: %v", len(result), result)
	}
	if result[0] != "abcd" || result[1] != "efgh" || result[2] != "ij" {
		t.Fatalf("unexpected wrap result: %v", result)
	}
}

func TestReflowLinesScrolling(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	result := ReflowLines(lines, 20, 3)
	if len(result) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(result), result)
	}
	if result[0] != "c" || result[1] != "d" || result[2] != "e" {
		t.Fatalf("expected [c d e], got %v", result)
	}
}

func TestReflowLinesDifferentWidths(t *testing.T) {
	lines := []string{"abcdefgh"}
	narrow := ReflowLines(lines, 4, 10)
	wide := ReflowLines(lines, 8, 10)
	if len(narrow) != 2 {
		t.Fatalf("narrow: expected 2 display lines, got %d", len(narrow))
	}
	if len(wide) != 1 {
		t.Fatalf("wide: expected 1 display line, got %d", len(wide))
	}
}

func TestReflowLinesScrollingWithWrap(t *testing.T) {
	// Long line wraps to 3 display lines, plus 2 short lines = 5 display lines
	// With height=3, should show last 3
	lines := []string{"abcdefghijkl", "x", "y"}
	result := ReflowLines(lines, 4, 3)
	// "abcdefghijkl" wraps to ["abcd", "efgh", "ijkl"], then "x", "y"
	// Total: 5 display lines, show last 3: ["ijkl", "x", "y"]
	if len(result) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(result), result)
	}
	if result[0] != "ijkl" || result[1] != "x" || result[2] != "y" {
		t.Fatalf("expected [ijkl x y], got %v", result)
	}
}

func TestReflowLinesEmpty(t *testing.T) {
	result := ReflowLines(nil, 20, 10)
	if len(result) != 0 {
		t.Fatalf("expected empty, got %v", result)
	}
}

func TestRenderReflowOnResize(t *testing.T) {
	// Verify that resizing the terminal reflows content
	term := New(NewANSI())
	long := strings.Repeat("x", 100)
	term.Write(process.Data{process.Chars(long + "\n")})

	// At default width (55), line wraps to 2 display lines
	out1 := term.Render()
	count1 := strings.Count(out1, strings.Repeat("x", 55))

	// Resize to 25
	term.Resize(25, 20)
	out2 := term.Render()
	count2 := strings.Count(out2, strings.Repeat("x", 25))

	if count1 < 1 {
		t.Fatal("expected at least one 55-char line at default width")
	}
	if count2 < 1 {
		t.Fatal("expected at least one 25-char line after resize")
	}
}
