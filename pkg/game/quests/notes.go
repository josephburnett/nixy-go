package quests

import (
	"strings"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Notes teaches making things: mkdir a folder in the player's home on
// nixy, write a to-do list into it with echo and `>`, add to it with
// `>>`, and cat it back.
type Notes struct{}

func (q *Notes) ID() string                               { return "notes" }
func (q *Notes) Description() string                      { return "Make a notes folder and write a to-do list" }
func (q *Notes) Machine() string                          { return "nixy" }
func (q *Notes) RequiredAchievements() []game.Achievement { return []game.Achievement{"self-aware"} }
func (q *Notes) GrantedAchievements() []game.Achievement  { return []game.Achievement{"writer"} }
func (q *Notes) Setup(_ *simulation.S) error              { return nil }

// todoPath is the to-do list in user's home.
func todoPath(user string) []string { return append(file.Home(user), "notes", "todo.txt") }

// todoLines is how many lines the to-do list on nixy has, -1 when it
// isn't a text file.
func todoLines(sim *simulation.S, user string) int {
	f, ok := lookup(sim, "nixy", todoPath(user))
	if !ok || f.Type != file.Text {
		return -1
	}
	return strings.Count(f.Data, "\n")
}

// readTodo reports whether the player read their to-do list with cat
// (and, with lines, whether it now has at least that many lines).
func readTodo(sim *simulation.S, t *game.CommandTracker, lines int) bool {
	return t.Any("nixy", func(r game.CommandRecord) bool {
		if !r.OK() || todoLines(sim, r.User) < lines {
			return false
		}
		for _, inv := range r.Invocations() {
			if inv.Name == "cat" && inv.HasPath(0, todoPath(r.User)) {
				return true
			}
		}
		return false
	})
}

// IsComplete: the player has read their to-do list, and it has two
// lines or more.
func (q *Notes) IsComplete(sim *simulation.S, tracker *game.CommandTracker) bool {
	return readTodo(sim, tracker, 2)
}

func (q *Notes) PlanNextCommand(ctx game.PlanContext) string {
	if nav := planNavigate(ctx, "nixy"); nav != "" {
		return nav
	}
	notes := append(file.Home(ctx.User), "notes")
	todo := todoPath(ctx.User)
	// Clear away anything in the way: a file called notes, or a notes
	// folder (or to-do list) the player can't write, which root made.
	if f, ok := lookup(ctx.Sim, "nixy", notes); ok && (f.Type != file.Folder || !f.CanWrite(ctx.User)) {
		return planRemove(ctx, notes)
	}
	if f, ok := lookup(ctx.Sim, "nixy", todo); ok && (f.Type != file.Text || !f.CanWrite(ctx.User)) {
		return planRemove(ctx, todo)
	}
	if !fileExists(ctx.Sim, "nixy", notes) {
		return firstOf(need(ctx, "mkdir"), planCdHome(ctx), "mkdir notes")
	}
	switch n := todoLines(ctx.Sim, ctx.User); {
	case n <= 0:
		return firstOf(need(ctx, "echo"), planCdHome(ctx, "notes"), "echo check the disk > todo.txt")
	case n == 1:
		return firstOf(need(ctx, "echo"), planCdHome(ctx, "notes"), "echo fix the network >> todo.txt")
	}
	if !readTodo(ctx.Sim, ctx.Tracker, 2) {
		return firstOf(need(ctx, "cat"), planCdHome(ctx, "notes"), "cat todo.txt")
	}
	return ""
}

// Repair puts the player's home back, for the notes to go in.
func (q *Notes) Repair(sim *simulation.S, _ *game.CommandTracker, user string) []string {
	return restoreHome(sim, "nixy", user)
}
