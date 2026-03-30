package quests

import (
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

type Modification struct{}

func (q *Modification) ID() string          { return "modification" }
func (q *Modification) Description() string { return "Create and delete files on Nixy" }
func (q *Modification) Machine() string     { return "nixy" }

func (q *Modification) RequiredAchievements() []game.Achievement {
	return []game.Achievement{"inspector"}
}

func (q *Modification) GrantedAchievements() []game.Achievement {
	return []game.Achievement{"modifier"}
}

func (q *Modification) Setup(sim *simulation.S) error {
	c, err := sim.GetComputer("nixy")
	if err != nil {
		return err
	}
	if fileExists(sim, "nixy", []string{"home", "nixy", "junk.txt"}) {
		return nil
	}
	return c.Filesystem.CreateFile([]string{"home", "nixy"}, "junk.txt", &file.F{
		Type:             file.Text,
		Owner:            "user",
		OwnerPermission:  file.Write,
		CommonPermission: file.Read,
		Data:             "This file is junk. Please delete me!",
	}, "root")
}

func (q *Modification) IsComplete(sim *simulation.S, _ *game.CommandTracker) bool {
	junkGone := !fileExists(sim, "nixy", []string{"home", "nixy", "junk.txt"})
	importantExists := fileExists(sim, "nixy", []string{"home", "nixy", "important.txt"})
	return junkGone && importantExists
}

func (q *Modification) PlanNextCommand(sim *simulation.S, _ *game.CommandTracker, hostname string, cwd []string) string {
	nav := planNavigate(hostname, "nixy", cwd)
	if nav != "" {
		return nav
	}

	if !binaryInstalled(sim, "nixy", "rm") {
		return "apt install rm"
	}
	if !binaryInstalled(sim, "nixy", "touch") {
		return "apt install touch"
	}
	if !pathEqual(cwd, []string{"home", "nixy"}) {
		return "cd /home/nixy"
	}
	if fileExists(sim, "nixy", []string{"home", "nixy", "junk.txt"}) {
		return "rm junk.txt"
	}
	if !fileExists(sim, "nixy", []string{"home", "nixy", "important.txt"}) {
		return "touch important.txt"
	}
	return ""
}
