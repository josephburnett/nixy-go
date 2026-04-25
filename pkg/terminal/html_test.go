package terminal

import (
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
)

func TestHTMLRenderBox(t *testing.T) {
	term := New(NewHTML())
	out := term.Render()
	if !strings.Contains(out, "<pre>") || !strings.Contains(out, "</pre>") {
		t.Fatal("expected <pre> tags")
	}
	if !strings.Contains(out, `class="box"`) {
		t.Fatal("expected box class")
	}
	if !strings.Contains(out, "┌") || !strings.Contains(out, "┘") {
		t.Fatal("expected box-drawing characters")
	}
}

func TestHTMLRenderPrompt(t *testing.T) {
	term := New(NewHTML())
	term.Write(process.Data{process.Chars("hello")})
	out := term.Render()
	if !strings.Contains(out, `class="prompt"`) {
		t.Fatal("expected prompt class")
	}
	if !strings.Contains(out, "&gt; ") || !strings.Contains(out, "hello") {
		t.Fatalf("expected prompt prefix and typed input, got:\n%s", out)
	}
}

func TestHTMLRenderDialog(t *testing.T) {
	term := New(NewHTML())
	term.SetDialog([]string{"Nixy says hi"})
	out := term.Render()
	if !strings.Contains(out, `class="dialog dialog-0"`) {
		t.Fatal("expected dialog dialog-0 class")
	}
	if !strings.Contains(out, "Nixy says hi") {
		t.Fatal("expected dialog content")
	}
	// Dialog persists across renders
	out2 := term.Render()
	if !strings.Contains(out2, "Nixy says hi") {
		t.Fatal("dialog should persist across renders")
	}
}

func TestHTMLRenderNotice(t *testing.T) {
	term := New(NewHTML())
	term.Notify("bad input")
	out := term.Render()
	if !strings.Contains(out, `class="notice"`) {
		t.Fatal("expected notice class")
	}
	if !strings.Contains(out, "bad input") {
		t.Fatal("expected notice content")
	}
}

func TestHTMLRenderDialogAboveBox(t *testing.T) {
	term := New(NewHTML())
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

func TestHTMLRenderNoticeBelowBox(t *testing.T) {
	term := New(NewHTML())
	term.Notify("oops")
	out := term.Render()
	noticeIdx := strings.Index(out, "oops")
	boxBottomIdx := strings.Index(out, "└")
	if noticeIdx < 0 || boxBottomIdx < 0 {
		t.Fatalf("expected notice and box bottom in output, got:\n%s", out)
	}
	if noticeIdx < boxBottomIdx {
		t.Fatal("notice should appear below the terminal box")
	}
}

func renderHTMLForKeyboard(valid []process.Datum, hint process.Datum) string {
	term := New(NewHTML())
	term.SetKeyboard(valid, hint)
	return term.Render()
}

func TestHTMLRenderKeyboardClasses(t *testing.T) {
	valid := []process.Datum{process.Chars("s")}
	hint := process.Chars("s")
	out := renderHTMLForKeyboard(valid, hint)
	if !strings.Contains(out, `class="key-hint"`) {
		t.Fatal("expected key-hint class for hint key")
	}
	if !strings.Contains(out, `class="key-dim"`) {
		t.Fatal("expected key-dim class for unavailable keys")
	}
}

func TestHTMLRenderKeyboardValid(t *testing.T) {
	valid := []process.Datum{process.Chars("l"), process.TermEnter}
	out := renderHTMLForKeyboard(valid, nil)
	if !strings.Contains(out, `class="key-valid">l</span>`) {
		t.Fatal("expected key-valid class for 'l'")
	}
	if !strings.Contains(out, `class="key-valid">[enter]</span>`) {
		t.Fatal("expected key-valid class for enter")
	}
}

func TestHTMLRenderEscaping(t *testing.T) {
	term := New(NewHTML())
	term.Write(process.Data{process.Chars("<script>alert('xss')</script>\n")})
	out := term.Render()
	if strings.Contains(out, "<script>") {
		t.Fatal("HTML should be escaped in output")
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatal("expected escaped HTML entities")
	}
}

func TestHTMLRenderKeyboardAllLetters(t *testing.T) {
	out := renderHTMLForKeyboard(nil, nil)
	for c := 'a'; c <= 'z'; c++ {
		if !strings.Contains(out, string(c)) {
			t.Fatalf("keyboard missing letter '%c'", c)
		}
	}
}
