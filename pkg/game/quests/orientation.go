package quests

import (
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

type Orientation struct{}

func (q *Orientation) ID() string          { return "orientation" }
func (q *Orientation) Description() string { return "Explore Nixy's filesystem" }
func (q *Orientation) Machine() string     { return "nixy" }

func (q *Orientation) RequiredAchievements() []game.Achievement {
	return []game.Achievement{"connected-to-nixy"}
}

func (q *Orientation) GrantedAchievements() []game.Achievement {
	return []game.Achievement{"oriented"}
}

func (q *Orientation) Setup(_ *simulation.S) error { return nil }

var nixyHome = []string{"home", "nixy"}

// IsComplete: pwd and ls ran fine on nixy, and the player has been in
// Nixy's home.
func (q *Orientation) IsComplete(_ *simulation.S, tracker *game.CommandTracker) bool {
	return ranOK(tracker, "nixy", "pwd") &&
		ranOK(tracker, "nixy", "ls") &&
		tracker.HasVisitedDir("nixy", nixyHome)
}

func (q *Orientation) PlanNextCommand(ctx game.PlanContext) string {
	if nav := planNavigate(ctx, "nixy"); nav != "" {
		return nav
	}
	if !ranOK(ctx.Tracker, "nixy", "pwd") {
		return firstOf(need(ctx, "pwd"), "pwd")
	}
	if !ranOK(ctx.Tracker, "nixy", "ls") {
		// The player starts in their own, empty home: /home shows
		// whose homes are here, Nixy's included.
		return firstOf(need(ctx, "ls"), "ls /home")
	}
	if !ctx.Tracker.HasVisitedDir("nixy", nixyHome) {
		return planCd(ctx, nixyHome)
	}
	return ""
}

// Repair puts Nixy's home back, so there's somewhere to visit.
func (q *Orientation) Repair(sim *simulation.S, _ *game.CommandTracker, _ string) []string {
	return restore(sim, "nixy", worlds.Nixy(), nixyHome, false)
}
