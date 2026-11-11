package command

import (
	"errors"
	"slices"
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
// positional arguments, e.g. `mv SRC DST` or `grep PATTERN [FILE]`,
// optionally preceded by single-letter options (`ls -a DIR`).
type Positional struct {
	Args []Arg
	// Variadic repeats the last Arg: any number of them (echo WORD...).
	Variadic bool
	// Required is how many args a standalone run needs (options don't
	// count).
	Required int
	// Flags are the single-character options the command takes, each
	// at most once, before any argument, alone or bundled: `ls -a`,
	// `kill -9 PID`, `ps -e`. Commands read them with Options. As in
	// POSIX, the first word that isn't an option ends them.
	Flags string
	// Writes commands print their results to stdout (ls, cat, grep),
	// so `|` may follow them. Commands that only change the world (rm,
	// mv, touch) print nothing; piping them is never offered, which
	// also keeps a pipeline from changing the files a later segment
	// was validated against.
	Writes bool
	// ReadsStdin commands can take their input from an upstream pipe
	// instead of their trailing args: in a pipe they take exactly
	// PipeRequired args (`ls | cat`, `ls | grep x`). Only they can be
	// pipe receivers: `ls | ls` runs, but connects nothing.
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
	if ctx.InPipe {
		if !p.ReadsStdin {
			return simulation.Suggestion{}
		}
		p.Args, p.Required, p.Variadic = p.Args[:p.PipeRequired], p.PipeRequired, false
	}
	sug := p.validate(ctx, partial)
	sug.Writes = sug.Complete && p.Writes
	return sug
}

func (p Positional) validate(ctx simulation.ArgContext, partial string) simulation.Suggestion {
	tokens := strings.Split(partial, " ")
	i := len(tokens) - 1
	prev, cur := tokens[:i], tokens[i]

	// Options come first: every word starting with `-` until the first
	// argument.
	used := ""
	nf := 0
	for ; nf < len(prev) && p.isOption(prev[nf]); nf++ {
		letters, ok := p.optionLetters(prev[nf], used)
		if !ok || letters == "" {
			return simulation.Suggestion{}
		}
		used += letters
	}
	args := prev[nf:]
	for j, tok := range args {
		a := p.arg(j)
		if a == nil || tok == "" || !a(ctx, args[:j], tok).Complete {
			return simulation.Suggestion{}
		}
	}
	k := len(args) // cur is the k-th argument

	if k == 0 && p.isOption(cur) {
		return p.validateOption(ctx, cur, used)
	}

	a := p.arg(k)
	if a == nil {
		// Only the trailing space after the last argument, or the
		// options of a command that takes no arguments (`ps -e`).
		if cur != "" {
			return simulation.Suggestion{}
		}
		sug := simulation.Suggestion{Complete: p.enough(k)}
		if k == 0 && p.moreOptions(used) && p.afterOptions(ctx) {
			sug.Chars = []rune{'-'}
		}
		return sug
	}

	sug := a(ctx, args, cur)
	var chars []rune
	for _, r := range sug.Chars {
		if r != ' ' && r != '|' && !(k == 0 && cur == "" && r == '-' && p.Flags != "") {
			chars = append(chars, r)
		}
	}
	if cur == "" {
		if k == 0 && p.moreOptions(used) && p.afterOptions(ctx) {
			chars = append(chars, '-')
		}
		return simulation.Suggestion{Chars: chars, Complete: p.enough(k)}
	}
	if !sug.Complete {
		return simulation.Suggestion{Chars: chars}
	}
	complete := p.enough(k + 1)
	if next := p.arg(k + 1); complete || (next != nil && next(ctx, append(args, cur), "").Live()) {
		chars = append(chars, ' ')
	}
	return simulation.Suggestion{Chars: chars, Complete: complete}
}

// arg is the validator for the k-th argument, nil past the last.
func (p Positional) arg(k int) Arg {
	switch {
	case k < len(p.Args):
		return p.Args[k]
	case p.Variadic && len(p.Args) > 0:
		return p.Args[len(p.Args)-1]
	}
	return nil
}

func (p Positional) isOption(tok string) bool {
	return p.Flags != "" && strings.HasPrefix(tok, "-")
}

// optionLetters returns the letters of an option word (`-al` → "al"),
// false if any isn't a flag or is given twice.
func (p Positional) optionLetters(tok, used string) (string, bool) {
	letters := tok[1:]
	for i, r := range letters {
		if !strings.ContainsRune(p.Flags, r) || strings.ContainsRune(used, r) || strings.ContainsRune(letters[:i], r) {
			return "", false
		}
	}
	return letters, true
}

// moreOptions reports whether some flag hasn't been given yet.
func (p Positional) moreOptions(used string) bool {
	for _, r := range p.Flags {
		if !strings.ContainsRune(used, r) {
			return true
		}
	}
	return false
}

// afterOptions reports whether the command leads somewhere once its
// options are given: it can run, or its first argument can be typed.
func (p Positional) afterOptions(ctx simulation.ArgContext) bool {
	a := p.arg(0)
	return p.enough(0) || (a != nil && a(ctx, nil, "").Live())
}

// validateOption validates the option word being typed.
func (p Positional) validateOption(ctx simulation.ArgContext, cur, used string) simulation.Suggestion {
	letters, ok := p.optionLetters(cur, used)
	if !ok {
		return simulation.Suggestion{}
	}
	all := used + letters
	var chars []rune
	for _, r := range p.Flags {
		if !strings.ContainsRune(all, r) {
			chars = append(chars, r)
		}
	}
	sort.Slice(chars, func(i, j int) bool { return chars[i] < chars[j] })
	if letters == "" {
		return simulation.Suggestion{Chars: chars}
	}
	if p.afterOptions(ctx) {
		chars = append(chars, ' ')
	}
	return simulation.Suggestion{Chars: chars, Complete: p.enough(0)}
}

// Options splits the leading options off a command's args, the way
// Positional validated them: `-a -l x` → "al", ["x"]. A lone `-` is an
// argument.
func Options(args []string) (flags string, rest []string) {
	for i, a := range args {
		if len(a) < 2 || a[0] != '-' {
			return flags, args[i:]
		}
		flags += a[1:]
	}
	return flags, nil
}

// enough reports whether n complete args make a runnable command.
func (p Positional) enough(n int) bool {
	return n >= p.Required
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
		if !ok || !slices.Contains(names, name) {
			return simulation.Suggestion{}
		}
		return va(ctx, rest)
	}
	var chars charSet
	complete, writes := false, false
	for _, n := range names {
		if !strings.HasPrefix(n, name) {
			continue
		}
		va, ok := lookup(n)
		if !ok {
			continue
		}
		s := va(ctx, "")
		if !s.Live() {
			continue
		}
		if len(n) > len(name) {
			chars.add(rune(n[len(name)]))
		} else {
			chars.add(' ')
			complete, writes = s.Complete, s.Writes
		}
	}
	return simulation.Suggestion{Chars: sortedKeys(chars), Complete: complete, Writes: writes}
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

// Word accepts a free-form word, e.g. for echo: portable filename
// characters, which no shell interprets, and no leading `-` (an
// option).
func Word(_ simulation.ArgContext, _ []string, token string) simulation.Suggestion {
	return freeForm(wordChars, token)
}

// Pattern accepts a literal search word for grep. It is limited to
// characters that mean nothing special to a real shell or to grep's
// regular expressions, so what the player types behaves the same in
// real Unix. A leading `-` would be an option, so it is not offered.
func Pattern(_ simulation.ArgContext, _ []string, token string) simulation.Suggestion {
	return freeForm(patternChars, token)
}

// freeForm offers any of chars, except `-` first, and completes on any
// non-empty token.
func freeForm(chars, token string) simulation.Suggestion {
	var out []rune
	for _, r := range chars {
		if r != '-' || token != "" {
			out = append(out, r)
		}
	}
	return simulation.Suggestion{Chars: out, Complete: token != ""}
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
		return f != nil && f.Type == file.Folder && !slices.Equal(target, ctx.Cwd)
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

// NewName accepts a path that doesn't exist yet, in an existing folder
// (mkdir: making what is already there is an error).
func NewName(ctx simulation.ArgContext, _ []string, token string) simulation.Suggestion {
	return pathArg(ctx, token, true, func(target []string, f *file.F, dir bool) bool {
		return f == nil && !dir && len(target) > 0
	})
}

// EmptyDir accepts an existing, empty folder the user can read (rmdir).
// Folders the user can't read aren't offered: whether they are empty
// is theirs to know. `.`, `..` and `/` are never accepted, as real
// rmdir refuses them.
func EmptyDir(ctx simulation.ArgContext, _ []string, token string) simulation.Suggestion {
	return pathArg(ctx, token, false, func(target []string, f *file.F, dir bool) bool {
		return f != nil && f.Type == file.Folder && len(f.Files) == 0 &&
			!dir && len(target) > 0 && f.CanRead(ctx.User)
	})
}

// RedirectTarget accepts the file of a `>` or `>>` redirect: an
// existing text file or a new name in an existing folder, never a
// folder. It may not be a file the command line names in its args
// (operands): the shell empties it before the command runs, so `cat
// f > f` would read nothing, and real cat and grep refuse it ("input
// file is output file").
func RedirectTarget(ctx simulation.ArgContext, operands []string, token string) simulation.Suggestion {
	var named [][]string
	for _, o := range operands {
		named = append(named, file.Resolve(ctx.Cwd, file.ExpandHome(o, ctx.Home)))
	}
	return pathArg(ctx, token, true, func(target []string, f *file.F, dir bool) bool {
		if dir || len(target) == 0 {
			return false
		}
		if f == nil {
			return true
		}
		if f.Type != file.Text {
			return false
		}
		for _, n := range named {
			if slices.Equal(n, target) {
				return false
			}
		}
		return true
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
			if slices.Equal(into, src) {
				return false
			}
			existing, ok := f.Files[base]
			return !ok || existing.Type != file.Folder
		}
		return !dir && len(target) > 0 && !slices.Equal(target, src)
	})
}

// wordChars are the characters offered for free-form words (new file
// names, grep patterns): portable filename characters, none of which
// a real shell would interpret, limited to keys on the on-screen
// keyboard (no capitals or `_`), so a player who can only tap never
// sees a valid key they can't press.
const wordChars = "-.0123456789abcdefghijklmnopqrstuvwxyz"

// patternChars are wordChars without `.`, which real grep reads as "any
// character" while ours matches it literally.
const patternChars = "-0123456789abcdefghijklmnopqrstuvwxyz"

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
