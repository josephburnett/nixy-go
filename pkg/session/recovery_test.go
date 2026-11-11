package session

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/process"
)

// TestBrokenPrerequisitesRecover breaks, for each quest, what that
// quest depends on (its files, the commands it plans, sudo itself),
// then demands that following the hints alone still finishes the whole
// quest line. Random play rarely reaches these; this is the directed
// guard for stale plans.
func TestBrokenPrerequisitesRecover(t *testing.T) {
	cases := []struct {
		name  string
		quest string // break things once this quest is active
		host  string // on this machine
		lines []string
	}{
		{"readme deleted", "inspection", "nixy", []string{"rm /home/nixy/readme.txt"}},
		{"server sudo deleted", "permissions", "server", []string{
			"sudo apt install rm", "sudo rm /bin/sudo"}},
		{"server apt deleted", "permissions", "server", []string{
			"sudo apt install rm", "sudo rm /bin/apt"}},
		{"touch installed then deleted", "permissions", "server", []string{
			"sudo apt install touch", "sudo apt install rm", "sudo rm /bin/touch"}},
		{"hosts deleted", "permissions", "server", []string{
			"sudo apt install rm", "sudo rm /etc/hosts"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sess, term := newLoggedIn(t, "user")
			followHints(t, sess, term, hintFinishKeys, func() bool { return activeQuestID(sess) == tc.quest })
			// Get to the machine at an empty prompt.
			for term.State.Line != "" {
				pressAccepted(t, sess, term, process.TermBackspace)
			}
			if sess.Shell.Hostname() != tc.host {
				if sess.Shell.Hostname() != "laptop" {
					typeCommand(t, sess, term, "exit")
				}
				typeCommand(t, sess, term, "ssh "+tc.host)
			}
			for _, l := range tc.lines {
				typeCommand(t, sess, term, l)
			}
			followHints(t, sess, term, hintFinishKeys, func() bool { return sess.Game.Manager.ActiveQuest() == nil })
			for id, st := range snapQuests(sess) {
				if st != game.QuestComplete {
					t.Fatalf("quest %s is %v", id, stateName(st))
				}
			}
		})
	}
}
