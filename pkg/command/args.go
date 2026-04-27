package command

import (
	"sort"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// ValidArgsFile returns valid next inputs for a file path argument (files and folders).
func ValidArgsFile(sim *simulation.S, hostname string, cwd []string, partial string) []process.Datum {
	return validArgsPath(sim, hostname, cwd, partial, false)
}

// ValidArgsFolder returns valid next inputs for a folder path argument (folders only).
func ValidArgsFolder(sim *simulation.S, hostname string, cwd []string, partial string) []process.Datum {
	return validArgsPath(sim, hostname, cwd, partial, true)
}

// ValidArgsHostname returns valid next inputs for hostnames from /etc/hosts.
func ValidArgsHostname(sim *simulation.S, hostname string, _ []string, partial string) []process.Datum {
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil
	}
	hostsFile, err := c.Filesystem.Navigate([]string{"etc", "hosts"})
	if err != nil {
		return nil
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

// validArgsFromList returns valid next chars to match items in the list.
func validArgsFromList(items []string, partial string) []process.Datum {
	var valid []process.Datum

	if partial == "" {
		firstChars := map[byte]bool{}
		for _, item := range items {
			if len(item) > 0 {
				firstChars[item[0]] = true
			}
		}
		for ch := range firstChars {
			valid = append(valid, process.Chars(string(ch)))
		}
		return valid
	}

	continuations := map[byte]bool{}
	exactMatch := false
	for _, item := range items {
		if strings.HasPrefix(item, partial) {
			if item == partial {
				exactMatch = true
			}
			if len(item) > len(partial) {
				continuations[item[len(partial)]] = true
			}
		}
	}
	for ch := range continuations {
		valid = append(valid, process.Chars(string(ch)))
	}
	if exactMatch {
		valid = append(valid, process.TermEnter)
	}

	return valid
}

func validArgsPath(sim *simulation.S, hostname string, cwd []string, partial string, foldersOnly bool) []process.Datum {
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil
	}

	dir, prefix := splitPathForCompletion(cwd, partial)

	parent, err := c.Filesystem.Navigate(dir)
	if err != nil {
		return nil
	}
	if parent.Type != file.Folder {
		return nil
	}

	var names []string
	for name, f := range parent.Files {
		if foldersOnly && f.Type != file.Folder {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	var valid []process.Datum

	if prefix == "" {
		firstChars := map[byte]bool{}
		for _, name := range names {
			if len(name) > 0 {
				firstChars[name[0]] = true
			}
		}
		if partial == "" {
			firstChars['/'] = true
			firstChars['.'] = true
		}
		for ch := range firstChars {
			valid = append(valid, process.Chars(string(ch)))
		}
		// Non-empty partial with prefix=="" means the partial ended with `/`
		// (e.g. "/", "../", "foo/") — fully resolved to a directory.
		if partial != "" {
			valid = append(valid, process.TermEnter)
		}
		return valid
	}

	// Special path components: "." (cwd) and ".." (parent).
	if prefix == "." || prefix == ".." {
		valid = append(valid, process.TermEnter)
		valid = append(valid, process.Chars("/"))
		if prefix == "." {
			// Allow another `.` to form `..`.
			valid = append(valid, process.Chars("."))
		}
		return valid
	}

	continuations := map[byte]bool{}
	exactMatch := false
	for _, name := range names {
		if strings.HasPrefix(name, prefix) {
			if name == prefix {
				exactMatch = true
			}
			if len(name) > len(prefix) {
				continuations[name[len(prefix)]] = true
			}
		}
	}
	for ch := range continuations {
		valid = append(valid, process.Chars(string(ch)))
	}
	if exactMatch {
		valid = append(valid, process.TermEnter)
		valid = append(valid, process.Chars("/"))
	}

	return valid
}

// ValidArgsCd is the validator for the cd builtin. It wraps
// ValidArgsFolder and filters out Enter at any partial that would be a
// no-op (resolves to the current cwd) — e.g. `cd ..` at root, `cd /`
// at root, `cd .`, `cd /home/user` when already in /home/user. The
// "no mistakes" invariant says the keyboard should only offer keys
// that lead to a real, observable action; a no-op cd is not one.
func ValidArgsCd(sim *simulation.S, hostname string, cwd []string, partial string) []process.Datum {
	keys := ValidArgsFolder(sim, hostname, cwd, partial)
	if !cdWouldMove(cwd, partial) {
		keys = filterDatum(keys, process.TermEnter)
	}
	return keys
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
	final := resolveCdPath(cwd, partial)
	return !pathsEqual(final, cwd)
}

// resolveCdPath applies cd's path-resolution rules (./ skipped, ../ pops,
// leading / makes absolute) to compute the final cwd `partial` would
// produce. Mirrors the logic in shell.builtinCd.
func resolveCdPath(cwd []string, partial string) []string {
	var result []string
	rest := partial
	if strings.HasPrefix(partial, "/") {
		result = []string{}
		rest = strings.TrimPrefix(partial, "/")
	} else {
		result = append([]string{}, cwd...)
	}
	for _, p := range strings.Split(rest, "/") {
		switch p {
		case "..":
			if len(result) > 0 {
				result = result[:len(result)-1]
			}
		case ".", "":
			// no-op
		default:
			result = append(result, p)
		}
	}
	return result
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

func filterDatum(in []process.Datum, drop process.Datum) []process.Datum {
	out := make([]process.Datum, 0, len(in))
	for _, d := range in {
		if equalDatum(d, drop) {
			continue
		}
		out = append(out, d)
	}
	return out
}

func equalDatum(a, b process.Datum) bool {
	switch av := a.(type) {
	case process.Chars:
		bv, ok := b.(process.Chars)
		return ok && av == bv
	case process.TermCode:
		bv, ok := b.(process.TermCode)
		return ok && av == bv
	}
	return false
}

func splitPathForCompletion(cwd []string, partial string) (dir []string, prefix string) {
	if partial == "" {
		return cwd, ""
	}

	if strings.HasPrefix(partial, "/") {
		parts := strings.Split(strings.TrimPrefix(partial, "/"), "/")
		if len(parts) == 1 {
			return []string{}, parts[0]
		}
		var dirParts []string
		for _, p := range parts[:len(parts)-1] {
			if p == ".." {
				if len(dirParts) > 0 {
					dirParts = dirParts[:len(dirParts)-1]
				}
			} else if p != "" && p != "." {
				dirParts = append(dirParts, p)
			}
		}
		return dirParts, parts[len(parts)-1]
	}

	parts := strings.Split(partial, "/")
	if len(parts) == 1 {
		return cwd, parts[0]
	}
	dir = append([]string{}, cwd...)
	for _, p := range parts[:len(parts)-1] {
		if p == ".." {
			if len(dir) > 0 {
				dir = dir[:len(dir)-1]
			}
		} else if p != "" && p != "." {
			dir = append(dir, p)
		}
	}
	return dir, parts[len(parts)-1]
}
