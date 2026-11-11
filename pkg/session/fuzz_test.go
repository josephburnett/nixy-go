package session

import (
	"math/rand"
	"slices"
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/guide"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

const (
	fuzzIterations = 5
	fuzzMaxSteps   = 200
	fuzzStuckLimit = 20
)

// worldKey captures variables that signal real player progress.
//
// Tracker record count is deliberately NOT included: it grows on every
// Enter regardless of whether the command had any effect on the world,
// so it makes a stuck quest (e.g. rm failing silently) look like
// progress. Quest progression and shell location are the honest signals.
type worldKey struct {
	mode               Mode
	hostname           string
	cwd                string
	activeQ            string
	numCompletedQuests int
}

func snapWorld(s *Session) worldKey {
	w := worldKey{mode: s.Mode}
	if s.Shell != nil {
		w.hostname = s.Shell.Hostname()
		w.cwd = strings.Join(s.Shell.CurrentDirectory(), "/")
	}
	if s.Game != nil {
		if a := s.Game.Manager.ActiveQuest(); a != nil {
			w.activeQ = a.ID()
		}
		for _, q := range s.Game.Manager.Quests() {
			if s.Game.Manager.GetQuestState(q.ID()) == game.QuestComplete {
				w.numCompletedQuests++
			}
		}
	}
	return w
}

// snapQuests returns a sorted dump of all quest states for regression checks.
func snapQuests(s *Session) map[string]game.QuestState {
	out := map[string]game.QuestState{}
	if s.Game == nil {
		return out
	}
	for _, q := range s.Game.Manager.Quests() {
		out[q.ID()] = s.Game.Manager.GetQuestState(q.ID())
	}
	return out
}

// pressKey sends one keystroke through the session and ensures the renderer
// doesn't blow up either.
func pressKey(t *testing.T, s *Session, term *terminal.T, d process.Datum) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on keystroke %v: %v", d, r)
		}
	}()
	s.HandleKeystroke(d, term)
	_ = term.Render()
}

// fuzzLogin generates a 1-8 char username the login prompt accepts:
// lowercase letters, and not an existing account.
func fuzzLogin(rng *rand.Rand) string {
	accounts := game.Accounts(worldMachines())
	for {
		n := 1 + rng.Intn(8)
		out := make([]byte, n)
		for i := range out {
			out[i] = byte('a' + rng.Intn(26))
		}
		if !slices.Contains(accounts, string(out)) {
			return string(out)
		}
	}
}

// TestFuzzE2EHintGuided drives keystrokes through Session.HandleKeystroke
// (the same chokepoint used by both CLI and web), 80% following the planner
// and 20% running off-plan. Every keystroke must be in the keyboard's
// valid set, and the hint key must be nil or in that set (so the
// planner can never ask for a key that would be swallowed). After every
// step it asserts:
//
//   - mode never regresses (Login → Playing, never back)
//   - in ModePlaying, Game/Guide/Shell are non-nil
//   - quest states never regress (Complete never returns to Active)
//   - the renderer doesn't panic
//   - the world changes within fuzzStuckLimit consecutive Enters
//     (the hang signature: typing produces no observable state change)
func TestFuzzE2EHintGuided(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for iter := range fuzzIterations {
		t.Run("iter", func(t *testing.T) {
			runE2EFuzzIteration(t, rng, iter)
		})
	}
}

func runE2EFuzzIteration(t *testing.T, rng *rand.Rand, iter int) {
	sess, err := New()
	if err != nil {
		t.Fatalf("iter %d: New: %v", iter, err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)
	_ = term.Render()

	if sess.Mode != ModeLogin {
		t.Fatalf("iter %d: expected ModeLogin, got %v", iter, sess.Mode)
	}

	// Login phase
	name := fuzzLogin(rng)
	typeCommand(t, sess, term, name)
	if sess.Mode != ModePlaying {
		t.Fatalf("iter %d: failed to enter ModePlaying with name %q", iter, name)
	}
	if sess.Game == nil || sess.Guide == nil || sess.Shell == nil {
		t.Fatalf("iter %d: ModePlaying but Game/Guide/Shell nil", iter)
	}

	prevWorld := snapWorld(sess)
	prevQuests := snapQuests(sess)
	stuck := 0

	for step := range fuzzMaxSteps {
		if sess.Game.Manager.ActiveQuest() == nil {
			// Verify everything completed.
			for id, st := range snapQuests(sess) {
				if st != game.QuestComplete {
					t.Fatalf("iter %d step %d: no active quest but %s is %v", iter, step, id, st)
				}
			}
			return
		}

		// 80% follow plan, 20% off-plan.
		host := sess.Shell.Hostname()
		cwd := sess.Shell.CurrentDirectory()
		var cmd string
		if rng.Float64() < 0.8 {
			cmd = sess.Game.GetPlannedCommand(host, cwd)
		}
		if cmd == "" {
			randCmds := []string{"pwd", "ls"}
			cmd = randCmds[rng.Intn(len(randCmds))]
		}

		// Key by key, failing if the keyboard doesn't offer a key the
		// planner asked for, or hints a key it doesn't offer.
		typeCommand(t, sess, term, cmd)

		// Invariants
		if sess.Mode != ModePlaying {
			t.Fatalf("iter %d step %d: mode regressed to %v", iter, step, sess.Mode)
		}
		curQuests := snapQuests(sess)
		for id, prev := range prevQuests {
			cur := curQuests[id]
			if prev == game.QuestComplete && cur != game.QuestComplete {
				t.Fatalf("iter %d step %d: quest %q regressed from Complete to %v", iter, step, id, cur)
			}
			if prev == game.QuestActive && cur == game.QuestInactive {
				t.Fatalf("iter %d step %d: quest %q regressed from Active to Inactive", iter, step, id)
			}
		}
		prevQuests = curQuests

		// Liveness: world must change within fuzzStuckLimit consecutive Enters.
		cur := snapWorld(sess)
		if cur == prevWorld {
			stuck++
			if stuck >= fuzzStuckLimit {
				t.Fatalf("iter %d step %d: stuck (no world change in %d steps). cmd=%q world=%+v",
					iter, step, fuzzStuckLimit, cmd, cur)
			}
		} else {
			stuck = 0
			prevWorld = cur
		}
	}

	// Hard terminal: if any quest is still incomplete after fuzzMaxSteps,
	// the run failed. "Planner has a path" was the old (too lenient) check
	// that let a stuck quest masquerade as in-progress.
	if sess.Game.Manager.ActiveQuest() != nil {
		var unfinished []string
		for id, st := range snapQuests(sess) {
			if st != game.QuestComplete {
				unfinished = append(unfinished, id+"="+stateName(st))
			}
		}
		t.Fatalf("iter %d: did not complete all quests in %d steps; unfinished=%v",
			iter, fuzzMaxSteps, unfinished)
	}
}

func stateName(s game.QuestState) string {
	switch s {
	case game.QuestInactive:
		return "Inactive"
	case game.QuestActive:
		return "Active"
	case game.QuestComplete:
		return "Complete"
	}
	return "?"
}

// TestFuzzE2ERandomKeys presses random valid keys (half the time the
// hint, when there is one) through Session.HandleKeystroke. Unlike the
// hint-guided fuzz it wanders off-plan into destructive commands
// (rm, mv, sudo, apt) and long pipelines. It asserts:
//
//   - no panic, and the keyboard is never empty while the session runs;
//   - no command writes to stderr except a permission error.
//
// Then, from wherever random play left the world, it follows only the
// hints and demands that they finish every quest: whatever the player
// broke, the game recovers (see followHints).
func TestFuzzE2ERandomKeys(t *testing.T) {
	const (
		runs  = 30
		steps = 2000
	)
	logins := 0
	defer func() { t.Logf("%d logins", logins) }()
	for seed := int64(0); seed < runs; seed++ {
		rng := rand.New(rand.NewSource(seed))
		sess, err := New()
		if err != nil {
			t.Fatal(err)
		}
		term := terminal.New(terminal.NewANSI())
		term.Resize(80, 24)
		sess.InitTerminal(term)
		typeCommand(t, sess, term, "user")
		// Start random play at a different quest for each seed, so
		// every quest gets broken in its own way.
		skip := int(seed) % len(sess.Game.Manager.Quests())
		followHints(t, sess, term, hintFinishKeys, func() bool { return completedQuests(sess) >= skip })
		tap := &stderrTap{P: sess.Shell.(process.P)}
		tapped := guide.New(tap)
		sess.Guide = tapped

		var script []process.Datum // keys of a scripted line being typed
		for step := 0; step < steps; step++ {
			valid := term.State.ValidKeys
			if len(valid) == 0 {
				t.Fatalf("seed %d step %d: no valid keys at %q", seed, step, term.State.Line)
			}
			if hint := term.State.HintKey; hint != nil && !containsKey(valid, hint) {
				t.Fatalf("seed %d step %d: hint %q is not a valid key at %q (plan %q)",
					seed, step, keyName(hint), term.State.Line, plannedCommand(sess))
			}
			d := randomKey(rng, sess, term, &script)
			line := term.State.Line
			if !containsKey(valid, d) {
				continue // a scripted line that can't be typed here
			}
			wasLogin := sess.Mode == ModeLogin
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("seed %d step %d: panic on %q at %q: %v", seed, step, keyName(d), line, r)
					}
				}()
				sess.HandleKeystroke(d, term)
			}()
			if msg := tap.take(); msg != "" && !strings.Contains(msg, "Permission denied") {
				t.Fatalf("seed %d step %d: %q + %s wrote %q", seed, step, line, keyName(d), msg)
			}
			if sess.Mode != ModePlaying {
				continue // logged out: play on at the login prompt
			}
			if wasLogin {
				logins++
			}
			if sess.Guide != tapped {
				// nx panic or a login replaced the shell; keep watching stderr.
				tap = &stderrTap{P: sess.Shell.(process.P)}
				tapped = guide.New(tap)
				sess.Guide = tapped
			}
		}

		// Liveness: from wherever random play left things, the hints
		// alone must finish the quest line.
		followHints(t, sess, term, hintFinishKeys, func() bool {
			return sess.Mode == ModePlaying && sess.Game.Manager.ActiveQuest() == nil
		})
	}
}

// hintFinishKeys bounds how many keys followHints may press to finish
// the whole quest line, from anywhere.
const hintFinishKeys = 3000

// followHints presses only what the planner hints until done holds,
// failing if it takes more than maxKeys keys. Off-plan typing is
// erased with Backspace, and at the login prompt it logs in as "user"
// (whose game random play started). It fails when a hint is not a
// valid key, when the planner has a plan the thought line doesn't
// explain, or when there is a quest but no plan at an empty prompt:
// each of those leaves the player stranded.
func followHints(t *testing.T, sess *Session, term *terminal.T, maxKeys int, done func() bool) {
	t.Helper()
	for range maxKeys {
		if done() {
			return
		}
		valid := term.State.ValidKeys
		hint := term.State.HintKey
		line := term.State.Line
		var d process.Datum
		switch {
		case sess.Mode == ModeLogin && line != "user" && line != "":
			d = process.TermBackspace
		case sess.Mode == ModeLogin:
			d = process.TermEnter
			if line == "" {
				typeAccepted(t, sess, term, "user")
				continue
			}
		case hint != nil:
			if !containsKey(valid, hint) {
				t.Fatalf("hint %q is not a valid key at %q (plan %q)", keyName(hint), line, plannedCommand(sess))
			}
			if plannedCommand(sess) != "" && term.State.Thought == "" {
				t.Fatalf("plan %q has no thought", plannedCommand(sess))
			}
			d = hint
		case line != "":
			d = process.TermBackspace
		default:
			t.Fatalf("quest %q has no plan at an empty prompt on %s in /%s",
				activeQuestID(sess), sess.Shell.Hostname(), strings.Join(sess.Shell.CurrentDirectory(), "/"))
		}
		pressAccepted(t, sess, term, d)
	}
	t.Fatalf("hints didn't finish within %d keys: quest %q, plan %q, line %q",
		maxKeys, activeQuestID(sess), plannedCommand(sess), term.State.Line)
}

func completedQuests(sess *Session) int {
	n := 0
	for _, st := range snapQuests(sess) {
		if st == game.QuestComplete {
			n++
		}
	}
	return n
}

func plannedCommand(sess *Session) string {
	if sess.Mode != ModePlaying {
		return ""
	}
	return sess.Game.GetPlannedCommand(sess.Shell.Hostname(), sess.Shell.CurrentDirectory())
}

func activeQuestID(sess *Session) string {
	if sess.Game == nil {
		return ""
	}
	if q := sess.Game.Manager.ActiveQuest(); q != nil {
		return q.ID()
	}
	return ""
}

// randomKey picks the next key for random play: half the time the hint
// (when there is one), otherwise any valid key. Now and then, at an
// empty line, it starts typing a line that leaves (exit, ssh), panics or
// logs in again, so random play keeps going through those transitions;
// script holds the rest of that line. The key may not be valid.
func randomKey(rng *rand.Rand, sess *Session, term *terminal.T, script *[]process.Datum) process.Datum {
	if len(*script) == 0 && term.State.Line == "" && rng.Intn(8) == 0 {
		line := scriptedLines[rng.Intn(len(scriptedLines))]
		if sess.Mode == ModeLogin {
			line = []string{"user", fuzzLogin(rng)}[rng.Intn(2)]
		}
		for _, r := range line {
			*script = append(*script, process.Chars(string(r)))
		}
		*script = append(*script, process.TermEnter)
	}
	if len(*script) > 0 {
		d := (*script)[0]
		*script = (*script)[1:]
		if !containsKey(term.State.ValidKeys, d) {
			*script = nil
		}
		return d
	}
	if hint := term.State.HintKey; hint != nil && rng.Intn(2) == 0 {
		return hint
	}
	return term.State.ValidKeys[rng.Intn(len(term.State.ValidKeys))]
}

// scriptedLines are lines random play types now and then: ones that
// leave a shell, and ones that break what quests depend on (each is
// skipped where it can't be typed).
var scriptedLines = []string{
	"exit", "nx panic", "ssh nixy",
	"rm /home/nixy/readme.txt",
	"sudo rm /home/nixy/readme.txt",
	"sudo rm /var/log/system.log",
	"sudo rm /home/nixy/projects/target.txt",
	"sudo rm /bin/grep", "sudo rm /bin/cat", "sudo rm /bin/rm",
	"sudo rm /bin/touch", "sudo rm /bin/apt", "sudo rm /bin/sudo",
	"sudo apt install mv",
	"sudo mv /home/nixy/readme.txt /home/nixy/r.txt",
	"sudo mv /bin/grep /bin/g",
	"sudo mv /bin/ls /bin/cat",
}

// stderrTap wraps the player's shell and keeps a copy of its stderr.
type stderrTap struct {
	process.P
	buf strings.Builder
}

func (s *stderrTap) Read() []process.Output {
	out := s.P.Read()
	s.buf.WriteString(process.Text(out, process.Stderr))
	return out
}

func (s *stderrTap) take() string {
	out := s.buf.String()
	s.buf.Reset()
	return out
}
