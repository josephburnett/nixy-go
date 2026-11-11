package quests

import (
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
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

var (
	readme    = []string{"home", "nixy", "readme.txt"}
	logDir    = []string{"var", "log"}
	systemLog = []string{"var", "log", "system.log"}
)

// IsComplete: the player read readme.txt and searched the system log,
// however they spelled the paths. A grep that found nothing, or a
// command that failed, doesn't count.
func (q *Inspection) IsComplete(_ *simulation.S, tracker *game.CommandTracker) bool {
	return readOK(tracker, "nixy", readme) && searchedLog(tracker)
}

// searchedLog reports whether a grep succeeded on a command line that
// read the system log (`grep error system.log`, or the log piped into
// grep).
func searchedLog(t *game.CommandTracker) bool {
	return t.Any("nixy", func(r game.CommandRecord) bool {
		if !r.OK() || !r.Invokes("grep") {
			return false
		}
		for _, inv := range r.Invocations() {
			if reads(inv, systemLog) {
				return true
			}
		}
		return false
	})
}

func (q *Inspection) PlanNextCommand(ctx game.PlanContext) string {
	if nav := planNavigate(ctx, "nixy"); nav != "" {
		return nav
	}
	if !readOK(ctx.Tracker, "nixy", readme) {
		return firstOf(need(ctx, "cat"), planCd(ctx, nixyHome), "cat readme.txt")
	}
	return firstOf(need(ctx, "grep"), planCd(ctx, logDir), "grep error system.log")
}

// Repair puts back whichever of the two files is still to be read.
func (q *Inspection) Repair(sim *simulation.S, tracker *game.CommandTracker, _ string) []string {
	var lines []string
	if !readOK(tracker, "nixy", readme) {
		lines = append(lines, restore(sim, "nixy", worlds.Nixy(), readme, false)...)
	}
	if !searchedLog(tracker) {
		lines = append(lines, restore(sim, "nixy", worlds.Nixy(), systemLog, false)...)
	}
	return lines
}
