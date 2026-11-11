package session

import (
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/command/apt"
	"github.com/josephburnett/nixy-go/pkg/process"
)

// The keyboard walk also covers every command apt can install: the
// worlds ship only some, and quests install the rest with sudo apt.
func init() {
	var all []string
	for pkg := range apt.PackageRegistry {
		all = append(all, pkg)
	}
	keyboardScenarios = append(keyboardScenarios,
		keyboardScenario{
			name:       "nixy-every-package",
			untilQuest: "modification",
			install:    map[string][]string{"nixy": all},
			commands:   []string{"cd /home/nixy", "mkdir notes", "echo hi"},
		},
		keyboardScenario{
			name:     "laptop-every-package",
			install:  map[string][]string{"laptop": all},
			commands: []string{"mkdir empty"},
		},
	)
}

// consequence reports whether stderr is empty or a real consequence of
// a well-typed command: a permission error, from files or from kill on
// someone else's process.
func consequence(stderr string) bool {
	return stderr == "" || strings.Contains(stderr, "Permission denied") ||
		strings.Contains(stderr, "Operation not permitted")
}

// Ctrl+C reaches a command running on the far side of ssh, through the
// session, and the keyboard offers it only while one runs.
func TestSessionCtrlC(t *testing.T) {
	sess, term := buildScenario(t, keyboardScenario{
		name:     "ctrl-c",
		install:  map[string][]string{"nixy": {"yes"}},
		commands: []string{"ssh nixy"},
	})
	if containsKey(term.State.ValidKeys, process.SigInt) {
		t.Fatal("Ctrl+C offered at a prompt")
	}
	typeCommand(t, sess, term, "yes")
	if !containsKey(term.State.ValidKeys, process.SigInt) {
		t.Fatalf("Ctrl+C not offered while yes runs: %v", term.State.ValidKeys)
	}
	pressAccepted(t, sess, term, process.SigInt)
	if containsKey(term.State.ValidKeys, process.SigInt) || sess.Shell.Hostname() != "nixy" {
		t.Fatalf("after Ctrl+C: host %s, keys %v", sess.Shell.Hostname(), term.State.ValidKeys)
	}
	if status, ok := sess.Shell.Active().LastStatus(); !ok || status != 130 {
		t.Fatalf("$? = %d %v, want 130", status, ok)
	}
}
