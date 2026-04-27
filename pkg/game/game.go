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

	return &Game{
		Sim:     sim,
		Manager: mgr,
	}, nil
}

// AfterCommand should be called after each command executes.
func (g *Game) AfterCommand() {
	g.Manager.AfterCommand(g.Sim)
}

// GetHint returns the planner hint for the current quest.
func (g *Game) GetHint(hostname string, cwd []string, partialLine string) process.Datum {
	active := g.Manager.ActiveQuest()
	return PlanHint(active, g.Sim, g.Manager.Tracker, hostname, cwd, partialLine)
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
	return active.PlanNextCommand(g.Sim, g.Manager.Tracker, hostname, cwd)
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

// NxPanic disconnects to laptop and ensures /bin/apt on all machines.
func (g *Game) NxPanic() string {
	// Ensure /bin/apt exists on all booted machines
	for _, e := range g.Manager.Machines().AllEntries() {
		if !g.Manager.Machines().IsBooted(e.Hostname) {
			continue
		}
		c, err := g.Sim.GetComputer(e.Hostname)
		if err != nil {
			continue
		}
		binDir, err := c.Filesystem.Navigate([]string{"bin"})
		if err != nil {
			continue
		}
		if _, exists := binDir.Files["apt"]; !exists {
			binDir.Files["apt"] = &file.F{
				Type:             file.Binary,
				Owner:            file.OwnerRoot,
				OwnerPermission:  file.Write,
				CommonPermission: file.Read,
				Data:             "apt",
			}
		}
	}
	return "Panic! Restoring system state...\n"
}
