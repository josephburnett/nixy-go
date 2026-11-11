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

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

// typeLine feeds each character of s as a keystroke, then presses Enter.
func typeLine(t *testing.T, s *Session, term *terminal.T, line string) {
	t.Helper()
	for _, r := range line {
		s.HandleKeystroke(process.Chars(string(r)), term)
	}
	s.HandleKeystroke(process.TermEnter, term)
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

func TestPromptPath(t *testing.T) {
	home := []string{"home", "alice"}
	for _, tc := range []struct {
		cwd  []string
		want string
	}{
		{[]string{}, "/"},
		{[]string{"home"}, "/home"},
		{[]string{"home", "alice"}, "~"},
		{[]string{"home", "alice", "docs"}, "~/docs"},
		{[]string{"home", "alicex"}, "/home/alicex"},
		{[]string{"home", "nixy"}, "/home/nixy"},
	} {
		if got := promptPath(tc.cwd, home); got != tc.want {
			t.Errorf("promptPath(%v) = %q, want %q", tc.cwd, got, tc.want)
		}
	}
}

// TestLoginLandsHome: the laptop login and ssh both start in the
// player's home, and the prompt says so the way bash does.
func TestLoginLandsHome(t *testing.T) {
	sess, _ := New()
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)
	loginAs(t, sess, term, "alice")
	if got := strings.Join(sess.Shell.CurrentDirectory(), "/"); got != "home/alice" {
		t.Fatalf("login cwd = %q, want home/alice", got)
	}
	if p := term.State.Prompt.Path; p != "~" {
		t.Errorf("prompt path = %q, want ~", p)
	}
	typeLine(t, sess, term, "ssh nixy")
	if sess.Shell.Hostname() != "nixy" {
		t.Fatalf("ssh nixy failed: on %q", sess.Shell.Hostname())
	}
	if got := strings.Join(sess.Shell.CurrentDirectory(), "/"); got != "home/alice" {
		t.Fatalf("ssh cwd = %q, want home/alice", got)
	}
}

// TestLoginRefusesExistingAccounts: every user that owns something in
// the world, once every quest has set its world up, is an account the
// player can't take at the login prompt. Root and nixy are the ones
// today; a new world user is covered without touching this test.
func TestLoginRefusesExistingAccounts(t *testing.T) {
	played, _ := buildScenario(t, keyboardScenario{name: "all", untilQuest: "permissions",
		commands: []string{"exit", "ssh server"}})
	owners := map[string]bool{"root": true}
	for _, h := range []string{"laptop", "nixy", "server"} {
		c, err := played.Game.Sim.GetComputer(h)
		if err != nil {
			t.Fatalf("%s not booted: %v", h, err)
		}
		var walk func(f *file.F)
		walk = func(f *file.F) {
			owners[f.Owner] = true
			for _, child := range f.Files {
				walk(child)
			}
		}
		walk(c.Filesystem)
	}
	delete(owners, played.Username)
	if !owners["nixy"] {
		t.Fatal("expected nixy among the world's users")
	}
	for name := range owners {
		sess, err := New()
		if err != nil {
			t.Fatal(err)
		}
		term := terminal.New(terminal.NewANSI())
		term.Resize(80, 24)
		sess.InitTerminal(term)
		typeAccepted(t, sess, term, name)
		if containsKey(term.State.ValidKeys, process.TermEnter) {
			t.Errorf("login offers Enter on existing account %q", name)
		}
		if !strings.Contains(term.State.Thought, "someone else's") {
			t.Errorf("thought for %q should explain the name is taken, got %q", name, term.State.Thought)
		}
		sess.HandleKeystroke(process.TermEnter, term)
		if sess.Mode != ModeLogin {
			t.Errorf("logged in as existing account %q", name)
		}
		// A longer name that starts with it is fine.
		typeAccepted(t, sess, term, "x")
		if !containsKey(term.State.ValidKeys, process.TermEnter) {
			t.Errorf("login refuses %q", name+"x")
		}
	}
}

// The tracker records each command line's exit status, for quests that
// check what a command did rather than what was typed.
func TestTrackerRecordsExitStatus(t *testing.T) {
	sess, term := newLoggedIn(t, "alice")
	typeLine(t, sess, term, "pwd")
	typeLine(t, sess, term, "ls /root")
	typeLine(t, sess, term, "ssh nixy")
	recs := sess.Game.Manager.Tracker.Records()
	want := []struct {
		cmd     string
		status  int
		running bool
	}{{"pwd", 0, false}, {"ls /root", 1, false}, {"ssh nixy", 0, true}}
	if len(recs) != len(want) {
		t.Fatalf("records: %+v", recs)
	}
	for i, w := range want {
		r := recs[i]
		if r.Command != w.cmd || r.Status != w.status || r.Running != w.running {
			t.Errorf("record %d = %+v, want %+v", i, r, w)
		}
	}
}
