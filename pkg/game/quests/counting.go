package quests

import (
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/game/worlds"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Counting puts wc to work: count the system log's lines, then only
// its errors, by piping grep into wc -l.
type Counting struct{}

func (q *Counting) ID() string                               { return "counting" }
func (q *Counting) Description() string                      { return "Count the errors in the system log" }
func (q *Counting) Machine() string                          { return "nixy" }
func (q *Counting) RequiredAchievements() []game.Achievement { return []game.Achievement{"reader"} }
func (q *Counting) GrantedAchievements() []game.Achievement  { return []game.Achievement{"counter"} }
func (q *Counting) Setup(_ *simulation.S) error              { return nil }

// countsLines reports whether inv is wc -l (any spelling of the flag).
func countsLines(inv game.Invocation) bool {
	flags, _ := command.Options(inv.Args)
	return inv.Name == "wc" && strings.ContainsRune(flags, 'l')
}

// countedLog reports whether wc -l counted the system log's lines.
func countedLog(t *game.CommandTracker) bool {
	return t.AnyInvocation("nixy", game.CommandRecord.OK, func(inv game.Invocation) bool {
		_, files := command.Options(inv.Args)
		return countsLines(inv) && len(files) > 0 && inv.HasPath(len(inv.Args)-len(files), systemLog)
	})
}

// countedErrors reports whether a pipeline searched the system log and
// counted what it found with wc -l.
func countedErrors(t *game.CommandTracker) bool {
	return t.Any("nixy", func(r game.CommandRecord) bool {
		invs := r.Invocations()
		if !r.OK() || len(invs) < 2 || !countsLines(invs[len(invs)-1]) {
			return false
		}
		for _, inv := range invs[:len(invs)-1] {
			if inv.Name == "grep" && reads(inv, systemLog) {
				return true
			}
		}
		return false
	})
}

func (q *Counting) IsComplete(_ *simulation.S, tracker *game.CommandTracker) bool {
	return countedLog(tracker) && countedErrors(tracker)
}

func (q *Counting) PlanNextCommand(ctx game.PlanContext) string {
	if nav := planNavigate(ctx, "nixy"); nav != "" {
		return nav
	}
	if !countedLog(ctx.Tracker) {
		return firstOf(need(ctx, "wc"), planCd(ctx, logDir), "wc -l system.log")
	}
	return firstOf(need(ctx, "grep"), need(ctx, "wc"), planCd(ctx, logDir), "grep error system.log | wc -l")
}

// Repair puts the system log back while there's counting to do.
func (q *Counting) Repair(sim *simulation.S, _ *game.CommandTracker, _ string) []string {
	return restore(sim, "nixy", worlds.Nixy(), systemLog, false)
}
