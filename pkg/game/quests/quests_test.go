package quests

import (
	"math/rand"
	"strings"
	"testing"

	_ "github.com/josephburnett/nixy-go/pkg/command/all"
	"github.com/josephburnett/nixy-go/pkg/command/shell"

	"github.com/josephburnett/nixy-go/pkg/character"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

func allQuests() []game.Quest { return All() }

func allMachines() []game.MachineEntry {
	return []game.MachineEntry{
		{Hostname: "laptop", Filesystem: worlds.Laptop},
		{Hostname: "nixy", Filesystem: worlds.Nixy},
		{Hostname: "server", Filesystem: worlds.Server, UnlockedBy: "server-unlocked"},
	}
}

func newGame(t *testing.T) *game.Game {
	t.Helper()
	g, err := game.NewGame(allQuests(), allMachines(), "user")
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// shellState drives a real login shell on the laptop the way the
// session does (types the line, reads until it finishes, records it in
// the tracker as resolved), but without the keyboard: the planner's
// lines are typed whole, so it catches quest-logic bugs, not
// keystroke-level ones. hostname and cwd are the innermost shell's.
type shellState struct {
	game     *game.Game
	shell    process.P
	hostname string
	cwd      []string
	running  []runningLine
}

type runningLine struct {
	record int
	shell  shell.Info
}

func newShellState(t *testing.T, g *game.Game) *shellState {
	t.Helper()
	s := &shellState{game: g}
	s.login(t)
	return s
}

func (s *shellState) login(t *testing.T) {
	t.Helper()
	p, err := s.game.Sim.Login("laptop", "user")
	if err != nil {
		t.Fatal(err)
	}
	s.shell = p
	s.where()
}

func (s *shellState) info() shell.Info { return s.shell.(shell.Info) }

func (s *shellState) where() {
	s.hostname = s.info().Hostname()
	s.cwd = s.info().CurrentDirectory()
}

// executeCommand types cmd and Enter (or, for game.Interrupt, sends
// Ctrl+C), runs it to completion as the session does, and records it.
func (s *shellState) executeCommand(t *testing.T, cmd string) {
	t.Helper()
	host, dir, typedIn := s.hostname, s.cwd, s.info().Active()
	if cmd == game.Interrupt {
		_ = s.shell.Stdin(process.Data{process.SigInt})
	} else {
		for _, r := range cmd {
			_ = s.shell.Stdin(process.Data{process.Chars(string(r))})
		}
		_ = s.shell.Stdin(process.Data{process.TermEnter})
	}
	programs := game.Resolve(s.game.Sim, host, "user", cmd)
	exited := false
	for range 50 {
		outs := s.shell.Read()
		if _, done := s.shell.Exit(); done {
			exited = true
			break
		}
		if _, done := s.info().Active().LastStatus(); len(outs) == 0 && done {
			break
		}
	}
	tracker := s.game.Manager.Tracker
	if s.game.TakePanic() || exited {
		s.shell.Kill()
		for _, r := range s.running {
			tracker.Finish(r.record, process.SigHup.Status())
		}
		s.running = nil
		s.login(t)
	}
	still := s.running[:0]
	for _, r := range s.running {
		if status, done := r.shell.LastStatus(); done {
			tracker.Finish(r.record, status)
		} else {
			still = append(still, r)
		}
	}
	s.running = still
	if cmd != game.Interrupt {
		cwd := dir
		if s.info().Hostname() == host {
			cwd = s.info().CurrentDirectory()
		}
		status, done := typedIn.LastStatus()
		i := tracker.Record(game.CommandRecord{Hostname: host, User: "user", Dir: dir, Cwd: cwd,
			Command: cmd, Status: status, Running: !done, Programs: programs})
		if !done {
			s.running = append(s.running, runningLine{record: i, shell: typedIn})
		}
	}
	s.where()
	s.game.AfterCommand()
}

// followHintsToCompletion follows planner hints until the quest completes.
// Returns true if completed, false if stuck.
func followHintsToCompletion(t *testing.T, g *game.Game, ss *shellState, maxSteps int) bool {
	t.Helper()
	for step := 0; step < maxSteps; step++ {
		active := g.Manager.ActiveQuest()
		if active == nil {
			return true // All quests done
		}

		target := g.GetPlannedCommand(ss.hostname, ss.cwd)
		if target == "" {
			// Quest thinks it's done but manager hasn't noticed
			g.AfterCommand()
			if g.Manager.ActiveQuest() != active {
				continue
			}
			return false
		}

		ss.executeCommand(t, target)
	}
	return false
}

// --- Deterministic Tests ---

func TestDeterministicConnect(t *testing.T) {
	g := newGame(t)
	ss := newShellState(t, g)
	if !followHintsToCompletion(t, g, ss, 10) {
		if g.Manager.GetQuestState("connect") != game.QuestComplete {
			t.Fatal("connect quest should be complete")
		}
	}
}

// TestPlannedCommandsAreTaught plays the quest line by the planner and
// checks that every program a plan runs was introduced in Nixy's
// dialog, in backticks, before it is planned, that the thought line
// explains every plan, and that every key of every plan is on the
// on-screen keyboard. A quest must build on what has been
// taught, not leap ahead.
func TestPlannedCommandsAreTaught(t *testing.T) {
	g := newGame(t)
	ss := newShellState(t, g)
	var said strings.Builder
	for step := 0; step < 200 && g.Manager.ActiveQuest() != nil; step++ {
		for _, l := range g.Manager.Dialog.Drain() {
			said.WriteString(l + "\n")
		}
		target := g.GetPlannedCommand(ss.hostname, ss.cwd)
		if target == "" {
			t.Fatalf("quest %q has no plan", g.Manager.ActiveQuest().ID())
		}
		if game.PlanThought(target) == "" {
			t.Errorf("plan %q has no thought", target)
		}
		if target == game.Interrupt {
			if !strings.Contains(said.String(), "`^C`") {
				t.Errorf("quest %q plans ^C before Nixy has taught it", g.Manager.ActiveQuest().ID())
			}
			ss.executeCommand(t, target)
			continue
		}
		for _, r := range target {
			if !terminal.OnScreen(process.Chars(string(r))) {
				t.Errorf("plan %q has %q, which isn't on the on-screen keyboard", target, r)
			}
		}
		for _, inv := range (game.CommandRecord{Command: target}).Invocations() {
			names := []string{inv.Name}
			if inv.Sudo {
				names = append(names, "sudo")
			}
			for _, n := range names {
				if !strings.Contains(said.String(), "`"+n) {
					t.Errorf("quest %q plans %q before Nixy has taught `%s`", g.Manager.ActiveQuest().ID(), target, n)
				}
			}
		}
		ss.executeCommand(t, target)
	}
	if a := g.Manager.ActiveQuest(); a != nil {
		t.Fatalf("quest %q didn't finish", a.ID())
	}
}

func TestEveryQuestHasDialog(t *testing.T) {
	entries := character.AllDialog()
	for _, q := range All() {
		if character.FindDialog(entries, character.OnQuestActivate, q.ID()) == nil {
			t.Errorf("quest %q has no activation dialog", q.ID())
		}
		if character.FindDialog(entries, character.OnQuestComplete, q.ID()) == nil {
			t.Errorf("quest %q has no completion dialog", q.ID())
		}
	}
}

func TestDeterministicAllQuests(t *testing.T) {
	g := newGame(t)
	ss := newShellState(t, g)

	for step := 0; step < 200; step++ {
		active := g.Manager.ActiveQuest()
		if active == nil {
			// All quests done!
			for _, q := range allQuests() {
				state := g.Manager.GetQuestState(q.ID())
				if state != game.QuestComplete {
					t.Fatalf("quest %q should be complete, got %v", q.ID(), state)
				}
			}
			return
		}

		target := g.GetPlannedCommand(ss.hostname, ss.cwd)
		if target == "" {
			g.AfterCommand()
			if g.Manager.ActiveQuest() == active {
				t.Fatalf("step %d: quest %q has no plan and is not complete (host=%s cwd=%v)",
					step, active.ID(), ss.hostname, ss.cwd)
			}
			continue
		}

		t.Logf("step %d: quest=%s host=%s cwd=%v cmd=%q", step, active.ID(), ss.hostname, ss.cwd, target)
		ss.executeCommand(t, target)

	}
	t.Fatal("did not complete all quests within 200 steps")
}

// --- Fuzz Test ---

func TestFuzzQuests(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	for iteration := 0; iteration < 20; iteration++ {
		g := newGame(t)
		ss := newShellState(t, g)

		for step := 0; step < 500; step++ {
			active := g.Manager.ActiveQuest()
			if active == nil {
				break // All done
			}

			// 80% follow hint, 20% random command
			target := g.GetPlannedCommand(ss.hostname, ss.cwd)
			if target != "" && rng.Float64() < 0.8 {
				ss.executeCommand(t, target)
			} else if target != "" {
				// Execute a random valid but off-plan command
				randCmds := []string{"pwd", "ls"}
				ss.executeCommand(t, randCmds[rng.Intn(len(randCmds))])
			}
		}

		// Verify: either all quests complete, or the planner can still find a path
		active := g.Manager.ActiveQuest()
		if active != nil {
			target := g.GetPlannedCommand(ss.hostname, ss.cwd)
			if target == "" && !active.IsComplete(g.Sim, g.Manager.Tracker) {
				t.Fatalf("iteration %d: quest %q stuck — not complete and no plan", iteration, active.ID())
			}
		}
	}
}
