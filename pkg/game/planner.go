package game

import (
	"fmt"
	"strings"

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

// PlanThought returns the player's "internal monologue" describing what
// the next planned command will accomplish. Bridges the gap between Nixy's
// dialog and the keyboard hint.
func PlanThought(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if strings.Contains(target, "|") {
		return "I need to chain some commands together"
	}
	fields := strings.Fields(target)
	switch fields[0] {
	case "ssh":
		if len(fields) > 1 {
			return fmt.Sprintf("I need to connect to %s", fields[1])
		}
	case "exit":
		return "I need to disconnect from this machine"
	case "pwd":
		return "I need to print the current working directory"
	case "ls":
		return "I need to list files here"
	case "cd":
		if len(fields) > 1 {
			return fmt.Sprintf("I need to change into %s", fields[1])
		}
	case "cat":
		return "I need to read this file"
	case "grep":
		return "I need to search for a pattern"
	case "apt":
		if len(fields) >= 3 && fields[1] == "install" {
			return fmt.Sprintf("I need to install %s", fields[2])
		}
	case "rm":
		return "I need to delete a file"
	case "touch":
		return "I need to create a file"
	case "mv":
		return "I need to move a file"
	case "sudo":
		if len(fields) > 1 {
			return fmt.Sprintf("I need elevated permissions to run %s", fields[1])
		}
	}
	return ""
}
