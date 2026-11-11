package quests

import (
	"strconv"

	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Processes: Nixy left a yes running. The player finds it with ps -e,
// tries kill (not permitted: it's Nixy's), and kills it with sudo.
type Processes struct{}

func (q *Processes) ID() string          { return "processes" }
func (q *Processes) Description() string { return "Find and stop Nixy's runaway process" }
func (q *Processes) Machine() string     { return "nixy" }
func (q *Processes) RequiredAchievements() []game.Achievement {
	return []game.Achievement{"interrupter"}
}
func (q *Processes) GrantedAchievements() []game.Achievement { return []game.Achievement{"sysadmin"} }

// Setup starts Nixy's yes on her machine, in her home.
func (q *Processes) Setup(sim *simulation.S) error {
	if runaway(sim) != 0 {
		return nil
	}
	_, err := sim.Launch("nixy", "nixy", "yes", nil, nixyHome)
	return err
}

// runaway is the PID of Nixy's yes on her machine, 0 when there's none.
func runaway(sim *simulation.S) int {
	c, err := sim.GetComputer("nixy")
	if err != nil {
		return 0
	}
	for _, p := range c.Processes() {
		if p.Owner == "nixy" && p.Command == "yes" {
			return p.PID
		}
	}
	return 0
}

// IsComplete: Nixy's yes is gone.
func (q *Processes) IsComplete(sim *simulation.S, _ *game.CommandTracker) bool {
	return runaway(sim) == 0
}

// listedAll reports whether the player listed every process on nixy.
func listedAll(t *game.CommandTracker) bool {
	return t.AnyInvocation("nixy", game.CommandRecord.OK, func(inv game.Invocation) bool {
		return inv.Name == "ps" && len(inv.Args) > 0 && inv.Args[0] == "-e"
	})
}

// triedKill reports whether the player tried kill without sudo on
// nixy, and was refused.
func triedKill(t *game.CommandTracker) bool {
	return t.AnyInvocation("nixy", game.CommandRecord.Failed, func(inv game.Invocation) bool {
		return inv.Name == "kill" && !inv.Sudo
	})
}

func (q *Processes) PlanNextCommand(ctx game.PlanContext) string {
	pid := runaway(ctx.Sim)
	if pid == 0 {
		return ""
	}
	if nav := planNavigate(ctx, "nixy"); nav != "" {
		return nav
	}
	if !listedAll(ctx.Tracker) {
		return firstOf(need(ctx, "ps"), "ps -e")
	}
	kill := "kill " + strconv.Itoa(pid)
	if !triedKill(ctx.Tracker) {
		// See the refusal first: it's Nixy's process, not yours.
		return firstOf(need(ctx, "kill"), kill)
	}
	return firstOf(need(ctx, "sudo"), need(ctx, "kill"), "sudo "+kill)
}
