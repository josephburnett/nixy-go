package quests

import (
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Identity follows Lockdown, where the player became root with sudo:
// whoami shows who the system thinks they are, and sudo whoami that
// sudo makes them root.
type Identity struct{}

func (q *Identity) ID() string { return "identity" }
func (q *Identity) Description() string {
	return "Find out who you are, and who sudo makes you"
}
func (q *Identity) Machine() string                          { return "nixy" }
func (q *Identity) RequiredAchievements() []game.Achievement { return []game.Achievement{"trusted"} }
func (q *Identity) GrantedAchievements() []game.Achievement  { return []game.Achievement{"self-aware"} }
func (q *Identity) Setup(_ *simulation.S) error              { return nil }

// IsComplete: whoami ran fine on nixy, as the player and under sudo.
func (q *Identity) IsComplete(_ *simulation.S, tracker *game.CommandTracker) bool {
	return ranOKAs(tracker, "nixy", "whoami", false) && ranOKAs(tracker, "nixy", "whoami", true)
}

func (q *Identity) PlanNextCommand(ctx game.PlanContext) string {
	if nav := planNavigate(ctx, "nixy"); nav != "" {
		return nav
	}
	if !ranOKAs(ctx.Tracker, "nixy", "whoami", false) {
		return firstOf(need(ctx, "whoami"), "whoami")
	}
	return firstOf(need(ctx, "sudo"), need(ctx, "whoami"), "sudo whoami")
}
