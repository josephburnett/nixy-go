package quests

import (
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

func (q *Connect) IsComplete(_ *simulation.S, tracker *game.CommandTracker) bool {
	return tracker.HasCommandOnHost("nixy") || tracker.HasCommand("laptop", "ssh nixy")
}

func (q *Connect) PlanNextCommand(_ *simulation.S, _ *game.CommandTracker, hostname string, _ []string) string {
	if hostname == "nixy" {
		return ""
	}
	return "ssh nixy"
}
