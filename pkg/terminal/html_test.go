package terminal

import (
	"html"
	"regexp"
	"slices"
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
	if !strings.Contains(out, `class="dialog"`) {
		t.Fatal("expected dialog class")
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
	if !strings.Contains(out, `class="key-valid" data-key="l">l</span>`) {
		t.Fatal("expected key-valid class for 'l'")
	}
	if !strings.Contains(out, `class="key-valid" data-key="Enter">[enter]</span>`) {
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

// allKeyboardDatums lists every key drawn on the on-screen keyboard.
func allKeyboardDatums() []process.Datum {
	var all []process.Datum
	for _, row := range keyboardRows {
		for _, k := range row {
			all = append(all, KeyDatum(k))
		}
	}
	for _, sk := range specialKeys {
		all = append(all, sk.datum)
	}
	return all
}

var dataKeyAttr = regexp.MustCompile(`data-key="([^"]*)"`)

// TestHTMLKeyboardTappableKeysMatchValidSet: exactly the valid keys carry
// data-key, and each one decodes back to its datum. That's what makes a tap
// send the same keystroke as typing, and a dim key do nothing.
func TestHTMLKeyboardTappableKeysMatchValidSet(t *testing.T) {
	all := allKeyboardDatums()
	cases := [][]process.Datum{
		nil,
		all,
		{process.Chars("l"), process.Chars(" "), process.Chars("|"), process.TermEnter},
		{process.TermBackspace, process.Chars("/"), process.Chars(".")},
	}
	for _, valid := range cases {
		// Hint on a key that isn't valid must not make it tappable.
		out := renderHTMLForKeyboard(valid, process.Chars("q"))
		var got []process.Datum
		for _, m := range dataKeyAttr.FindAllStringSubmatch(out, -1) {
			d := KeyDatum(html.UnescapeString(m[1]))
			if d == nil {
				t.Fatalf("data-key %q doesn't decode to a keystroke", m[1])
			}
			got = append(got, d)
		}
		if len(got) != len(valid) {
			t.Fatalf("valid=%v: %d tappable keys, want %d\n%s", valid, len(got), len(valid), out)
		}
		for _, d := range valid {
			if !slices.Contains(got, d) {
				t.Errorf("valid key %v is not tappable", d)
			}
		}
	}
}

func TestKeyNameRoundTrip(t *testing.T) {
	for _, d := range allKeyboardDatums() {
		name := KeyName(d)
		if back := KeyDatum(name); back != d {
			t.Errorf("KeyDatum(KeyName(%v)) = %v", d, back)
		}
	}
	// Ctrl+L isn't drawn, but both front ends send it.
	if KeyDatum(KeyName(process.TermClear)) != process.Datum(process.TermClear) {
		t.Error("Ctrl+L doesn't round-trip")
	}
	for _, name := range []string{"", "Tab", "ArrowUp", "Shift"} {
		if d := KeyDatum(name); d != nil {
			t.Errorf("KeyDatum(%q) = %v, want nil", name, d)
		}
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
