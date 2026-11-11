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
		{"system log deleted", "inspection", "nixy", []string{"sudo rm /var/log/system.log"}},
		{"cat and grep deleted", "inspection", "nixy", []string{"sudo rm /bin/cat", "sudo rm /bin/grep"}},
		{"grep and sudo deleted", "inspection", "nixy", []string{"sudo rm /bin/grep", "sudo rm /bin/sudo"}},
		{"cat replaced by rm", "inspection", "nixy", []string{
			"sudo apt install mv", "sudo rm /bin/cat", "sudo mv /bin/rm /bin/cat"}},
		{"readme moved", "inspection", "nixy", []string{
			"sudo apt install mv", "sudo mv /home/nixy/readme.txt /home/nixy/r.txt"}},
		{"rm and apt deleted", "modification", "nixy", []string{"sudo rm /bin/apt", "sudo rm /bin/rm"}},
		{"touch deleted", "modification", "nixy", []string{"sudo rm /bin/touch"}},
		{"grep renamed", "composition", "nixy", []string{
			"sudo apt install mv", "sudo mv /bin/grep /bin/g"}},
		{"target deleted", "composition", "nixy", []string{"sudo rm /home/nixy/projects/target.txt"}},
		{"server sudo deleted", "permissions", "server", []string{
			"sudo apt install rm", "sudo rm /bin/sudo"}},
		{"server apt deleted", "permissions", "server", []string{
			"sudo apt install rm", "sudo rm /bin/apt"}},
		{"touch installed then deleted", "permissions", "server", []string{
			"sudo apt install touch", "sudo apt install rm", "sudo rm /bin/touch"}},
		{"hosts deleted", "permissions", "server", []string{
			"sudo apt install rm", "sudo rm /etc/hosts"}},
		{"nixy sudo deleted", "lockdown", "nixy", []string{"sudo rm /bin/sudo"}},
		{"nixy rm deleted", "lockdown", "nixy", []string{"sudo rm /bin/rm"}},
		{"whoami deleted", "identity", "nixy", []string{"sudo rm /bin/whoami"}},
		{"file in the way of notes", "notes", "nixy", []string{"touch /home/user/notes"}},
		{"root's notes folder", "notes", "nixy", []string{
			"sudo mkdir /home/user/notes", "sudo touch /home/user/notes/todo.txt"}},
		{"root's to-do list", "notes", "nixy", []string{
			"mkdir /home/user/notes", "sudo touch /home/user/notes/todo.txt"}},
		{"home deleted", "notes", "nixy", []string{"sudo apt install rmdir", "sudo rmdir /home/user"}},
		{"echo and mkdir deleted", "notes", "nixy", []string{"sudo rm /bin/echo", "sudo rm /bin/mkdir"}},
		{"man deleted", "manual", "nixy", []string{"sudo rm /bin/man"}},
		{"wc deleted", "counting", "nixy", []string{"sudo rm /bin/wc"}},
		{"log deleted before counting", "counting", "nixy", []string{"sudo rm /var/log/system.log"}},
		{"sudo deleted before yes", "interrupt", "nixy", []string{"sudo rm /bin/sudo"}},
		{"ps and kill deleted", "processes", "nixy", []string{"sudo rm /bin/ps", "sudo rm /bin/kill"}},
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

// The tracker records the program that ran: with /bin/cat replaced by
// rm, `cat readme.txt` deletes the readme, and must not count as
// reading it.
func TestTrackerRecordsWhatRan(t *testing.T) {
	sess, term := newLoggedIn(t, "user")
	followHints(t, sess, term, hintFinishKeys, func() bool { return activeQuestID(sess) == "inspection" })
	for term.State.Line != "" {
		pressAccepted(t, sess, term, process.TermBackspace)
	}
	for _, l := range []string{"sudo apt install mv", "sudo rm /bin/cat", "sudo mv /bin/rm /bin/cat", "cat /home/nixy/readme.txt"} {
		typeCommand(t, sess, term, l)
	}
	last := sess.Game.Manager.Tracker.LastN(1)[0]
	if invs := last.Invocations(); len(invs) != 1 || invs[0].Name != "rm" {
		t.Fatalf("`cat readme.txt` with cat replaced by rm recorded as %+v", invs)
	}
	if activeQuestID(sess) != "inspection" {
		t.Fatalf("running rm completed %q", "inspection")
	}
}
