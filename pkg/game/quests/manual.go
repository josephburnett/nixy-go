package quests

import (
	"slices"

	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Manual: Nixy wants to count her log's errors, which takes wc. The
// player installs it (as on the server) and reads its manual page, so
// they learn to look things up instead of being told.
type Manual struct{}

func (q *Manual) ID() string                               { return "manual" }
func (q *Manual) Description() string                      { return "Install wc and read its manual" }
func (q *Manual) Machine() string                          { return "nixy" }
func (q *Manual) RequiredAchievements() []game.Achievement { return []game.Achievement{"writer"} }
func (q *Manual) GrantedAchievements() []game.Achievement  { return []game.Achievement{"reader"} }
func (q *Manual) Setup(_ *simulation.S) error              { return nil }

// readManual reports whether man showed the page for page on nixy.
func readManual(t *game.CommandTracker, page string) bool {
	return t.AnyInvocation("nixy", game.CommandRecord.OK, func(inv game.Invocation) bool {
		return inv.Name == "man" && slices.Contains(inv.Args, page)
	})
}

func (q *Manual) IsComplete(_ *simulation.S, tracker *game.CommandTracker) bool {
	return readManual(tracker, "wc")
}

func (q *Manual) PlanNextCommand(ctx game.PlanContext) string {
	if nav := planNavigate(ctx, "nixy"); nav != "" {
		return nav
	}
	return firstOf(need(ctx, "wc"), need(ctx, "man"), "man wc")
}
