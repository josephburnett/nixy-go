package game

// CommandTracker records player actions for quest completion checks.
type CommandTracker struct {
	// Commands records every command executed as {hostname, cwd, command}
	commands []CommandRecord
}

type CommandRecord struct {
	Hostname string
	Cwd      []string
	Command  string
}

func NewCommandTracker() *CommandTracker {
	return &CommandTracker{}
}

func (t *CommandTracker) Record(hostname string, cwd []string, command string) {
	t.commands = append(t.commands, CommandRecord{
		Hostname: hostname,
		Cwd:      cwd,
		Command:  command,
	})
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
		if r.Hostname == hostname {
			for _, c := range r.Command {
				if c == '|' {
					return true
				}
			}
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
