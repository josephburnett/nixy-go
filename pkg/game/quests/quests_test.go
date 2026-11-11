package quests

import (
	"math/rand"
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

	"github.com/josephburnett/nixy-go/pkg/character"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
	"github.com/josephburnett/nixy-go/pkg/process"
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

// shellState tracks the current shell context through SSH chains.
type shellState struct {
	game     *game.Game
	shell    process.P // current foreground shell
	hostname string
	cwd      []string
	shells   []shellFrame // SSH stack
}

type shellFrame struct {
	shell    process.P
	hostname string
	cwd      []string
}

func newShellState(t *testing.T, g *game.Game) *shellState {
	t.Helper()
	p, err := g.Sim.Launch("laptop", "user", "shell", []string{"-l"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &shellState{
		game:     g,
		shell:    p,
		hostname: "laptop",
		cwd:      file.Home("user"),
	}
}

func (s *shellState) executeCommand(t *testing.T, cmd string) {
	t.Helper()
	rec := game.CommandRecord{Hostname: s.hostname, User: "user", Dir: s.cwd, Cwd: s.cwd, Command: cmd}
	record := func(status int) {
		rec.Status = status
		s.game.Manager.Tracker.Record(rec)
		s.game.AfterCommand()
	}

	// Handle builtins directly (bypass shell for reliability)
	if strings.HasPrefix(cmd, "cd ") {
		s.handleCd(cmd)
		rec.Cwd = s.cwd
		record(0)
		return
	}
	if cmd == "exit" {
		s.handleExit(t)
		record(0)
		return
	}
	if cmd == "nx panic" {
		s.game.NxPanic()
		s.game.TakePanic()
		for len(s.shells) > 0 {
			s.handleExit(t)
		}
		record(0)
		return
	}
	if strings.HasPrefix(cmd, "ssh ") {
		targetHost := strings.TrimPrefix(cmd, "ssh ")
		s.shells = append(s.shells, shellFrame{shell: s.shell, hostname: s.hostname, cwd: s.cwd})
		remoteShell, err := s.game.Sim.Launch(targetHost, "user", "shell", []string{"-l"}, nil)
		if err != nil {
			t.Fatalf("ssh failed: %v", err)
		}
		s.shell = remoteShell
		s.hostname = targetHost
		s.cwd = file.Home("user")
		rec.Running = true
		record(0)
		return
	}
	// Parse and execute command through the simulation directly
	fields := strings.Fields(cmd)
	name := fields[0]
	args := fields[1:]
	status := 0
	if name == "sudo" && len(args) > 0 {
		status = s.launchAndDrain(args[0], args[1:], "root")
	} else if strings.Contains(cmd, "|") {
		// Handle pipe: execute each segment, pipe data
		segments := strings.Split(cmd, "|")
		var data string
		for i, seg := range segments {
			seg = strings.TrimSpace(seg)
			segFields := strings.Fields(seg)
			p, err := s.game.Sim.Launch(s.hostname, "user", segFields[0], segFields[1:], s.cwd)
			if err != nil {
				status = 127
				break
			}
			if i > 0 {
				p.Stdin(process.CharsData(data))
				p.CloseStdin()
			}
			data = process.Text(p.Read(), process.Stdout)
			status, _ = p.Exit()
		}
	} else {
		status = s.launchAndDrain(name, args, "user")
	}
	record(status)
}

func (s *shellState) launchAndDrain(name string, args []string, owner string) int {
	p, err := s.game.Sim.Launch(s.hostname, owner, name, args, s.cwd)
	if err != nil {
		return 127
	}
	p.Read()
	status, _ := p.Exit()
	return status
}

func (s *shellState) handleCd(cmd string) {
	target := strings.TrimPrefix(cmd, "cd ")
	if strings.HasPrefix(target, "/") {
		parts := strings.Split(strings.TrimPrefix(target, "/"), "/")
		var clean []string
		for _, p := range parts {
			if p != "" {
				clean = append(clean, p)
			}
		}
		s.cwd = clean
	} else {
		for _, p := range strings.Split(target, "/") {
			if p == ".." {
				if len(s.cwd) > 0 {
					s.cwd = s.cwd[:len(s.cwd)-1]
				}
			} else if p != "" && p != "." {
				s.cwd = append(s.cwd, p)
			}
		}
	}
}

func (s *shellState) handleExit(t *testing.T) {
	t.Helper()
	if len(s.shells) > 0 {
		frame := s.shells[len(s.shells)-1]
		s.shells = s.shells[:len(s.shells)-1]
		s.shell = frame.shell
		s.hostname = frame.hostname
		s.cwd = frame.cwd
	}
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
// dialog, in backticks, before it is planned, and that the thought
// line explains every plan. A quest must build on what has been
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
