package quests

import (
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Interrupt teaches Ctrl+C: run yes, which never ends, and stop it.
// The planner plans `yes`; once it runs, the game plans
// game.Interrupt (see Game.GetPlannedCommand).
type Interrupt struct{}

func (q *Interrupt) ID() string                               { return "interrupt" }
func (q *Interrupt) Description() string                      { return "Stop a command that never ends" }
func (q *Interrupt) Machine() string                          { return "nixy" }
func (q *Interrupt) RequiredAchievements() []game.Achievement { return []game.Achievement{"counter"} }
func (q *Interrupt) GrantedAchievements() []game.Achievement {
	return []game.Achievement{"interrupter"}
}
func (q *Interrupt) Setup(_ *simulation.S) error { return nil }

// IsComplete: a yes on nixy ended by Ctrl+C.
func (q *Interrupt) IsComplete(_ *simulation.S, tracker *game.CommandTracker) bool {
	return tracker.Any("nixy", func(r game.CommandRecord) bool {
		return !r.Running && r.Status == process.SigInt.Status() && r.Invokes("yes")
	})
}

func (q *Interrupt) PlanNextCommand(ctx game.PlanContext) string {
	if nav := planNavigate(ctx, "nixy"); nav != "" {
		return nav
	}
	return firstOf(need(ctx, "yes"), "yes")
}
