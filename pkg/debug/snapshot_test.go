package debug

import (
	"strings"
	"testing"

	_ "github.com/josephburnett/nixy-go/pkg/command/apt"
	_ "github.com/josephburnett/nixy-go/pkg/command/cat"
	_ "github.com/josephburnett/nixy-go/pkg/command/grep"
	_ "github.com/josephburnett/nixy-go/pkg/command/ls"
	_ "github.com/josephburnett/nixy-go/pkg/command/mv"
	_ "github.com/josephburnett/nixy-go/pkg/command/pwd"
	_ "github.com/josephburnett/nixy-go/pkg/command/rm"
	_ "github.com/josephburnett/nixy-go/pkg/command/shell"
	_ "github.com/josephburnett/nixy-go/pkg/command/ssh"
	_ "github.com/josephburnett/nixy-go/pkg/command/sudo"
	_ "github.com/josephburnett/nixy-go/pkg/command/touch"

	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/session"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

// typeLine drives keystroke-by-keystroke through HandleKeystroke, pushing
// a snapshot after each one (mirroring how cmd/repl uses the recorder).
func typeLine(t *testing.T, s *session.Session, term *terminal.T, r *Recorder, line string) {
	t.Helper()
	for _, c := range line {
		s.HandleKeystroke(process.Chars(string(c)), term)
		r.Push(s, term, process.Chars(string(c)))
	}
	s.HandleKeystroke(process.TermEnter, term)
	r.Push(s, term, process.TermEnter)
}

func TestRecorderCapturesPlayingState(t *testing.T) {
	sess, err := session.New()
	if err != nil {
		t.Fatal(err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)

	r := NewRecorder(20)
	r.Push(sess, term, nil)

	typeLine(t, sess, term, r, "alice")
	typeLine(t, sess, term, r, "ssh nixy")

	dump := r.Dump()

	// Should reflect the playing state on the latest snapshot.
	if !strings.Contains(dump, "Playing") {
		t.Fatalf("expected Playing mode in dump; got:\n%s", dump)
	}
	if !strings.Contains(dump, `"alice"`) {
		t.Fatalf("expected username 'alice' in dump; got:\n%s", dump)
	}
	if !strings.Contains(dump, "ssh nixy") {
		t.Fatalf("expected 'ssh nixy' in tracker; got:\n%s", dump)
	}
	if !strings.Contains(dump, "connect") {
		t.Fatalf("expected 'connect' quest id in dump; got:\n%s", dump)
	}
}

func TestRecorderRingDropsOldest(t *testing.T) {
	sess, err := session.New()
	if err != nil {
		t.Fatal(err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)

	r := NewRecorder(3)
	for range 10 {
		r.Push(sess, term, process.Chars("a"))
	}
	if got := strings.Count(r.Dump(), "--- snapshot "); got != 3 {
		t.Fatalf("expected 3 entries in ring, got %d", got)
	}
}

func TestRecorderInitialSnapshotIsLogin(t *testing.T) {
	sess, err := session.New()
	if err != nil {
		t.Fatal(err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)

	r := NewRecorder(5)
	r.Push(sess, term, nil)

	dump := r.Dump()
	if !strings.Contains(dump, "Login") {
		t.Fatalf("expected initial Login mode in dump:\n%s", dump)
	}
	if !strings.Contains(dump, "(initial)") {
		t.Fatalf("expected initial keystroke marker in dump:\n%s", dump)
	}
}
