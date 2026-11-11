package game

import "github.com/josephburnett/nixy-go/pkg/command"

// CommandTracker records player actions for quest completion checks.
type CommandTracker struct {
	// Commands records every command executed as {hostname, cwd, command}
	commands []CommandRecord
}

type CommandRecord struct {
	Hostname string
	Cwd      []string
	Command  string
	// Status is the command line's exit status, as in $?: 0 is
	// success. Running means it hadn't finished when it was recorded
	// (ssh, until the connection closes), and Status means nothing.
	Status  int
	Running bool
}

func NewCommandTracker() *CommandTracker {
	return &CommandTracker{}
}

func (t *CommandTracker) Record(r CommandRecord) {
	t.commands = append(t.commands, r)
}

// Records returns a copy of all recorded commands, oldest first.
func (t *CommandTracker) Records() []CommandRecord {
	out := make([]CommandRecord, len(t.commands))
	copy(out, t.commands)
	return out
}

// Len is how many commands have been recorded.
func (t *CommandTracker) Len() int { return len(t.commands) }

// LastN returns a copy of the last n recorded commands, oldest first.
func (t *CommandTracker) LastN(n int) []CommandRecord {
	if n > len(t.commands) {
		n = len(t.commands)
	}
	out := make([]CommandRecord, n)
	copy(out, t.commands[len(t.commands)-n:])
	return out
}

// HasCommandOnHost checks if any command was run on the given hostname.
func (t *CommandTracker) HasCommandOnHost(hostname string) bool {
	for _, r := range t.commands {
		if r.Hostname == hostname {
			return true
		}
	}
	return false
}

// HasCommand checks if a specific command was run on a host.
func (t *CommandTracker) HasCommand(hostname, command string) bool {
	for _, r := range t.commands {
		if r.Hostname == hostname && r.Command == command {
			return true
		}
	}
	return false
}

// HasCommandPrefix checks if any command starting with prefix was run on a host.
func (t *CommandTracker) HasCommandPrefix(hostname, prefix string) bool {
	for _, r := range t.commands {
		if r.Hostname == hostname && len(r.Command) >= len(prefix) && r.Command[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// HasVisitedDir checks if the player has been in a specific directory on a host.
func (t *CommandTracker) HasVisitedDir(hostname string, dir []string) bool {
	for _, r := range t.commands {
		if r.Hostname == hostname && pathEqual(r.Cwd, dir) {
			return true
		}
	}
	return false
}

// HasPipe checks if a pipe command was executed on a host.
func (t *CommandTracker) HasPipe(hostname string) bool {
	for _, r := range t.commands {
		if r.Hostname == hostname && command.Parse(r.Command).IsPipeline() {
			return true
		}
	}
	return false
}

func pathEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
