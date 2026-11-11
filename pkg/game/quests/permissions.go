package quests

import (
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

type Permissions struct{}

func (q *Permissions) ID() string          { return "permissions" }
func (q *Permissions) Description() string { return "Use sudo to create a config on the server" }
func (q *Permissions) Machine() string     { return "server" }

func (q *Permissions) RequiredAchievements() []game.Achievement {
	return []game.Achievement{"server-unlocked"}
}

func (q *Permissions) GrantedAchievements() []game.Achievement {
	return []game.Achievement{"admin"}
}

func (q *Permissions) Setup(_ *simulation.S) error { return nil }

func (q *Permissions) IsComplete(sim *simulation.S, _ *game.CommandTracker) bool {
	return fileExists(sim, "server", []string{"etc", "config"})
}

func (q *Permissions) PlanNextCommand(sim *simulation.S, tracker *game.CommandTracker, hostname string, cwd []string) string {
	nav := planNavigate(hostname, "server", cwd)
	if nav != "" {
		return nav
	}

	if !binaryInstalled(sim, "server", "touch") {
		// Try it as yourself first: apt's "are you root?" is the
		// lesson that makes sudo make sense.
		if !tracker.HasCommand("server", "apt install touch") {
			return "apt install touch"
		}
		return "sudo apt install touch"
	}

	return "sudo touch /etc/config"
}
