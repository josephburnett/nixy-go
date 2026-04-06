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
