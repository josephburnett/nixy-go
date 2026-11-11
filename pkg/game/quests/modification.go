package quests

import (
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
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
	if fileExists(sim, "nixy", junk) {
		return nil
	}
	return c.Filesystem.CreateFile([]string{"home", "nixy"}, "junk.txt", &file.F{
		Type:             file.Text,
		Owner:            "nixy",
		OwnerPermission:  file.Write,
		CommonPermission: file.Read,
		Data:             "This file is junk. Please delete me!\n",
	}, "root")
}

var (
	junk      = []string{"home", "nixy", "junk.txt"}
	important = []string{"home", "nixy", "important.txt"}
)

// IsComplete: the file junk.txt is gone and important.txt exists.
func (q *Modification) IsComplete(sim *simulation.S, _ *game.CommandTracker) bool {
	return !isFile(sim, "nixy", junk) && fileExists(sim, "nixy", important)
}

func (q *Modification) PlanNextCommand(ctx game.PlanContext) string {
	if nav := planNavigate(ctx, "nixy"); nav != "" {
		return nav
	}
	if isFile(ctx.Sim, "nixy", junk) {
		return firstOf(need(ctx, "rm"), planCd(ctx, nixyHome), "rm junk.txt")
	}
	if !fileExists(ctx.Sim, "nixy", important) {
		return firstOf(need(ctx, "touch"), planCd(ctx, nixyHome), "touch important.txt")
	}
	return ""
}

// Repair puts Nixy's home back, as the quest's files live there.
func (q *Modification) Repair(sim *simulation.S, _ *game.CommandTracker, _ string) []string {
	return restore(sim, "nixy", worlds.Nixy(), nixyHome, false)
}
