package session

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

	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

// typeLine feeds each character of s as a keystroke, then presses Enter.
func typeLine(t *testing.T, s *Session, term *terminal.T, line string) {
	t.Helper()
	for _, r := range line {
		if s.HandleKeystroke(process.Chars(string(r)), term) {
			t.Fatalf("unexpected EOF while typing %q", line)
		}
	}
	if s.HandleKeystroke(process.TermEnter, term) {
		t.Fatalf("unexpected EOF on Enter for %q", line)
	}
}

// loginAs drives the session through the login phase so the rest of the
// test can exercise game behaviour. The default username is short and
// safe.
func loginAs(t *testing.T, s *Session, term *terminal.T, name string) {
	t.Helper()
	if s.Mode != ModeLogin {
		t.Fatalf("expected session to start in ModeLogin, got %v", s.Mode)
	}
	typeLine(t, s, term, name)
	if s.Mode != ModePlaying {
		t.Fatalf("expected ModePlaying after login, got %v (username=%q)", s.Mode, s.Username)
	}
}

// TestSshNixyCompletesConnectQuest drives the full keystroke pipeline to
// confirm that typing "ssh nixy" + Enter actually records the command,
// completes the Connect quest, and activates Orientation with its dialog.
// Regression test for a bug where tracker.Record was only called in test
// helpers, never in the real session loop.
func TestSshNixyCompletesConnectQuest(t *testing.T) {
	sess, err := New()
	if err != nil {
		t.Fatal(err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)
	loginAs(t, sess, term, "alice")

	if state := sess.Game.Manager.GetQuestState("connect"); state != game.QuestActive {
		t.Fatalf("connect should start active, got %v", state)
	}

	typeLine(t, sess, term, "ssh nixy")

	if state := sess.Game.Manager.GetQuestState("connect"); state != game.QuestComplete {
		t.Fatalf("connect should be complete after ssh nixy, got %v", state)
	}
	if state := sess.Game.Manager.GetQuestState("orientation"); state != game.QuestActive {
		t.Fatalf("orientation should be active after connect, got %v", state)
	}

	// Orientation activation dialog should be on the terminal.
	out := term.Render()
	if !strings.Contains(out, "look around") {
		t.Fatalf("expected orientation dialog ('look around') in render, got:\n%s", out)
	}
}

// TestLoginRejectsInvalidAndAcceptsValid drives the login phase through a
// few edge cases: empty Enter is ignored, invalid chars are dropped at the
// guard layer (we feed them anyway and verify they don't land), Backspace
// trims, and a valid username transitions to ModePlaying with that name in
// the prompt.
func TestLoginRejectsInvalidAndAcceptsValid(t *testing.T) {
	sess, err := New()
	if err != nil {
		t.Fatal(err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)

	// Enter on empty buffer should NOT log in.
	sess.HandleKeystroke(process.TermEnter, term)
	if sess.Mode != ModeLogin {
		t.Fatal("empty enter should not log in")
	}

	// Type letters; UPPERCASE should be ignored (login filter), lowercase accepted.
	for _, r := range []rune{'a', 'B', 'c', '1', 'd'} {
		sess.HandleKeystroke(process.Chars(string(r)), term)
	}
	if term.State.Line != "acd" {
		t.Fatalf("expected 'acd' (uppercase + digit dropped), got %q", term.State.Line)
	}

	// Backspace once.
	sess.HandleKeystroke(process.TermBackspace, term)
	if term.State.Line != "ac" {
		t.Fatalf("expected 'ac' after backspace, got %q", term.State.Line)
	}

	// Enter with a valid name should transition.
	sess.HandleKeystroke(process.TermEnter, term)
	if sess.Mode != ModePlaying {
		t.Fatalf("expected ModePlaying after valid login, got %v", sess.Mode)
	}
	if sess.Username != "ac" {
		t.Fatalf("expected username 'ac', got %q", sess.Username)
	}
	if term.State.Prompt.User != "ac" {
		t.Fatalf("expected prompt user 'ac', got %q", term.State.Prompt.User)
	}
}

// TestLoginRejectsTooLong confirms that input past the 8-char limit is dropped.
func TestLoginRejectsTooLong(t *testing.T) {
	sess, err := New()
	if err != nil {
		t.Fatal(err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)

	for _, r := range "abcdefghijkl" { // 12 chars
		sess.HandleKeystroke(process.Chars(string(r)), term)
	}
	if len(term.State.Line) != 8 {
		t.Fatalf("expected buffer capped at 8 chars, got %d: %q", len(term.State.Line), term.State.Line)
	}
}

// TestOrientationQuestCompletes drives enough commands on nixy to complete
// the Orientation quest and verifies the next quest activates.
func TestOrientationQuestCompletes(t *testing.T) {
	sess, err := New()
	if err != nil {
		t.Fatal(err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)
	loginAs(t, sess, term, "alice")

	typeLine(t, sess, term, "ssh nixy")
	typeLine(t, sess, term, "pwd")
	typeLine(t, sess, term, "ls")
	typeLine(t, sess, term, "cd /home/nixy")

	// Orientation requires visiting /home/nixy. The cd itself should count —
	// the user shouldn't have to type a second command after cd.
	if state := sess.Game.Manager.GetQuestState("orientation"); state != game.QuestComplete {
		t.Fatalf("orientation should be complete after cd /home/nixy, got %v", state)
	}
	if state := sess.Game.Manager.GetQuestState("inspection"); state != game.QuestActive {
		t.Fatalf("inspection should be active after orientation, got %v", state)
	}
}
