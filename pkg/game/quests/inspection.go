package quests

import (
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

type Inspection struct{}

func (q *Inspection) ID() string          { return "inspection" }
func (q *Inspection) Description() string { return "Read Nixy's files and search the logs" }
func (q *Inspection) Machine() string     { return "nixy" }

func (q *Inspection) RequiredAchievements() []game.Achievement {
	return []game.Achievement{"oriented"}
}

func (q *Inspection) GrantedAchievements() []game.Achievement {
	return []game.Achievement{"inspector"}
}

func (q *Inspection) Setup(_ *simulation.S) error { return nil }

func (q *Inspection) IsComplete(_ *simulation.S, tracker *game.CommandTracker) bool {
	return tracker.HasCommand("nixy", "cat readme.txt") &&
		tracker.HasCommandPrefix("nixy", "grep")
}

func (q *Inspection) PlanNextCommand(sim *simulation.S, tracker *game.CommandTracker, hostname string, cwd []string) string {
	nav := planNavigate(hostname, "nixy", cwd)
	if nav != "" {
		return nav
	}

	if !tracker.HasCommand("nixy", "cat readme.txt") {
		if !pathEqual(cwd, []string{"home", "nixy"}) {
			return "cd /home/nixy"
		}
		return "cat readme.txt"
	}

	if !binaryInstalled(sim, "nixy", "grep") {
		return "apt install grep"
	}
	if !pathEqual(cwd, []string{"var", "log"}) {
		return "cd /var/log"
	}
	return "grep error system.log"
}
