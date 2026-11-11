package session

import (
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

func newLoggedIn(t *testing.T, name string) (*Session, *terminal.T) {
	t.Helper()
	sess, err := New()
	if err != nil {
		t.Fatal(err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)
	loginAs(t, sess, term, name)
	return sess, term
}

// assertKeyboardLive fails if the player has no key to press. An empty
// valid set is a frozen game (the web build has no other way out).
func assertKeyboardLive(t *testing.T, term *terminal.T) {
	t.Helper()
	if len(term.State.ValidKeys) == 0 {
		t.Fatalf("keyboard has no valid keys; prompt=%+v line=%q", term.State.Prompt, term.State.Line)
	}
}

func TestExitOnLaptopReturnsToLogin(t *testing.T) {
	sess, term := newLoggedIn(t, "alice")
	typeLine(t, sess, term, "exit")

	if sess.Mode != ModeLogin {
		t.Fatalf("exit on laptop should log out to the login prompt, got mode %v", sess.Mode)
	}
	if term.State.Prompt.Raw != loginPrompt {
		t.Fatalf("expected %q prompt, got %+v", loginPrompt, term.State.Prompt)
	}
	assertKeyboardLive(t, term)
	if term.State.Line != "" {
		t.Fatalf("typed command should be committed to history, input line still %q", term.State.Line)
	}
	if !historyContains(term, "exit") {
		t.Fatalf("the exit command line is missing from history:\n%s", term.Render())
	}
	if !strings.Contains(term.Render(), "logout") {
		t.Fatalf("expected a logout line in the scrollback:\n%s", term.Render())
	}
}

func TestReloginSameUserResumesGame(t *testing.T) {
	sess, term := newLoggedIn(t, "alice")
	g := sess.Game
	typeLine(t, sess, term, "ssh nixy")
	typeLine(t, sess, term, "exit") // back to laptop
	if sess.Mode != ModePlaying || sess.Shell.Hostname() != "laptop" {
		t.Fatalf("exit on nixy should return to laptop, got mode=%v", sess.Mode)
	}
	typeLine(t, sess, term, "exit") // log out
	loginAs(t, sess, term, "alice")

	if sess.Game != g {
		t.Fatal("logging back in as the same user should resume the same game")
	}
	if st := sess.Game.Manager.GetQuestState("connect"); st != game.QuestComplete {
		t.Fatalf("connect progress lost across logout: %v", st)
	}
	if got := sess.Shell.Hostname(); got != "laptop" {
		t.Fatalf("login shell should be on laptop, got %q", got)
	}
	assertKeyboardLive(t, term)

	// The resumed shell is live: commands still run.
	typeLine(t, sess, term, "ssh nixy")
	if got := sess.Shell.Hostname(); got != "nixy" {
		t.Fatalf("resumed shell should accept commands, hostname=%q", got)
	}
}

func TestLoginAsOtherUserStartsOwnGame(t *testing.T) {
	sess, term := newLoggedIn(t, "alice")
	alice := sess.Game
	typeLine(t, sess, term, "ssh nixy")
	typeLine(t, sess, term, "exit")
	typeLine(t, sess, term, "exit")

	loginAs(t, sess, term, "bob")
	if sess.Game == alice {
		t.Fatal("a different user should get their own game")
	}
	if st := sess.Game.Manager.GetQuestState("connect"); st != game.QuestActive {
		t.Fatalf("bob should start at the first quest, connect=%v", st)
	}
	if term.State.Prompt.User != "bob" {
		t.Fatalf("prompt should show bob, got %+v", term.State.Prompt)
	}
	typeLine(t, sess, term, "exit")

	loginAs(t, sess, term, "alice")
	if sess.Game != alice {
		t.Fatal("alice's game should still be there after bob played")
	}
	// nx talks to the active game; it must follow the login, not stay
	// pointed at bob's game.
	typeLine(t, sess, term, "nx log")
	if !historyContains(term, "✓") {
		t.Fatalf("nx log should show alice's completed quest:\n%s", term.Render())
	}
}

func historyContains(term *terminal.T, sub string) bool {
	for _, l := range term.State.Lines {
		if strings.Contains(l.Input, sub) {
			return true
		}
	}
	return false
}

func TestNxPanicUnwindsNestedShellsToLaptop(t *testing.T) {
	sess, term := newLoggedIn(t, "alice")
	typeLine(t, sess, term, "ssh nixy")
	typeLine(t, sess, term, "ssh laptop")
	typeLine(t, sess, term, "ssh nixy")
	typeLine(t, sess, term, "nx panic")

	if sess.Mode != ModePlaying {
		t.Fatalf("nx panic must not log out, mode=%v", sess.Mode)
	}
	if got := sess.Shell.Hostname(); got != "laptop" {
		t.Fatalf("nx panic should land on laptop, got %q", got)
	}
	if !strings.Contains(term.Render(), "Panic!") {
		t.Fatalf("panic message missing:\n%s", term.Render())
	}
	assertKeyboardLive(t, term)

	// All nested shells are gone: one exit logs out of the laptop.
	typeLine(t, sess, term, "exit")
	if sess.Mode != ModeLogin {
		t.Fatalf("after nx panic the player should be in the laptop's login shell; exit gave mode %v", sess.Mode)
	}
}

func TestNxPanicOnLaptopStaysLoggedIn(t *testing.T) {
	sess, term := newLoggedIn(t, "alice")
	typeLine(t, sess, term, "nx panic")

	if sess.Mode != ModePlaying {
		t.Fatalf("nx panic on laptop must not log out, mode=%v", sess.Mode)
	}
	if got := sess.Shell.Hostname(); got != "laptop" {
		t.Fatalf("expected laptop, got %q", got)
	}
	assertKeyboardLive(t, term)
	typeLine(t, sess, term, "ssh nixy")
	if got := sess.Shell.Hostname(); got != "nixy" {
		t.Fatalf("shell after panic should be usable, hostname=%q", got)
	}
}

func TestCtrlLClearsScreenKeepsInput(t *testing.T) {
	sess, term := newLoggedIn(t, "alice")
	typeLine(t, sess, term, "ls")
	for _, r := range "ssh" {
		sess.HandleKeystroke(process.Chars(string(r)), term)
	}
	sess.HandleKeystroke(process.TermClear, term)
	if len(term.State.Lines) != 0 {
		t.Fatalf("Ctrl+L should clear the scrollback, have %d lines", len(term.State.Lines))
	}
	if term.State.Line != "ssh" || sess.Shell.CurrentCommand() != "ssh" {
		t.Fatalf("Ctrl+L should keep the typed input, line=%q shell=%q", term.State.Line, sess.Shell.CurrentCommand())
	}
	typeLine(t, sess, term, " nixy")
	if got := sess.Shell.Hostname(); got != "nixy" {
		t.Fatalf("command typed across Ctrl+L should still run, hostname=%q", got)
	}
}

// TestKeyboardNeverEmpty drives a few hundred keystrokes (hint when there
// is one, otherwise cycling through the valid set) from several starting
// states and checks the keyboard always offers something. Guards the
// frozen-web-build class: a state with zero valid keys is a dead end.
func TestKeyboardNeverEmpty(t *testing.T) {
	for _, prefix := range [][]string{nil, {"ssh nixy"}, {"exit"}, {"nx panic"}} {
		sess, term := newLoggedIn(t, "alice")
		for _, l := range prefix {
			typeLine(t, sess, term, l)
		}
		for i := 0; i < 300; i++ {
			assertKeyboardLive(t, term)
			d := term.State.HintKey
			if d == nil {
				valid := term.State.ValidKeys
				d = valid[i%len(valid)]
			}
			sess.HandleKeystroke(d, term)
		}
		assertKeyboardLive(t, term)
	}
}

// TestNxPanicRestoresShippedBin: a machine's shipped commands come back
// after nx panic, however they were lost (here: deleted from /bin), so
// losing ssh on the laptop can't strand the player. Commands installed
// since stay.
func TestNxPanicRestoresShippedBin(t *testing.T) {
	sess, term := newLoggedIn(t, "alice")
	laptop, _ := sess.Game.Sim.GetComputer("laptop")
	nixy, _ := sess.Game.Sim.GetComputer("nixy")
	delete(laptop.Filesystem.Files["bin"].Files, "ssh")
	delete(nixy.Filesystem.Files["bin"].Files, "apt")
	nixy.Filesystem.Files["bin"].Files["mv"] = &file.F{Type: file.Binary, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read, Data: "mv"}
	nixy.Filesystem.Files["bin"].Files["rm"].Data = "cat" // a broken copy

	typeLine(t, sess, term, "nx panic")

	for host, names := range map[string][]string{"laptop": {"ssh", "ls", "cat"}, "nixy": {"apt", "mv"}} {
		for _, n := range names {
			if _, err := sess.Game.Sim.LookupBinary(host, "alice", n); err != nil {
				t.Errorf("%s: %s missing after panic: %v", host, n, err)
			}
		}
	}
	if d := nixy.Filesystem.Files["bin"].Files["rm"].Data; d != "rm" {
		t.Errorf("nixy /bin/rm not restored: runs %q", d)
	}
	typeLine(t, sess, term, "ssh nixy")
	if got := sess.Shell.Hostname(); got != "nixy" {
		t.Fatalf("ssh after panic: on %q", got)
	}
}

// TestProcessTableNoLeak: ssh, exit, nx panic, logout and login, over
// and over, leave only the live shells in the process tables.
func TestProcessTableNoLeak(t *testing.T) {
	sess, term := newLoggedIn(t, "alice")
	for range 10 {
		typeLine(t, sess, term, "ssh nixy")
		if sess.Shell.Hostname() != "nixy" {
			t.Fatal("ssh nixy didn't connect")
		}
		assertProcessTable(t, sess)
		typeLine(t, sess, term, "exit")
		typeLine(t, sess, term, "ssh nixy")
		typeLine(t, sess, term, "nx panic")
		if sess.Shell.Hostname() != "laptop" {
			t.Fatal("nx panic didn't return to the laptop")
		}
		assertProcessTable(t, sess)
		typeLine(t, sess, term, "exit") // log out
		if sess.Mode != ModeLogin {
			t.Fatal("exit on the laptop didn't log out")
		}
		assertProcessTable(t, sess)
		typeLine(t, sess, term, "alice")
		assertProcessTable(t, sess)
	}
	c, _ := sess.Game.Sim.GetComputer("laptop")
	if ps := c.Processes(); len(ps) != 1 || ps[0].Owner != "alice" {
		t.Fatalf("laptop processes: %+v", ps)
	}
}
