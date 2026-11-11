package quests

import (
	"slices"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

type Composition struct{}

func (q *Composition) ID() string          { return "composition" }
func (q *Composition) Description() string { return "Use pipes to find a file" }
func (q *Composition) Machine() string     { return "nixy" }

func (q *Composition) RequiredAchievements() []game.Achievement {
	return []game.Achievement{"modifier"}
}

func (q *Composition) GrantedAchievements() []game.Achievement {
	return []game.Achievement{"composer", "server-unlocked"}
}

var projects = []string{"home", "nixy", "projects"}

// projectsWorld is Nixy's world with her projects folder in it, for
// Setup and Repair.
func projectsWorld() *file.F {
	w := worlds.Nixy()
	w.Files["home"].Files["nixy"].Files["projects"] = &file.F{
		Type: file.Folder, Owner: "nixy",
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Files: map[string]*file.F{
			"notes.txt":  {Type: file.Text, Owner: "nixy", OwnerPermission: file.Write, CommonPermission: file.Read, Data: "some notes\n"},
			"draft.txt":  {Type: file.Text, Owner: "nixy", OwnerPermission: file.Write, CommonPermission: file.Read, Data: "draft\n"},
			"target.txt": {Type: file.Text, Owner: "nixy", OwnerPermission: file.Write, CommonPermission: file.Read, Data: "you found it!\n"},
			"readme.txt": {Type: file.Text, Owner: "nixy", OwnerPermission: file.Write, CommonPermission: file.Read, Data: "readme\n"},
			"config.txt": {Type: file.Text, Owner: "nixy", OwnerPermission: file.Write, CommonPermission: file.Read, Data: "config\n"},
		},
	}
	return w
}

func (q *Composition) Setup(sim *simulation.S) error {
	restore(sim, "nixy", projectsWorld(), projects, true)
	return nil
}

// IsComplete: a pipeline into grep found something on nixy.
func (q *Composition) IsComplete(_ *simulation.S, tracker *game.CommandTracker) bool {
	return tracker.Any("nixy", func(r game.CommandRecord) bool {
		invs := r.Invocations()
		return r.OK() && len(invs) > 1 && slices.ContainsFunc(invs[1:], func(inv game.Invocation) bool {
			return inv.Name == "grep"
		})
	})
}

func (q *Composition) PlanNextCommand(ctx game.PlanContext) string {
	if nav := planNavigate(ctx, "nixy"); nav != "" {
		return nav
	}
	return firstOf(need(ctx, "ls"), need(ctx, "grep"), planCd(ctx, projects), "ls | grep target")
}

// Repair puts back the projects folder and any of its files, so there
// is a target.txt to find.
func (q *Composition) Repair(sim *simulation.S, _ *game.CommandTracker, _ string) []string {
	return restore(sim, "nixy", projectsWorld(), projects, true)
}
