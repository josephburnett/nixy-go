package file

import "strings"

// Resolve takes a path expression (absolute, relative, with `.`, `..`, or
// empty components) and a current working directory, producing the
// resolved canonical path components.
//
//	Resolve([]string{"home", "joe"}, "..")       → []string{"home"}
//	Resolve([]string{"home", "joe"}, "/etc/hosts") → []string{"etc", "hosts"}
//	Resolve([]string{"home", "joe"}, "./foo")    → []string{"home", "joe", "foo"}
//	Resolve([]string{}, "..")                    → []string{}    // clamps at root
//
// Absolute paths (starting with `/`) discard cwd. `..` pops the last
// component (clamping at root). `.` and empty components are skipped
// (empties arise from leading `/`, trailing `/`, or `//`).
func Resolve(cwd []string, expr string) []string {
	var parts []string
	rest := expr
	if strings.HasPrefix(expr, "/") {
		parts = []string{}
		rest = strings.TrimPrefix(expr, "/")
	} else {
		parts = append([]string{}, cwd...)
	}
	for p := range strings.SplitSeq(rest, "/") {
		switch p {
		case "..":
			if len(parts) > 0 {
				parts = parts[:len(parts)-1]
			}
		case ".", "":
			// no-op
		default:
			parts = append(parts, p)
		}
	}
	return parts
}

// Split resolves the directory portion of an expression and returns the
// resolved directory plus the trailing (unresolved) last component.
//
//	Split([]string{"home"}, "joe/foo")     → ["home", "joe"], "foo"
//	Split([]string{"home"}, "/etc/hosts")  → ["etc"], "hosts"
//	Split([]string{}, "foo")               → [], "foo"
//	Split([]string{"home"}, "foo/")        → ["home", "foo"], ""    // trailing `/` → empty last
//	Split(cwd, "")                         → cwd, ""
//
// The trailing component is returned verbatim — no `.` / `..` / empty
// filtering — because completion validators need to match it as a literal
// prefix against directory entries.
func Split(cwd []string, expr string) (dir []string, last string) {
	if expr == "" {
		return append([]string{}, cwd...), ""
	}
	idx := strings.LastIndex(expr, "/")
	if idx < 0 {
		// No `/` — entire expr is the trailing component, dir is cwd.
		return append([]string{}, cwd...), expr
	}
	return Resolve(cwd, expr[:idx+1]), expr[idx+1:]
}

// ExpandHome does the shell's tilde expansion on one word: `~` is the
// home folder and `~/rest` is rest under it. Other words, including
// `~user` forms, are returned unchanged.
//
//	ExpandHome("~", ["home", "joe"])        → "/home/joe"
//	ExpandHome("~/notes", ["home", "joe"])  → "/home/joe/notes"
//	ExpandHome("a~b", ["home", "joe"])      → "a~b"
func ExpandHome(word string, home []string) string {
	if word != "~" && !strings.HasPrefix(word, "~/") {
		return word
	}
	return "/" + strings.Join(home, "/") + word[1:]
}
