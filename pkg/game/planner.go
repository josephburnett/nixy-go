package game

import (
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
)

// Interrupt is the plan "press Ctrl+C": stop the command running in the
// foreground. It is a key, not a command line.
const Interrupt = "^C"

// PlanHint returns the next Datum the player should type to reach the
// planned command line target. If the player's partial line matches the
// plan, it is the next character (or Enter at exact match). If the
// player has gone off-plan, it is nil: let them run whatever command
// they want; the planner re-engages on the next empty prompt. It is nil
// when there is no plan.
func PlanHint(target, partialLine string) process.Datum {
	if target == "" {
		return nil
	}
	if target == Interrupt {
		return process.SigInt
	}
	if partialLine == target {
		return process.TermEnter
	}
	if strings.HasPrefix(target, partialLine) {
		return process.Chars(string(target[len(partialLine)]))
	}
	// Off-plan. Don't shove them toward Backspace: they may be running
	// a perfectly valid command (e.g. `ls`) before getting back to the
	// plan.
	return nil
}

// PlanThought returns the player's "internal monologue" describing what
// the next planned command will accomplish. Bridges the gap between Nixy's
// dialog and the keyboard hint. Every command a planner can plan has
// one.
func PlanThought(target string) string {
	if target == Interrupt {
		return "It won't stop by itself. Ctrl+C (the ^C key) interrupts it"
	}
	p := command.Parse(target)
	if len(p.Segments) == 0 {
		return ""
	}
	if p.IsPipeline() {
		var names []string
		for _, seg := range p.Segments {
			names = append(names, seg.Name)
		}
		return "I need to send the output of " + strings.Join(names, " into ")
	}
	seg := p.Segments[0]
	if seg.Name == "sudo" && len(seg.Args) > 0 {
		inner := command.Segment{Name: seg.Args[0], Args: seg.Args[1:]}
		if what := intent(inner); what != "" {
			return "I need root to " + what
		}
		return ""
	}
	switch seg.Name {
	case "nx":
		switch first(seg.Args) {
		case "panic":
			return "Something I need is broken. nx panic puts every machine's commands back"
		case "quest":
			return "I need to check what we're doing"
		case "log":
			return "I need to see how far I've come"
		}
		return ""
	}
	if what := intent(seg); what != "" {
		return "I need to " + what
	}
	return ""
}

// intent says what one command does, as a verb phrase: "read
// readme.txt". "" for a command no planner uses.
func intent(seg command.Segment) string {
	args := seg.Args
	switch seg.Name {
	case "ssh":
		if len(args) > 0 {
			return "connect to " + args[0]
		}
	case "exit":
		return "disconnect from this machine"
	case "pwd":
		return "print the current working directory"
	case "ls":
		if len(args) > 0 {
			return "list what's in " + args[0]
		}
		return "list files here"
	case "cd":
		switch first(args) {
		case "", "~":
			return "go back to my home folder"
		}
		return "change into " + args[0]
	case "cat":
		if len(args) > 0 {
			return "read " + args[0]
		}
	case "grep":
		if len(args) >= 2 {
			return "search " + args[1] + " for " + args[0]
		}
		if len(args) == 1 {
			return "search for " + args[0]
		}
	case "apt":
		if len(args) >= 2 && args[0] == "install" {
			return "install " + args[1]
		}
		if first(args) == "list" {
			return "see what I can install"
		}
	case "rm":
		if len(args) > 0 {
			return "delete " + args[0]
		}
	case "touch":
		if len(args) > 0 {
			return "create " + args[0]
		}
	case "mv":
		if len(args) >= 2 {
			return "move " + args[0] + " to " + args[1]
		}
	}
	return ""
}

func first(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
