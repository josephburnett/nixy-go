package session

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/game"
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

func typeString(t *testing.T, s *Session, term *terminal.T, str string) {
	t.Helper()
	for _, r := range str {
		pressKey(t, s, term, process.Chars(string(r)))
	}
}

func enter(t *testing.T, s *Session, term *terminal.T) {
	t.Helper()
	pressKey(t, s, term, process.TermEnter)
}

// fuzzLogin generates a 1-8 char username that satisfies the login filter.
func fuzzLogin(rng *rand.Rand) string {
	n := 1 + rng.Intn(8)
	out := make([]byte, n)
	for i := range out {
		out[i] = byte('a' + rng.Intn(26))
	}
	return string(out)
}

// TestFuzzE2EHintGuided drives keystrokes through Session.HandleKeystroke
// (the same chokepoint used by both CLI and web), 80% following the planner
// and 20% running off-plan. After every step it asserts:
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
	typeString(t, sess, term, name)
	enter(t, sess, term)
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

		typeString(t, sess, term, cmd)
		enter(t, sess, term)

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
