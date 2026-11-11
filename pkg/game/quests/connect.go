package quests

import (
	"slices"

	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

type Connect struct{}

func (q *Connect) ID() string          { return "connect" }
func (q *Connect) Description() string { return "Connect to Nixy using SSH" }
func (q *Connect) Machine() string     { return "laptop" }

func (q *Connect) RequiredAchievements() []game.Achievement { return nil }
func (q *Connect) GrantedAchievements() []game.Achievement {
	return []game.Achievement{"connected-to-nixy"}
}

func (q *Connect) Setup(_ *simulation.S) error { return nil }

// IsComplete: the player got a shell on nixy.
func (q *Connect) IsComplete(_ *simulation.S, tracker *game.CommandTracker) bool {
	return tracker.HasCommandOnHost("nixy") ||
		tracker.AnyInvocation("laptop", func(r game.CommandRecord) bool { return r.Running || r.OK() },
			func(inv game.Invocation) bool {
				return inv.Name == "ssh" && slices.Equal(inv.Args, []string{"nixy"})
			})
}

func (q *Connect) PlanNextCommand(ctx game.PlanContext) string {
	return planNavigate(ctx, "nixy")
}
