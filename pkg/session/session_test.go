package session

import (
	"strings"
	"testing"

	_ "github.com/josephburnett/nixy-go/pkg/command/apt"
	_ "github.com/josephburnett/nixy-go/pkg/command/cat"
	_ "github.com/josephburnett/nixy-go/pkg/command/grep"
	_ "github.com/josephburnett/nixy-go/pkg/command/ls"
	_ "github.com/josephburnett/nixy-go/pkg/command/mv"
	_ "github.com/josephburnett/nixy-go/pkg/command/pwd"
	_ "github.com/josephburnett/nixy-go/pkg/command/rm"
	_ "github.com/josephburnett/nixy-go/pkg/command/shell"
	_ "github.com/josephburnett/nixy-go/pkg/command/ssh"
	_ "github.com/josephburnett/nixy-go/pkg/command/sudo"
	_ "github.com/josephburnett/nixy-go/pkg/command/touch"

	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

// typeLine feeds each character of s as a keystroke, then presses Enter.
func typeLine(t *testing.T, s *Session, term *terminal.T, line string) {
	t.Helper()
	for _, r := range line {
		if s.HandleKeystroke(process.Chars(string(r)), term) {
			t.Fatalf("unexpected EOF while typing %q", line)
		}
	}
	if s.HandleKeystroke(process.TermEnter, term) {
		t.Fatalf("unexpected EOF on Enter for %q", line)
	}
}

// TestSshNixyCompletesConnectQuest drives the full keystroke pipeline to
// confirm that typing "ssh nixy" + Enter actually records the command,
// completes the Connect quest, and activates Orientation with its dialog.
// Regression test for a bug where tracker.Record was only called in test
// helpers, never in the real session loop.
func TestSshNixyCompletesConnectQuest(t *testing.T) {
	sess, err := New()
	if err != nil {
		t.Fatal(err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)

	if state := sess.Game.Manager.GetQuestState("connect"); state != game.QuestActive {
		t.Fatalf("connect should start active, got %v", state)
	}

	typeLine(t, sess, term, "ssh nixy")

	if state := sess.Game.Manager.GetQuestState("connect"); state != game.QuestComplete {
		t.Fatalf("connect should be complete after ssh nixy, got %v", state)
	}
	if state := sess.Game.Manager.GetQuestState("orientation"); state != game.QuestActive {
		t.Fatalf("orientation should be active after connect, got %v", state)
	}

	// Orientation activation dialog should be on the terminal.
	out := term.Render()
	if !strings.Contains(out, "look around") {
		t.Fatalf("expected orientation dialog ('look around') in render, got:\n%s", out)
	}
}

// TestOrientationQuestCompletes drives enough commands on nixy to complete
// the Orientation quest and verifies the next quest activates.
func TestOrientationQuestCompletes(t *testing.T) {
	sess, err := New()
	if err != nil {
		t.Fatal(err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)

	typeLine(t, sess, term, "ssh nixy")
	typeLine(t, sess, term, "pwd")
	typeLine(t, sess, term, "ls")
	typeLine(t, sess, term, "cd /home/nixy")
	typeLine(t, sess, term, "pwd") // record a command while at /home/nixy

	if state := sess.Game.Manager.GetQuestState("orientation"); state != game.QuestComplete {
		t.Fatalf("orientation should be complete, got %v", state)
	}
	if state := sess.Game.Manager.GetQuestState("inspection"); state != game.QuestActive {
		t.Fatalf("inspection should be active after orientation, got %v", state)
	}
}
