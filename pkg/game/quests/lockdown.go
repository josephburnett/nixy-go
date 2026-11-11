package quests

import (
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Lockdown follows Permissions: Nixy, having seen what root can do on
// the server, closes her home to everyone else. Deleting a file means
// writing the folder it's in, so the player's plain rm is now denied,
// and sudo is the way back in.
type Lockdown struct{}

func (q *Lockdown) ID() string          { return "lockdown" }
func (q *Lockdown) Description() string { return "Use sudo in Nixy's locked home" }
func (q *Lockdown) Machine() string     { return "nixy" }
func (q *Lockdown) RequiredAchievements() []game.Achievement {
	return []game.Achievement{"admin"}
}
func (q *Lockdown) GrantedAchievements() []game.Achievement {
	return []game.Achievement{"trusted"}
}

var oldFile = []string{"home", "nixy", "old.txt"}

// Setup locks Nixy's home (others may read it, not change it) and
// leaves old.txt in it to delete.
func (q *Lockdown) Setup(sim *simulation.S) error {
	q.Repair(sim, nil, "")
	c, err := sim.GetComputer("nixy")
	if err != nil {
		return err
	}
	if fileExists(sim, "nixy", oldFile) {
		return nil
	}
	return c.Filesystem.CreateFile(nixyHome, "old.txt", &file.F{
		Type: file.Text, Owner: "nixy",
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Data: "Old notes. I don't need these any more.",
	}, file.OwnerRoot)
}

// IsComplete: old.txt is gone.
func (q *Lockdown) IsComplete(sim *simulation.S, _ *game.CommandTracker) bool {
	return !fileExists(sim, "nixy", oldFile)
}

func (q *Lockdown) PlanNextCommand(ctx game.PlanContext) string {
	if nav := planNavigate(ctx, "nixy"); nav != "" {
		return nav
	}
	if !triedRm(ctx.Tracker) {
		// See the lock first: Permission denied is the lesson.
		return firstOf(need(ctx, "rm"), planCd(ctx, nixyHome), "rm old.txt")
	}
	return firstOf(need(ctx, "sudo"), need(ctx, "rm"), planCd(ctx, nixyHome), "sudo rm old.txt")
}

// triedRm reports whether the player tried to delete old.txt without
// sudo, and was refused.
func triedRm(t *game.CommandTracker) bool {
	return t.AnyInvocation("nixy", game.CommandRecord.Failed, func(inv game.Invocation) bool {
		return inv.Name == "rm" && !inv.Sudo && inv.HasPath(0, oldFile)
	})
}

// Repair keeps Nixy's home there and locked.
func (q *Lockdown) Repair(sim *simulation.S, _ *game.CommandTracker, _ string) []string {
	lines := restore(sim, "nixy", worlds.Nixy(), nixyHome, false)
	if home, ok := lookup(sim, "nixy", nixyHome); ok {
		home.CommonPermission = file.Read
	}
	return lines
}
