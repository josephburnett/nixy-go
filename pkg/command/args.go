package command

import (
	"errors"
	"sort"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Arg validates one argument token (never contains a space). prev holds
// the arguments already typed before it. Complete means the token is a
// valid value for this position. Every char offered must lead toward a
// Complete token.
type Arg func(ctx simulation.ArgContext, prev []string, token string) simulation.Suggestion

// Positional is the validator for a command that takes a fixed list of
// positional arguments, e.g. `mv SRC DST` or `grep PATTERN [FILE]`.
type Positional struct {
	Args []Arg
	// Required is how many args a standalone run needs.
	Required int
	// ReadsStdin commands can take their input from an upstream pipe
	// instead of their trailing args: in a pipe they need only
	// PipeRequired args (`ls | cat`, `ls | grep x`).
	ReadsStdin   bool
	PipeRequired int
}

// NoArgs is the validator for commands that take no arguments.
var NoArgs = Positional{}.Validate

// Validate implements simulation.ValidArgs. Arguments are separated by
// single spaces. A space is offered after a complete argument when
// another argument can follow, or when the command is complete (the
// trailing space before `|`).
func (p Positional) Validate(ctx simulation.ArgContext, partial string) simulation.Suggestion {
	tokens := strings.Split(partial, " ")
	i := len(tokens) - 1
	prev, cur := tokens[:i], tokens[i]
	for j, tok := range prev {
		if j >= len(p.Args) || tok == "" || !p.Args[j](ctx, prev[:j], tok).Complete {
			return simulation.Suggestion{}
		}
	}
	if i >= len(p.Args) {
		// Only the trailing space after the last argument.
		if cur != "" {
			return simulation.Suggestion{}
		}
		return simulation.Suggestion{Complete: p.enough(ctx, i)}
	}

	sug := p.Args[i](ctx, prev, cur)
	var chars []rune
	for _, r := range sug.Chars {
		if r != ' ' && r != '|' {
			chars = append(chars, r)
		}
	}
	if cur == "" {
		return simulation.Suggestion{Chars: chars, Complete: p.enough(ctx, i)}
	}
	if !sug.Complete {
		return simulation.Suggestion{Chars: chars}
	}
	complete := p.enough(ctx, i+1)
	if complete || (i+1 < len(p.Args) && live(p.Args[i+1](ctx, tokens, ""))) {
		chars = append(chars, ' ')
	}
	return simulation.Suggestion{Chars: chars, Complete: complete}
}

// enough reports whether n complete args make a runnable command.
func (p Positional) enough(ctx simulation.ArgContext, n int) bool {
	if n >= p.Required {
		return true
	}
	return ctx.InPipe && p.ReadsStdin && n >= p.PipeRequired
}

// live reports whether a suggestion leads anywhere: it is complete or
// has a key to type next.
func live(s simulation.Suggestion) bool {
	return s.Complete || len(s.Chars) > 0
}

// ValidCommand validates "NAME ARGS..." for a choice of names, each with
// its own args validator. It serves the shell's command position, sudo's
// command argument and subcommands (apt install, nx quest). A name is
// offered only when its args validator leads somewhere, so the player
// can't type a command (or subcommand) that has no valid completion.
func ValidCommand(
	ctx simulation.ArgContext,
	text string,
	names []string,
	lookup func(name string) (simulation.ValidArgs, bool),
) simulation.Suggestion {
	name, rest, hasArgs := strings.Cut(text, " ")
	if hasArgs {
		va, ok := lookup(name)
		if !ok || !contains(names, name) {
			return simulation.Suggestion{}
		}
		return va(ctx, rest)
	}
	var chars charSet
	complete := false
	for _, n := range names {
		if !strings.HasPrefix(n, name) {
			continue
		}
		va, ok := lookup(n)
		if !ok {
			continue
		}
		s := va(ctx, "")
		if !live(s) {
			continue
		}
		if len(n) > len(name) {
			chars.add(rune(n[len(name)]))
		} else {
			chars.add(' ')
			complete = s.Complete
		}
	}
	return simulation.Suggestion{Chars: sortedKeys(chars), Complete: complete}
}

// Subcommands is the validator for `CMD SUB ARGS...`, e.g. apt install.
func Subcommands(subs map[string]simulation.ValidArgs) simulation.ValidArgs {
	names := make([]string, 0, len(subs))
	for n := range subs {
		names = append(names, n)
	}
	sort.Strings(names)
	lookup := func(n string) (simulation.ValidArgs, bool) {
		va, ok := subs[n]
		return va, ok
	}
	return func(ctx simulation.ArgContext, partial string) simulation.Suggestion {
		return ValidCommand(ctx, partial, names, lookup)
	}
}

// OneOf returns an Arg that accepts exactly one of the given words.
func OneOf(words func(ctx simulation.ArgContext) []string) Arg {
	return func(ctx simulation.ArgContext, _ []string, token string) simulation.Suggestion {
		var chars charSet
		complete := false
		for _, w := range words(ctx) {
			if !strings.HasPrefix(w, token) {
				continue
			}
			if len(w) > len(token) {
				chars.add(rune(w[len(token)]))
			} else {
				complete = true
			}
		}
		return simulation.Suggestion{Chars: sortedKeys(chars), Complete: complete}
	}
}

// Hostname accepts a host listed in /etc/hosts.
var Hostname = OneOf(func(ctx simulation.ArgContext) []string {
	c, err := ctx.Sim.GetComputer(ctx.Hostname)
	if err != nil {
		return nil
	}
	data, err := c.Filesystem.ReadFile([]string{"etc", "hosts"}, ctx.User)
	if err != nil {
		return nil
	}
	var hosts []string
	for _, line := range strings.Split(data, "\n") {
		if h := strings.TrimSpace(line); h != "" {
			hosts = append(hosts, h)
		}
	}
	return hosts
})

// Pattern accepts a literal search word for grep. It is limited to
// characters that mean nothing special to a real shell or to grep's
// regular expressions, so what the player types behaves the same in
// real Unix. A leading `-` would be an option, so it is not offered.
func Pattern(_ simulation.ArgContext, _ []string, token string) simulation.Suggestion {
	var chars []rune
	for _, r := range patternChars {
		if r == '-' && token == "" {
			continue
		}
		chars = append(chars, r)
	}
	return simulation.Suggestion{Chars: chars, Complete: token != ""}
}

// FilePath accepts an existing file (not a folder). Folders are offered
// only as path components to descend into.
func FilePath(ctx simulation.ArgContext, _ []string, token string) simulation.Suggestion {
	return pathArg(ctx, token, false, func(_ []string, f *file.F, dir bool) bool {
		return f != nil && f.Type != file.Folder && !dir
	})
}

// AnyPath accepts any existing path, file or folder.
func AnyPath(ctx simulation.ArgContext, _ []string, token string) simulation.Suggestion {
	return pathArg(ctx, token, false, func(_ []string, f *file.F, _ bool) bool {
		return f != nil
	})
}

// CdTarget accepts a folder other than the cwd: cd to where you
// already are is a no-op, not an action.
func CdTarget(ctx simulation.ArgContext, _ []string, token string) simulation.Suggestion {
	return pathArg(ctx, token, false, func(target []string, f *file.F, _ bool) bool {
		return f != nil && f.Type == file.Folder && !pathsEqual(target, ctx.Cwd)
	})
}

// NewPath accepts a path whose parent folder exists and whose last
// component is a plain name: a new file, or an existing one (touch
// on an existing file just updates it).
func NewPath(ctx simulation.ArgContext, _ []string, token string) simulation.Suggestion {
	return pathArg(ctx, token, true, func(target []string, _ *file.F, dir bool) bool {
		return !dir && len(target) > 0
	})
}

// MoveDest accepts a destination for moving the file prev[0], the way
// real mv treats its last argument:
//   - an existing folder: move into it, unless that is where the file
//     already is or a folder of the same name is in the way;
//   - an existing file other than the source: overwrite it;
//   - a new name in an existing folder: rename.
func MoveDest(ctx simulation.ArgContext, prev []string, token string) simulation.Suggestion {
	if len(prev) == 0 {
		return simulation.Suggestion{}
	}
	src := file.Resolve(ctx.Cwd, file.ExpandHome(prev[0], ctx.Home))
	if len(src) == 0 {
		return simulation.Suggestion{}
	}
	base := src[len(src)-1]
	return pathArg(ctx, token, true, func(target []string, f *file.F, dir bool) bool {
		if f != nil && f.Type == file.Folder {
			into := append(append([]string{}, target...), base)
			if pathsEqual(into, src) {
				return false
			}
			existing, ok := f.Files[base]
			return !ok || existing.Type != file.Folder
		}
		return !dir && len(target) > 0 && !pathsEqual(target, src)
	})
}

// wordChars are the characters offered for free-form words (new file
// names, grep patterns): portable filename characters, none of which
// a real shell would interpret.
const wordChars = "-.0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"

// patternChars are wordChars without `.`, which real grep reads as "any
// character" while ours matches it literally.
const patternChars = "-0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"

func validNewName(name string) bool {
	if name == "" || name == "." || name == ".." || name[0] == '-' {
		return false
	}
	return isWord(name)
}

func isWord(s string) bool {
	for _, r := range s {
		if !strings.ContainsRune(wordChars, r) {
			return false
		}
	}
	return true
}

// pathAccept decides whether a resolved path is a valid value. f is nil
// when nothing exists there. dir is true when the token is written as
// a directory (ends in `/`, or is `.` or `..`).
type pathAccept func(target []string, f *file.F, dir bool) bool

// pathArg is path completion shared by the path Args. It offers only
// keys that lead to an accepted path: existing names whose subtree
// holds an accepted path, `/` after a folder worth descending into,
// `.`/`..`, and (when allowNew) characters of a new name.
func pathArg(ctx simulation.ArgContext, token string, allowNew bool, accept pathAccept) simulation.Suggestion {
	c, err := ctx.Sim.GetComputer(ctx.Hostname)
	if err != nil {
		return simulation.Suggestion{}
	}
	root := c.Filesystem
	viable := func(path []string, f *file.F) bool { return viableDir(ctx.User, path, f, allowNew, accept) }

	// `~` is the home folder, written as a directory; `~/...` is a path
	// below it (see file.ExpandHome).
	home, homeOK := ctx.Home, ctx.Home != nil
	homeFile, err := root.Lookup(home, ctx.User)
	homeOK = homeOK && err == nil && homeFile.Type == file.Folder
	if token == "~" {
		if !homeOK {
			return simulation.Suggestion{}
		}
		var chars charSet
		if viable(home, homeFile) {
			chars.add('/')
		}
		return simulation.Suggestion{Chars: sortedKeys(chars), Complete: accept(home, homeFile, true)}
	}
	token = file.ExpandHome(token, home)

	var chars charSet
	dirPath, prefix := file.Split(ctx.Cwd, token)
	if token == "" && viable(nil, root) {
		chars.add('/')
	}
	if token == "" && homeOK && (accept(home, homeFile, true) || viable(home, homeFile)) {
		chars.add('~')
	}
	// Nothing inside the folder is offered when it is gone (deleted
	// from under the player) or unreadable; an absolute path still
	// works, as in a real shell.
	if parent, err := root.LookupDir(dirPath, ctx.User); err == nil {
		type entry struct {
			path []string
			f    *file.F
			dot  bool // `.` or `..`: a directory written as such
		}
		entries := map[string]entry{
			".":  {dirPath, parent, true},
			"..": {nil, nil, true},
		}
		if len(dirPath) > 0 {
			up := dirPath[:len(dirPath)-1]
			if f, err := root.Navigate(up); err == nil {
				entries[".."] = entry{up, f, true}
			}
		} else {
			entries[".."] = entry{nil, root, true}
		}
		for name, f := range parent.Files {
			entries[name] = entry{append(append([]string{}, dirPath...), name), f, false}
		}

		for name, e := range entries {
			if e.f == nil || !strings.HasPrefix(name, prefix) {
				continue
			}
			isFolder := e.f.Type == file.Folder
			if len(name) > len(prefix) {
				if accept(e.path, e.f, e.dot) || (isFolder && viable(e.path, e.f)) {
					chars.add(rune(name[len(prefix)]))
				}
			} else if isFolder && viable(e.path, e.f) {
				chars.add('/')
			}
		}
		if allowNew && (prefix == "" || prefix[0] != '-') && isWord(prefix) {
			for _, r := range wordChars {
				if prefix == "" && r == '-' {
					continue
				}
				chars.add(r)
			}
		}
	}

	complete := false
	if token != "" {
		target := file.Resolve(ctx.Cwd, token)
		dir := strings.HasSuffix(token, "/") || prefix == "." || prefix == ".."
		f, err := root.Lookup(target, ctx.User)
		switch {
		case err == nil:
			complete = accept(target, f, dir)
		case errors.Is(err, file.ErrNotExist):
			complete = allowNew && !dir && validNewName(prefix) && accept(target, nil, false)
		}
	}
	return simulation.Suggestion{Chars: sortedKeys(chars), Complete: complete}
}

// viableDir reports whether typing `/` after folder f (at path) can lead
// to an accepted path: the folder itself written as `dir/`, a new name
// inside it, or something accepted further down. Completion never looks
// inside a folder the user can't read: it may be named (`ls /root` is a
// real lesson in permission denied) but its contents stay hidden.
func viableDir(user string, path []string, f *file.F, allowNew bool, accept pathAccept) bool {
	if accept(path, f, true) {
		return true
	}
	if user != "" && !f.CanRead(user) {
		return false
	}
	if allowNew {
		return true
	}
	for name, child := range f.Files {
		p := append(append([]string{}, path...), name)
		if accept(p, child, false) {
			return true
		}
		if child.Type == file.Folder && viableDir(user, p, child, allowNew, accept) {
			return true
		}
	}
	return false
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

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// charSet collects suggested ASCII characters; sortedKeys lists them in
// order.
type charSet [128]bool

// add ignores non-ASCII: the keyboard is ASCII.
func (c *charSet) add(r rune) {
	if r >= 0 && r < 128 {
		c[r] = true
	}
}

func sortedKeys(c charSet) []rune {
	var out []rune
	for r, ok := range c {
		if ok {
			out = append(out, rune(r))
		}
	}
	return out
}
