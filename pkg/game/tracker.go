package game

import (
	"slices"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
)

// CommandTracker records the command lines the player ran, for quests
// to check what the player has done.
//
// Quests should ask what a command did, not how it was spelled: "cat
// read /home/nixy/readme.txt and succeeded" is true of `cat readme.txt`
// in /home/nixy, `cat ~/../nixy/readme.txt` and `cat
// /home/nixy/readme.txt | grep x` alike. Invocations and Path give that
// view; when a quest's goal is an effect on the world (a file exists),
// it should check the world instead.
type CommandTracker struct {
	commands []CommandRecord
}

type CommandRecord struct {
	Hostname string
	// User ran the command; ~ in it is their home.
	User string
	// Dir is the folder the command ran in, which its relative paths
	// are relative to. Cwd is the folder the shell was in afterwards
	// (so a cd counts as a visit to where it went).
	Dir     []string
	Cwd     []string
	Command string
	// Status is the command line's exit status, as in $?: 0 is
	// success. Running means it hadn't finished when it was recorded
	// (ssh, until the connection closes), and Status means nothing.
	Status  int
	Running bool
}

// OK reports whether the command line finished and succeeded.
func (r CommandRecord) OK() bool { return !r.Running && r.Status == 0 }

// Failed reports whether the command line finished and failed.
func (r CommandRecord) Failed() bool { return !r.Running && r.Status != 0 }

// Invocation is one program a command line ran: each segment of a
// pipeline, with sudo seen through (`sudo rm x` invokes rm, as root).
type Invocation struct {
	Name string
	Args []string // as the program saw them: ~ already expanded
	Sudo bool
	dir  []string
}

// Path resolves one of the invocation's args as a path, the way the
// program does: relative to the folder it ran in.
func (i Invocation) Path(arg string) []string { return file.Resolve(i.dir, arg) }

// Paths resolves every arg from index from onwards as a path.
func (i Invocation) Paths(from int) [][]string {
	var out [][]string
	for j := from; j < len(i.Args); j++ {
		out = append(out, i.Path(i.Args[j]))
	}
	return out
}

// HasPath reports whether an arg from index from onwards resolves to
// path.
func (i Invocation) HasPath(from int, path []string) bool {
	for _, p := range i.Paths(from) {
		if slices.Equal(p, path) {
			return true
		}
	}
	return false
}

// Invocations parses the command line into the programs it ran, in
// pipeline order.
func (r CommandRecord) Invocations() []Invocation {
	home := file.Home(r.User)
	var out []Invocation
	for _, seg := range command.Parse(r.Command).Segments {
		if seg.Name == "" {
			continue
		}
		inv := Invocation{Name: seg.Name, dir: r.Dir}
		for _, a := range seg.Args {
			inv.Args = append(inv.Args, file.ExpandHome(a, home))
		}
		if inv.Name == "sudo" && len(inv.Args) > 0 {
			inv = Invocation{Name: inv.Args[0], Args: inv.Args[1:], Sudo: true, dir: r.Dir}
		}
		out = append(out, inv)
	}
	return out
}

// Invokes reports whether the command line ran the program name.
func (r CommandRecord) Invokes(name string) bool {
	for _, inv := range r.Invocations() {
		if inv.Name == name {
			return true
		}
	}
	return false
}

// IsPipeline reports whether the command line is a pipeline.
func (r CommandRecord) IsPipeline() bool {
	return command.Parse(r.Command).IsPipeline()
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

// Any reports whether a command recorded on hostname matches.
func (t *CommandTracker) Any(hostname string, match func(CommandRecord) bool) bool {
	for _, r := range t.commands {
		if r.Hostname == hostname && match(r) {
			return true
		}
	}
	return false
}

// AnyInvocation reports whether a command recorded on hostname, for
// which ok holds, ran a program that matches.
func (t *CommandTracker) AnyInvocation(hostname string, ok func(CommandRecord) bool, match func(Invocation) bool) bool {
	return t.Any(hostname, func(r CommandRecord) bool {
		if !ok(r) {
			return false
		}
		return slices.ContainsFunc(r.Invocations(), match)
	})
}

// HasCommandOnHost checks if any command was run on the given hostname.
func (t *CommandTracker) HasCommandOnHost(hostname string) bool {
	return t.Any(hostname, func(CommandRecord) bool { return true })
}

// HasVisitedDir checks if the player has been in a specific directory on a host.
func (t *CommandTracker) HasVisitedDir(hostname string, dir []string) bool {
	return t.Any(hostname, func(r CommandRecord) bool { return slices.Equal(r.Cwd, dir) })
}
