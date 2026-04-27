package game

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// PlanHint returns the next Datum the player should type to advance toward
// quest completion. If the player's partial line matches the plan, returns
// the next character (or Enter at exact match). If the player has gone
// off-plan, returns nil — let them run whatever command they want; the
// planner re-engages on the next empty prompt. Returns nil if no plan
// applies.
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

	// Player is off-plan. Don't shove them toward Backspace — they may be
	// running a perfectly valid command (e.g. `ls`) before getting back to
	// the plan. The plan re-engages naturally once the line is empty again.
	if len(partialLine) > 0 {
		return nil
	}

	// Empty line, return first char of target
	if len(target) > 0 {
		return process.Chars(string(target[0]))
	}

	return nil
}

// PlanThought returns the player's "internal monologue" describing what
// the next planned command will accomplish. Bridges the gap between Nixy's
// dialog and the keyboard hint.
func PlanThought(target string) string {
	p := command.Parse(target)
	if len(p.Segments) == 0 {
		return ""
	}
	if p.IsPipeline() {
		return "I need to chain some commands together"
	}
	seg := p.Segments[0]
	switch seg.Name {
	case "ssh":
		if len(seg.Args) > 0 {
			return fmt.Sprintf("I need to connect to %s", seg.Args[0])
		}
	case "exit":
		return "I need to disconnect from this machine"
	case "pwd":
		return "I need to print the current working directory"
	case "ls":
		return "I need to list files here"
	case "cd":
		if len(seg.Args) > 0 {
			return fmt.Sprintf("I need to change into %s", seg.Args[0])
		}
	case "cat":
		return "I need to read this file"
	case "grep":
		return "I need to search for a pattern"
	case "apt":
		if len(seg.Args) >= 2 && seg.Args[0] == "install" {
			return fmt.Sprintf("I need to install %s", seg.Args[1])
		}
	case "rm":
		return "I need to delete a file"
	case "touch":
		return "I need to create a file"
	case "mv":
		return "I need to move a file"
	case "sudo":
		if len(seg.Args) > 0 {
			return fmt.Sprintf("I need elevated permissions to run %s", seg.Args[0])
		}
	}
	return ""
}
