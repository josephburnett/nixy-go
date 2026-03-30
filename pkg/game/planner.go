package game

import (
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// PlanHint returns the next Datum the player should type to advance toward
// quest completion. If the player has typed something off-plan, returns
// TermBackspace. If the player's partial line matches the plan, returns
// the next character of the target command. Returns nil if no hint is available.
func PlanHint(
	quest Quest,
	sim *simulation.S,
	tracker *CommandTracker,
	hostname string,
	cwd []string,
	partialLine string,
) process.Datum {
	if quest == nil {
		return nil
	}

	target := quest.PlanNextCommand(sim, tracker, hostname, cwd)
	if target == "" {
		return nil
	}

	if partialLine == target {
		return process.TermEnter
	}

	if len(partialLine) < len(target) && target[:len(partialLine)] == partialLine {
		// Player is on-plan, return next character
		return process.Chars(string(target[len(partialLine)]))
	}

	// Player is off-plan, suggest backspace
	if len(partialLine) > 0 {
		return process.TermBackspace
	}

	// Empty line, return first char of target
	if len(target) > 0 {
		return process.Chars(string(target[0]))
	}

	return nil
}
