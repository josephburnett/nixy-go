package quests

import (
	"slices"

	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
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

var etcConfig = []string{"etc", "config"}

func (q *Permissions) IsComplete(sim *simulation.S, _ *game.CommandTracker) bool {
	return fileExists(sim, "server", etcConfig)
}

func (q *Permissions) PlanNextCommand(ctx game.PlanContext) string {
	if nav := planNavigate(ctx, "server"); nav != "" {
		return nav
	}
	if !runnable(ctx.Sim, "server", ctx.User, "touch") {
		if !canSudoApt(ctx) || binaryInstalled(ctx.Sim, "server", "touch") {
			return "nx panic"
		}
		// Try it as yourself first: apt's "are you root?" is the
		// lesson that makes sudo make sense.
		if !triedAptInstall(ctx.Tracker, "touch") && runnable(ctx.Sim, "server", ctx.User, "apt") {
			return "apt install touch"
		}
		return "sudo apt install touch"
	}
	return firstOf(need(ctx, "sudo"), "sudo touch /etc/config")
}

// triedAptInstall reports whether the player ran apt install pkg on
// the server without sudo, and it failed.
func triedAptInstall(t *game.CommandTracker, pkg string) bool {
	return t.AnyInvocation("server", game.CommandRecord.Failed, func(inv game.Invocation) bool {
		return inv.Name == "apt" && !inv.Sudo && len(inv.Args) > 1 &&
			inv.Args[0] == "install" && slices.Contains(inv.Args[1:], pkg)
	})
}

// Repair puts /etc back, for the config to go in.
func (q *Permissions) Repair(sim *simulation.S, _ *game.CommandTracker) []string {
	return restore(sim, "server", worlds.Server(), []string{"etc"}, false)
}
