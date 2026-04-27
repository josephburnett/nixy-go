package command

import (
	"sort"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// ValidArgsFile returns valid next inputs for a file path argument (files and folders).
func ValidArgsFile(sim *simulation.S, hostname string, cwd []string, partial string) simulation.Suggestion {
	return validArgsPath(sim, hostname, cwd, partial, false)
}

// ValidArgsFolder returns valid next inputs for a folder path argument (folders only).
func ValidArgsFolder(sim *simulation.S, hostname string, cwd []string, partial string) simulation.Suggestion {
	return validArgsPath(sim, hostname, cwd, partial, true)
}

// ValidArgsHostname returns valid next inputs for hostnames from /etc/hosts.
func ValidArgsHostname(sim *simulation.S, hostname string, _ []string, partial string) simulation.Suggestion {
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return simulation.Suggestion{}
	}
	hostsFile, err := c.Filesystem.Navigate([]string{"etc", "hosts"})
	if err != nil {
		return simulation.Suggestion{}
	}

	var hosts []string
	for _, line := range strings.Split(hostsFile.Data, "\n") {
		h := strings.TrimSpace(line)
		if h != "" {
			hosts = append(hosts, h)
		}
	}

	return validArgsFromList(hosts, partial)
}

// validArgsFromList returns the suggestion for matching items in a list.
func validArgsFromList(items []string, partial string) simulation.Suggestion {
	if partial == "" {
		firstChars := map[rune]bool{}
		for _, item := range items {
			if len(item) > 0 {
				firstChars[rune(item[0])] = true
			}
		}
		return simulation.Suggestion{Chars: sortedKeys(firstChars)}
	}

	continuations := map[rune]bool{}
	exactMatch := false
	for _, item := range items {
		if strings.HasPrefix(item, partial) {
			if item == partial {
				exactMatch = true
			}
			if len(item) > len(partial) {
				continuations[rune(item[len(partial)])] = true
			}
		}
	}
	return simulation.Suggestion{Chars: sortedKeys(continuations), Complete: exactMatch}
}

func validArgsPath(sim *simulation.S, hostname string, cwd []string, partial string, foldersOnly bool) simulation.Suggestion {
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return simulation.Suggestion{}
	}

	dir, prefix := file.Split(cwd, partial)

	parent, err := c.Filesystem.Navigate(dir)
	if err != nil {
		return simulation.Suggestion{}
	}
	if parent.Type != file.Folder {
		return simulation.Suggestion{}
	}

	var names []string
	for name, f := range parent.Files {
		if foldersOnly && f.Type != file.Folder {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	if prefix == "" {
		chars := map[rune]bool{}
		for _, name := range names {
			if len(name) > 0 {
				chars[rune(name[0])] = true
			}
		}
		if partial == "" {
			chars['/'] = true
			chars['.'] = true
		}
		// Non-empty partial with prefix=="" means the partial ended with `/`
		// (e.g. "/", "../", "foo/") — fully resolved to a directory, so the
		// arg is complete (Enter is valid for commands that accept dirs).
		return simulation.Suggestion{Chars: sortedKeys(chars), Complete: partial != ""}
	}

	// Special path components: "." (cwd) and ".." (parent). The arg is
	// complete (the path resolves), and `/` lets the player descend; `.`
	// after `.` forms `..`.
	if prefix == "." || prefix == ".." {
		chars := map[rune]bool{'/': true}
		if prefix == "." {
			chars['.'] = true
		}
		return simulation.Suggestion{Chars: sortedKeys(chars), Complete: true}
	}

	continuations := map[rune]bool{}
	exactMatch := false
	for _, name := range names {
		if strings.HasPrefix(name, prefix) {
			if name == prefix {
				exactMatch = true
			}
			if len(name) > len(prefix) {
				continuations[rune(name[len(prefix)])] = true
			}
		}
	}
	if exactMatch {
		// Exact match — `/` is valid to descend into the directory (or
		// continue addressing under it).
		continuations['/'] = true
	}
	return simulation.Suggestion{Chars: sortedKeys(continuations), Complete: exactMatch}
}

// ValidArgsCd is the validator for the cd builtin. It wraps
// ValidArgsFolder and clears Complete at any partial that would be a
// no-op (resolves to the current cwd) — e.g. `cd ..` at root, `cd /`
// at root, `cd .`, `cd /home/user` when already in /home/user. The
// "no mistakes" invariant says the keyboard should only offer keys
// that lead to a real, observable action; a no-op cd is not one.
func ValidArgsCd(sim *simulation.S, hostname string, cwd []string, partial string) simulation.Suggestion {
	sug := ValidArgsFolder(sim, hostname, cwd, partial)
	if !cdWouldMove(cwd, partial) {
		sug.Complete = false
	}
	return sug
}

// cdWouldMove reports whether `cd partial` from `cwd` would land on a
// different path. Returns false for partials that resolve back to cwd
// itself.
func cdWouldMove(cwd []string, partial string) bool {
	if partial == "" {
		// Bare `cd` (handled by OptionalArgs) goes to /. That's a move
		// iff cwd is non-root.
		return len(cwd) > 0
	}
	return !pathsEqual(file.Resolve(cwd, partial), cwd)
}

func pathsEqual(a, b []string) bool {
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

func sortedKeys(m map[rune]bool) []rune {
	out := make([]rune, 0, len(m))
	for r := range m {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

