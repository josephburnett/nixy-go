package terminal

import (
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
)

func TestRenderKeyboardAllDisabled(t *testing.T) {
	out := RenderKeyboard(nil, nil)
	// All keys should be dim
	if !strings.Contains(out, colorDim+"q"+colorReset) {
		t.Fatal("expected dim 'q' when no valid keys")
	}
	if strings.Contains(out, colorWhite) {
		t.Fatal("no keys should be white when none are valid")
	}
}

func TestRenderKeyboardValidKeys(t *testing.T) {
	valid := []process.Datum{
		process.Chars("s"),
		process.Chars("l"),
		process.TermEnter,
	}
	out := RenderKeyboard(valid, nil)
	if !strings.Contains(out, colorWhite+"s"+colorReset) {
		t.Fatal("expected white 's'")
	}
	if !strings.Contains(out, colorWhite+"l"+colorReset) {
		t.Fatal("expected white 'l'")
	}
	if !strings.Contains(out, colorWhite+"[enter]"+colorReset) {
		t.Fatal("expected white '[enter]'")
	}
	// Non-valid key should be dim
	if !strings.Contains(out, colorDim+"q"+colorReset) {
		t.Fatal("expected dim 'q'")
	}
}

func TestRenderKeyboardHintKey(t *testing.T) {
	valid := []process.Datum{
		process.Chars("s"),
		process.Chars("l"),
	}
	hint := process.Chars("s")
	out := RenderKeyboard(valid, hint)
	if !strings.Contains(out, colorGreen+"s"+colorReset) {
		t.Fatal("expected green 's' for hint")
	}
	// 'l' should be white (valid but not hint)
	if !strings.Contains(out, colorWhite+"l"+colorReset) {
		t.Fatal("expected white 'l'")
	}
}

func TestRenderKeyboardHintOverridesValid(t *testing.T) {
	valid := []process.Datum{process.TermEnter}
	hint := process.TermEnter
	out := RenderKeyboard(valid, hint)
	// Should be green, not white
	if !strings.Contains(out, colorGreen+"[enter]"+colorReset) {
		t.Fatal("expected green '[enter]' when it's the hint")
	}
	if strings.Contains(out, colorWhite+"[enter]"+colorReset) {
		t.Fatal("should not be white when it's the hint")
	}
}

func TestRenderKeyboardSpace(t *testing.T) {
	valid := []process.Datum{process.Chars(" ")}
	out := RenderKeyboard(valid, nil)
	if !strings.Contains(out, colorWhite+"[space]"+colorReset) {
		t.Fatal("expected white '[space]'")
	}
}

func TestRenderKeyboardBackspace(t *testing.T) {
	valid := []process.Datum{process.TermBackspace}
	out := RenderKeyboard(valid, nil)
	if !strings.Contains(out, colorWhite+"[bksp]"+colorReset) {
		t.Fatal("expected white '[bksp]'")
	}
}

func TestRenderKeyboardSpecialChars(t *testing.T) {
	valid := []process.Datum{
		process.Chars("."),
		process.Chars("/"),
		process.Chars("|"),
	}
	out := RenderKeyboard(valid, nil)
	if !strings.Contains(out, colorWhite+"[.]"+colorReset) {
		t.Fatal("expected white '[.]'")
	}
	if !strings.Contains(out, colorWhite+"[/]"+colorReset) {
		t.Fatal("expected white '[/]'")
	}
	if !strings.Contains(out, colorWhite+"[|]"+colorReset) {
		t.Fatal("expected white '[|]'")
	}
}

func TestRenderKeyboardHasAllRows(t *testing.T) {
	out := RenderKeyboard(nil, nil)
	lines := strings.Split(out, "\n")
	// Should have 4 rows: 3 letter rows + 1 special row (+ trailing newline)
	nonEmpty := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmpty++
		}
	}
	if nonEmpty != 4 {
		t.Fatalf("expected 4 keyboard rows, got %d", nonEmpty)
	}
}

func TestRenderKeyboardContainsAllLetters(t *testing.T) {
	out := RenderKeyboard(nil, nil)
	for c := 'a'; c <= 'z'; c++ {
		if !strings.Contains(out, string(c)) {
			t.Fatalf("keyboard missing letter '%c'", c)
		}
	}
}

func TestDatumSetContains(t *testing.T) {
	s := buildDatumSet([]process.Datum{
		process.Chars("a"),
		process.TermEnter,
	})
	if !s.contains(process.Chars("a")) {
		t.Fatal("should contain 'a'")
	}
	if !s.contains(process.TermEnter) {
		t.Fatal("should contain TermEnter")
	}
	if s.contains(process.Chars("b")) {
		t.Fatal("should not contain 'b'")
	}
	if s.contains(process.TermBackspace) {
		t.Fatal("should not contain TermBackspace")
	}
}
