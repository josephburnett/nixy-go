package quests

import (
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/game"
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

func (q *Composition) Setup(sim *simulation.S) error {
	c, err := sim.GetComputer("nixy")
	if err != nil {
		return err
	}
	if fileExists(sim, "nixy", []string{"home", "nixy", "projects"}) {
		return nil
	}
	dir := &file.F{
		Type: file.Folder, Owner: "user",
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Files: map[string]*file.F{
			"notes.txt":  {Type: file.Text, Owner: "user", OwnerPermission: file.Write, CommonPermission: file.Read, Data: "some notes"},
			"draft.txt":  {Type: file.Text, Owner: "user", OwnerPermission: file.Write, CommonPermission: file.Read, Data: "draft"},
			"target.txt": {Type: file.Text, Owner: "user", OwnerPermission: file.Write, CommonPermission: file.Read, Data: "you found it!"},
			"readme.txt": {Type: file.Text, Owner: "user", OwnerPermission: file.Write, CommonPermission: file.Read, Data: "readme"},
			"config.txt": {Type: file.Text, Owner: "user", OwnerPermission: file.Write, CommonPermission: file.Read, Data: "config"},
		},
	}
	return c.Filesystem.CreateFile([]string{"home", "nixy"}, "projects", dir, "root")
}

func (q *Composition) IsComplete(_ *simulation.S, tracker *game.CommandTracker) bool {
	return tracker.HasPipe("nixy")
}

func (q *Composition) PlanNextCommand(sim *simulation.S, _ *game.CommandTracker, hostname string, cwd []string) string {
	nav := planNavigate(hostname, "nixy", cwd)
	if nav != "" {
		return nav
	}

	if !binaryInstalled(sim, "nixy", "grep") {
		return "apt install grep"
	}
	if !pathEqual(cwd, []string{"home", "nixy", "projects"}) {
		return "cd /home/nixy/projects"
	}
	return "ls | grep target"
}
