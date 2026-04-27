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

	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
	"github.com/josephburnett/nixy-go/pkg/process"
)

func allQuests() []game.Quest {
	return []game.Quest{
		&Connect{},
		&Orientation{},
		&Inspection{},
		&Modification{},
		&Composition{},
		&Permissions{},
	}
}

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
}

func newShellState(t *testing.T, g *game.Game) *shellState {
	t.Helper()
	p, err := g.Sim.Launch("laptop", "user", "shell", nil, []string{})
	if err != nil {
		t.Fatal(err)
	}
	return &shellState{
		game:     g,
		shell:    p,
		hostname: "laptop",
		cwd:      []string{},
	}
}

func (s *shellState) executeCommand(t *testing.T, cmd string) {
	t.Helper()

	// Track before execution
	s.game.Manager.Tracker.Record(s.hostname, s.cwd, cmd)

	// Handle builtins directly (bypass shell for reliability)
	if strings.HasPrefix(cmd, "cd ") {
		s.handleCd(cmd)
		s.game.AfterCommand()
		return
	}
	if cmd == "exit" {
		s.handleExit(t)
		s.game.AfterCommand()
		return
	}
	if strings.HasPrefix(cmd, "ssh ") {
		targetHost := strings.TrimPrefix(cmd, "ssh ")
		s.shells = append(s.shells, shellFrame{shell: s.shell, hostname: s.hostname})
		remoteShell, err := s.game.Sim.Launch(targetHost, "user", "shell", nil, []string{})
		if err != nil {
			t.Fatalf("ssh failed: %v", err)
		}
		s.shell = remoteShell
		s.hostname = targetHost
		s.cwd = []string{}
		s.game.AfterCommand()
		return
	}

	// Parse and execute command through the simulation directly
	fields := strings.Fields(cmd)
	name := fields[0]
	args := fields[1:]

	// Handle sudo
	if name == "sudo" && len(args) > 0 {
		name = args[0]
		args = args[1:]
		s.launchAndDrain(t, name, args, "root")
	} else if strings.Contains(cmd, "|") {
		// Handle pipe: execute each segment, pipe data
		segments := strings.Split(cmd, "|")
		var data process.Data
		for _, seg := range segments {
			seg = strings.TrimSpace(seg)
			segFields := strings.Fields(seg)
			p, err := s.game.Sim.Launch(s.hostname, "user", segFields[0], segFields[1:], s.cwd)
			if err != nil {
				break
			}
			if len(data) > 0 {
				p.Stdin(data)
				p.Kill()
			}
			data, _, _ = p.Stdout()
		}
	} else {
		s.launchAndDrain(t, name, args, "user")
	}

	s.game.AfterCommand()
}

func (s *shellState) launchAndDrain(t *testing.T, name string, args []string, owner string) {
	t.Helper()
	p, err := s.game.Sim.Launch(s.hostname, owner, name, args, s.cwd)
	if err != nil {
		return // Command failed, that's okay
	}
	// Drain output
	for i := 0; i < 20; i++ {
		_, eof, _ := p.Stdout()
		if eof {
			break
		}
	}
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
		s.cwd = []string{}
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

		target := active.PlanNextCommand(g.Sim, g.Manager.Tracker, ss.hostname, ss.cwd)
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

		target := active.PlanNextCommand(g.Sim, g.Manager.Tracker, ss.hostname, ss.cwd)
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
			target := active.PlanNextCommand(g.Sim, g.Manager.Tracker, ss.hostname, ss.cwd)
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
			target := active.PlanNextCommand(g.Sim, g.Manager.Tracker, ss.hostname, ss.cwd)
			if target == "" && !active.IsComplete(g.Sim, g.Manager.Tracker) {
				t.Fatalf("iteration %d: quest %q stuck — not complete and no plan", iteration, active.ID())
			}
		}
	}
}
