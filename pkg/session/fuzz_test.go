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
// It does not demand quest progress: random play can legitimately
// strand a quest (e.g. by deleting the file it asks to read).
func TestFuzzE2ERandomKeys(t *testing.T) {
	const (
		runs  = 30
		steps = 2000
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
		typeCommand(t, sess, term, "user")
		tap := &stderrTap{P: sess.Shell.(process.P)}
		tapped := guide.New(tap)
		sess.Guide = tapped

		for step := 0; step < steps; step++ {
			valid := term.State.ValidKeys
			if len(valid) == 0 {
				t.Fatalf("seed %d step %d: no valid keys at %q", seed, step, term.State.Line)
			}
			var d process.Datum
			if hint := term.State.HintKey; hint != nil && rng.Intn(2) == 0 {
				d = hint
			} else {
				d = valid[rng.Intn(len(valid))]
			}
			line := term.State.Line
			if !containsKey(valid, d) {
				continue // random play can break a plan (e.g. delete its file), leaving a stale hint
			}
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
				break // exit on the laptop logs out
			}
			if sess.Guide != tapped {
				// nx panic replaced the shell stack; keep watching stderr.
				tap = &stderrTap{P: sess.Shell.(process.P)}
				tapped = guide.New(tap)
				sess.Guide = tapped
			}
		}
	}
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
