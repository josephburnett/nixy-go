package session

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

// allKeys is every key a physical keyboard can send to the session,
// except Ctrl+L, which the terminal always handles itself.
func allKeys() []process.Datum {
	keys := []process.Datum{process.TermEnter, process.TermBackspace}
	for r := rune(32); r < 127; r++ {
		keys = append(keys, process.Chars(string(r)))
	}
	return keys
}

// invalidKeys returns the keys the keyboard does not offer right now.
func invalidKeys(term *terminal.T) []process.Datum {
	var out []process.Datum
	for _, k := range allKeys() {
		if !containsKey(term.State.ValidKeys, k) {
			out = append(out, k)
		}
	}
	return out
}

// observe is everything a keystroke could change: the screen, the
// session, the game's records and quests, and every machine's files.
func observe(sess *Session, term *terminal.T) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "mode=%v user=%q\nstate=%#v\n", sess.Mode, sess.Username, term.State)
	if sess.Shell != nil {
		fmt.Fprintf(&sb, "shell=%s:%v %q\n", sess.Shell.Hostname(), sess.Shell.CurrentDirectory(), sess.Shell.CurrentCommand())
	}
	if g := sess.Game; g != nil {
		fmt.Fprintf(&sb, "tracker=%+v\n", g.Manager.Tracker.Records())
		for _, q := range g.Manager.Quests() {
			fmt.Fprintf(&sb, "quest %s=%v\n", q.ID(), g.Manager.GetQuestState(q.ID()))
		}
		for _, h := range []string{"laptop", "nixy", "server"} {
			if c, err := g.Sim.GetComputer(h); err == nil {
				fmt.Fprintf(&sb, "host %s:\n", h)
				dumpFS(&sb, c.Filesystem, "/")
			}
		}
	}
	return sb.String()
}

func dumpFS(sb *strings.Builder, f *file.F, path string) {
	fmt.Fprintf(sb, "  %s %v %s %v/%v %q\n", path, f.Type, f.Owner, f.OwnerPermission, f.CommonPermission, f.Data)
	names := make([]string, 0, len(f.Files))
	for n := range f.Files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		dumpFS(sb, f.Files[n], strings.TrimSuffix(path, "/")+"/"+n)
	}
}

// assertRejectedNoop presses key (which the keyboard doesn't offer) and
// fails if anything observable changed.
func assertRejectedNoop(t *testing.T, sess *Session, term *terminal.T, key process.Datum) {
	t.Helper()
	before := observe(sess, term)
	sess.HandleKeystroke(key, term)
	if after := observe(sess, term); after != before {
		t.Fatalf("rejected key %s at %q changed state:\nbefore:\n%s\nafter:\n%s", keyName(key), term.State.Line, before, after)
	}
}

// A rejected Enter used to be recorded as a command: `ls |` + Enter
// completed the Composition quest without running anything.
func TestRejectedEnterIsNotACommand(t *testing.T) {
	sess, term := buildScenario(t, keyboardScenario{name: "composition", untilQuest: "composition"})
	typeAccepted(t, sess, term, "ls |")
	if containsKey(term.State.ValidKeys, process.TermEnter) {
		t.Fatal("Enter should not be offered after `ls |`")
	}
	assertRejectedNoop(t, sess, term, process.TermEnter)
}

func TestRejectedLoginKeysDoNothing(t *testing.T) {
	sess, err := New()
	if err != nil {
		t.Fatal(err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)
	for _, k := range invalidKeys(term) {
		assertRejectedNoop(t, sess, term, k)
	}
	typeAccepted(t, sess, term, "abcdefgh")
	for _, k := range invalidKeys(term) {
		assertRejectedNoop(t, sess, term, k)
	}
}

// TestFuzzE2ERejectedKeys plays randomly (valid keys, often the hint)
// and, before each valid key, presses a few keys the keyboard doesn't
// offer. None of them may change anything a player or quest could see.
// It keeps going through logouts and logins and nx panics.
func TestFuzzE2ERejectedKeys(t *testing.T) {
	const (
		runs     = 10
		steps    = 600
		rejected = 2 // invalid presses per step
	)
	for seed := int64(0); seed < runs; seed++ {
		rng := rand.New(rand.NewSource(seed))
		sess, err := New()
		if err != nil {
			t.Fatal(err)
		}
		term := terminal.New(terminal.NewANSI())
		term.Resize(80, 24)
		sess.InitTerminal(term)
		for step := 0; step < steps; step++ {
			bad := invalidKeys(term)
			for i := 0; i < rejected && len(bad) > 0; i++ {
				assertRejectedNoop(t, sess, term, bad[rng.Intn(len(bad))])
			}
			valid := term.State.ValidKeys
			if len(valid) == 0 {
				t.Fatalf("seed %d step %d: no valid keys at %q", seed, step, term.State.Line)
			}
			d := valid[rng.Intn(len(valid))]
			if hint := term.State.HintKey; hint != nil && rng.Intn(2) == 0 {
				d = hint
			}
			pressAccepted(t, sess, term, d)
		}
	}
}
