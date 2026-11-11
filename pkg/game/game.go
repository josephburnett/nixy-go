package game

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Game is the top-level orchestrator owning simulation and quest manager.
type Game struct {
	Sim     *simulation.S
	Manager *Manager

	panicked bool // set by NxPanic, cleared by TakePanic
}

// NewGame creates a game with the given quests and machines. The username
// is used to provision /home/<username> on every booted machine — both at
// initial boot and at later unlock-time boots (e.g. server).
func NewGame(quests []Quest, machines []MachineEntry, username string) (*Game, error) {
	sim := simulation.New()
	registry := NewMachineRegistry(machines, username)

	if err := registry.BootInitialMachines(sim); err != nil {
		return nil, err
	}

	mgr := NewManager(quests, registry)
	// Activate first quest
	mgr.AfterCommand(sim)

	g := &Game{
		Sim:     sim,
		Manager: mgr,
	}
	sim.Nx = g
	return g, nil
}

// AfterCommand should be called after each command executes.
func (g *Game) AfterCommand() {
	g.Manager.AfterCommand(g.Sim)
}

// planContext is what the active quest's planner sees for a shell on
// hostname in cwd.
func (g *Game) planContext(hostname string, cwd []string) PlanContext {
	return PlanContext{
		Sim:      g.Sim,
		Tracker:  g.Manager.Tracker,
		Hostname: hostname,
		Cwd:      cwd,
		User:     g.Manager.Machines().Username(),
	}
}

// GetHint returns the planner hint for the current quest.
func (g *Game) GetHint(hostname string, cwd []string, partialLine string) process.Datum {
	return PlanHint(g.GetPlannedCommand(hostname, cwd), partialLine)
}

// GetThought returns a natural-language description of the next planned
// command, bridging dialog and hints.
func (g *Game) GetThought(hostname string, cwd []string) string {
	return PlanThought(g.GetPlannedCommand(hostname, cwd))
}

// GetPlannedCommand returns the literal next command the planner suggests,
// or "" if no quest is active.
func (g *Game) GetPlannedCommand(hostname string, cwd []string) string {
	active := g.Manager.ActiveQuest()
	if active == nil {
		return ""
	}
	return active.PlanNextCommand(g.planContext(hostname, cwd))
}

// NxQuest returns the current quest description.
func (g *Game) NxQuest() string {
	active := g.Manager.ActiveQuest()
	if active == nil {
		return "No active quest. You've completed everything!\n"
	}
	return active.Description() + "\n"
}

// NxLog returns the adventure log.
func (g *Game) NxLog() string {
	var sb strings.Builder
	currentMachine := ""
	for _, q := range g.Manager.Quests() {
		machine := q.Machine()
		if machine != currentMachine {
			if currentMachine != "" {
				sb.WriteString("\n")
			}
			locked := ""
			if !g.Manager.Machines().IsBooted(machine) {
				locked = " (locked)"
			}
			sb.WriteString(fmt.Sprintf("== %s ==%s\n", machine, locked))
			currentMachine = machine
		}
		state := g.Manager.GetQuestState(q.ID())
		switch state {
		case QuestComplete:
			sb.WriteString(fmt.Sprintf("  ✓ %s\n", q.Description()))
		case QuestActive:
			sb.WriteString(fmt.Sprintf("  ● %s\n", q.Description()))
		case QuestInactive:
			sb.WriteString(fmt.Sprintf("    %s\n", q.Description()))
		}
	}
	return sb.String()
}

// TakePanic reports whether nx panic ran since the last call. The session
// uses it to disconnect the player back to a fresh laptop shell.
func (g *Game) TakePanic() bool {
	p := g.panicked
	g.panicked = false
	return p
}

// NxPanic puts every booted machine's /bin back as shipped, keeping
// the programs properly installed since (so a deleted ssh, apt or sudo
// can't strand the player, and neither can a program replaced by
// another), and flags the session to disconnect the player back to the
// laptop (see TakePanic). It is the recovery planners fall back on when
// sudo apt install can't fix what's broken.
func (g *Game) NxPanic() string {
	g.panicked = true
	for _, e := range g.Manager.Machines().AllEntries() {
		if !g.Manager.Machines().IsBooted(e.Hostname) {
			continue
		}
		c, err := g.Sim.GetComputer(e.Hostname)
		if err != nil {
			continue
		}
		shipped, ok := e.Filesystem().Files["bin"]
		if !ok {
			continue
		}
		if bin, ok := c.Filesystem.Files["bin"]; ok && bin.Type == file.Folder {
			for name, f := range bin.Files {
				_, isShipped := shipped.Files[name]
				if !isShipped && f.Type == file.Binary && f.Data == name {
					shipped.Files[name] = f // installed: keep it
				}
			}
		}
		c.Filesystem.Files["bin"] = shipped
	}
	return "Panic! Restoring system state...\n"
}
