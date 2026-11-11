package session

import (
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/command/apt"
	"github.com/josephburnett/nixy-go/pkg/process"
)

// The keyboard walk also covers every command apt can install that no
// world ships (quests install them with sudo apt), over nixy's world.
func init() {
	shipped := map[string]bool{}
	for _, m := range worldMachines() {
		if bin, err := m.Filesystem().Navigate([]string{"bin"}); err == nil {
			for name := range bin.Files {
				shipped[name] = true
			}
		}
	}
	install := []string{"sudo"} // so the walk tries each under sudo too
	for pkg := range apt.PackageRegistry {
		if !shipped[pkg] {
			install = append(install, pkg)
		}
	}
	keyboardScenarios = append(keyboardScenarios, keyboardScenario{
		name:       "nixy-every-package",
		untilQuest: "modification",
		install:    map[string][]string{"nixy": install},
		commands:   []string{"cd /home/nixy", "mkdir empty", "echo hi"},
	})
}

// consequence reports whether stderr is empty or a real consequence of
// a well-typed command: a permission error, from files or from kill on
// someone else's process, or the shell reporting a job the player
// killed.
func consequence(stderr string) bool {
	return stderr == "" || strings.Contains(stderr, "Permission denied") ||
		strings.Contains(stderr, "Operation not permitted") ||
		// The player killed their own job (or the shell an ssh runs).
		stderr == "Killed\n" || stderr == "Terminated\n"
}

// A pipeline that works quietly before printing finishes within the
// keystroke that started it, so the prompt comes straight back.
func TestSessionQuietPipelineFinishes(t *testing.T) {
	sess, term := buildScenario(t, keyboardScenario{
		name:    "quiet-pipeline",
		install: map[string][]string{"laptop": {"yes", "head", "wc"}},
	})
	typeCommand(t, sess, term, "yes | head | wc -l")
	if status, ok := sess.Shell.Active().LastStatus(); !ok || status != 0 {
		t.Fatalf("still running after Enter (status %d %v), keys %v", status, ok, term.State.ValidKeys)
	}
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
