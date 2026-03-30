package quests

import (
	"github.com/josephburnett/nixy-go/pkg/game"
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

func (q *Orientation) IsComplete(_ *simulation.S, tracker *game.CommandTracker) bool {
	return tracker.HasCommand("nixy", "pwd") &&
		tracker.HasCommandPrefix("nixy", "ls") &&
		tracker.HasVisitedDir("nixy", []string{"home", "nixy"})
}

func (q *Orientation) PlanNextCommand(_ *simulation.S, tracker *game.CommandTracker, hostname string, cwd []string) string {
	nav := planNavigate(hostname, "nixy", cwd)
	if nav != "" {
		return nav
	}

	if !tracker.HasCommand("nixy", "pwd") {
		return "pwd"
	}
	if !tracker.HasCommandPrefix("nixy", "ls") {
		return "ls"
	}
	if !tracker.HasVisitedDir("nixy", []string{"home", "nixy"}) {
		return "cd /home/nixy"
	}
	return ""
}
