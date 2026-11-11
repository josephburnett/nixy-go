package session

import (
	"sort"
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/guide"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

// keyboardScenario is a reproducible world state for the keyboard walk:
// a fresh logged-in session, optionally played forward until a quest is
// active, then a list of extra commands typed key by key.
type keyboardScenario struct {
	name       string
	untilQuest string              // follow the planner until this quest is active ("" = don't)
	install    map[string][]string // then put these commands in each host's /bin
	commands   []string            // then type these
}

var keyboardScenarios = []keyboardScenario{
	{name: "laptop-fresh"},
	{name: "nixy-fresh", commands: []string{"ssh nixy"}},
	{
		// Not reachable in play (nixy has no sudo), but it puts every
		// command's validator in front of nixy's world.
		name:       "nixy-home-all-installed",
		untilQuest: "modification",
		install:    map[string][]string{"nixy": {"mv", "sudo"}},
		commands:   []string{"cd /home/nixy"},
	},
	{
		name:       "nixy-projects",
		untilQuest: "composition",
		install:    map[string][]string{"nixy": {"mv"}},
		commands:   []string{"cd /home/nixy/projects"},
	},
	{
		name:       "server",
		untilQuest: "permissions",
		commands:   []string{"exit", "ssh server"},
	},
	{
		name:       "server-sudo-installed",
		untilQuest: "permissions",
		commands: []string{
			"exit", "ssh server", "sudo apt install touch", "sudo apt install rm",
			"sudo apt install mv", "sudo apt install grep",
		},
	},
}

// buildScenario creates a session in the scenario's world state. Every
// keystroke goes through the same validity check the player faces.
func buildScenario(t *testing.T, sc keyboardScenario) (*Session, *terminal.T) {
	t.Helper()
	sess, err := New()
	if err != nil {
		t.Fatal(err)
	}
	term := terminal.New(terminal.NewANSI())
	term.Resize(80, 24)
	sess.InitTerminal(term)
	typeAccepted(t, sess, term, "user")
	pressAccepted(t, sess, term, process.TermEnter)

	if sc.untilQuest != "" {
		for i := 0; ; i++ {
			a := sess.Game.Manager.ActiveQuest()
			if a == nil {
				t.Fatalf("%s: ran out of quests before %q", sc.name, sc.untilQuest)
			}
			if a.ID() == sc.untilQuest {
				break
			}
			if i > 100 {
				t.Fatalf("%s: never reached quest %q", sc.name, sc.untilQuest)
			}
			cmd := sess.Game.GetPlannedCommand(sess.Shell.Hostname(), sess.Shell.CurrentDirectory())
			if cmd == "" {
				t.Fatalf("%s: planner has no command for %q", sc.name, a.ID())
			}
			typeCommand(t, sess, term, cmd)
		}
	}
	for host, names := range sc.install {
		c, err := sess.Game.Sim.GetComputer(host)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range names {
			c.Filesystem.Files["bin"].Files[n] = &file.F{Type: file.Binary, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read, Data: n}
		}
	}
	sess.updatePlayingTerminal(term) // the keyboard sees what was installed
	for _, cmd := range sc.commands {
		typeCommand(t, sess, term, cmd)
	}
	return sess, term
}

// TestKeyboardWalk_NoMistakes is the structural guard for principle 1 over
// the real game worlds. In each scenario it walks the keyboard (BFS over
// valid Chars) and checks every reachable state:
//
//   - it is not a dead end: some key other than Backspace is valid;
//   - if Enter is valid, pressing it terminates (no hang) and writes
//     nothing to stderr except a permission error. Permission denied is
//     a real consequence (principle 2); anything else (usage errors,
//     missing files, wrong file type) is a keyboard bug.
//
// To reach deep states (mv's second argument, sudo's inner command,
// nested paths) with a small budget, the walk restarts from every state
// that just began something new — a space or `/` — giving each its own
// budget. The first seeds are the empty prompt and every command name,
// standalone and as a pipe receiver.
//
// Enter is checked on a fresh shell after restoring every filesystem to
// the scenario snapshot, because commands like rm and mv change it.
func TestKeyboardWalk_NoMistakes(t *testing.T) {
	const (
		statesPerSeed = 20  // per restart point
		statesPerRoot = 250 // per initial seed (command), including its restarts
		maxLen        = 50
		maxFanout     = 3 // sampled children per state, so free-form names can't eat the budget
		maxPolls      = 200
	)
	for _, sc := range keyboardScenarios {
		t.Run(sc.name, func(t *testing.T) {
			sess, _ := buildScenario(t, sc)
			w := newWalkWorld(t, sess)

			seen := map[string]bool{}
			var enterStates []string
			for _, root := range w.seeds() {
				if seen[root] || !w.typeable(root) {
					continue
				}
				seen[root] = true
				seeds := []string{root}
				for total := 0; len(seeds) > 0 && total < statesPerRoot; {
					queue := []string{seeds[0]}
					seeds = seeds[1:]
					for n := 0; len(queue) > 0 && n < statesPerSeed; n++ {
						total++
						cur := queue[0]
						queue = queue[1:]
						next := w.next(t, cur)

						var chars []string
						hasEnter, hasOther := false, false
						for _, d := range next {
							switch d := d.(type) {
							case process.Chars:
								chars = append(chars, string(d))
								hasOther = true
							case process.TermCode:
								if d == process.TermEnter {
									hasEnter = true
									hasOther = true
								}
							}
						}
						if !hasOther {
							t.Errorf("dead end: %q offers only Backspace", cur)
						}
						if hasEnter {
							enterStates = append(enterStates, cur)
						}
						if len(cur) >= maxLen {
							continue
						}
						sort.Strings(chars)
						picks := chars
						if len(chars) > maxFanout {
							picks = nil
							step := len(chars) / maxFanout
							for i := 0; i < maxFanout; i++ {
								picks = append(picks, chars[i*step])
							}
							for _, c := range chars {
								if c == " " || c == "/" || c == "~" || c == "|" {
									picks = append(picks, c)
								}
							}
						}
						for _, c := range picks {
							ns := cur + c
							// Skipped: `.`/`..` after the first path
							// component (aliases of paths already
							// walked).
							if strings.Contains(ns, "/.") || seen[ns] {
								continue
							}
							seen[ns] = true
							if c == " " || c == "/" || c == "|" {
								seeds = append(seeds, ns)
							}
							if c == " " {
								// Sampling rarely spells out an existing
								// name among free-form name chars, so
								// try each name in the cwd as the next
								// argument too.
								for _, name := range w.cwdNames() {
									if a := ns + name; !seen[a] && w.typeable(a) {
										seen[a] = true
										seeds = append(seeds, a)
									}
								}
							} else if c != "/" {
								queue = append(queue, ns)
							}
						}
					}
				}
			}
			if len(enterStates) == 0 {
				t.Fatal("walk reached no Enter-valid states")
			}
			for _, line := range enterStates {
				w.checkEnter(t, line, maxPolls)
			}
			t.Logf("%d states walked, %d Enter states checked", len(seen), len(enterStates))
		})
	}
}

// walkWorld is a scenario's world frozen for the keyboard walk: the
// player's host and cwd, plus a snapshot of every filesystem to restore
// between Enter checks.
type walkWorld struct {
	sess      *Session
	host      string
	cwd       []string
	snapshots map[string]*file.F
	g         *guide.G
}

func newWalkWorld(t *testing.T, sess *Session) *walkWorld {
	t.Helper()
	w := &walkWorld{
		sess:      sess,
		host:      sess.Shell.Hostname(),
		cwd:       append([]string{}, sess.Shell.CurrentDirectory()...),
		snapshots: map[string]*file.F{},
	}
	for _, h := range []string{"laptop", "nixy", "server"} {
		if c, err := sess.Game.Sim.GetComputer(h); err == nil {
			w.snapshots[h] = copyFS(c.Filesystem)
		}
	}
	w.g = w.freshShell(t)
	return w
}

// freshShell restores every filesystem and launches a new shell for the
// player at the scenario's host and cwd.
func (w *walkWorld) freshShell(t *testing.T) *guide.G {
	t.Helper()
	for h, snap := range w.snapshots {
		c, _ := w.sess.Game.Sim.GetComputer(h)
		*c.Filesystem = *copyFS(snap)
	}
	p, err := w.sess.Game.Sim.Launch(w.host, w.sess.Username, "shell", nil, append([]string{}, w.cwd...))
	if err != nil {
		t.Fatal(err)
	}
	return guide.New(p)
}

// seeds returns the walk's starting lines: the empty prompt, and each
// command name followed by a space, standalone and as a pipe receiver.
func (w *walkWorld) seeds() []string {
	names := []string{"cd", "exit", "nx"}
	if bin, err := w.snapshots[w.host].Navigate([]string{"bin"}); err == nil {
		for n := range bin.Files {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	seeds := []string{"", "ls |", "ls | "}
	for _, n := range names {
		seeds = append(seeds, n+" ", "ls | "+n, "ls | "+n+" ")
	}
	return seeds
}

// cwdNames lists the entries of the scenario's cwd.
func (w *walkWorld) cwdNames() []string {
	dir, err := w.snapshots[w.host].Navigate(w.cwd)
	if err != nil {
		return nil
	}
	var names []string
	for n := range dir.Files {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// typeable reports whether every key of line is offered as it is typed.
func (w *walkWorld) typeable(line string) bool {
	ok := true
	n := 0
	for _, r := range line {
		if err := w.g.Stdin(process.Data{process.Chars(string(r))}); err != nil {
			ok = false
			break
		}
		n++
	}
	for range n {
		_ = w.g.Stdin(process.Data{process.TermBackspace})
	}
	drain(w.g)
	return ok
}

// next returns the valid keys after typing line (which must be typeable).
func (w *walkWorld) next(t *testing.T, line string) []process.Datum {
	t.Helper()
	for _, r := range line {
		if err := w.g.Stdin(process.Data{process.Chars(string(r))}); err != nil {
			t.Fatalf("replaying %q: %q rejected", line, string(r))
		}
	}
	next := w.g.Next()
	for range line {
		_ = w.g.Stdin(process.Data{process.TermBackspace})
	}
	drain(w.g)
	return next
}

// checkEnter types line on a fresh world, presses Enter, and verifies the
// command terminates without a non-permission error. A pipeline must
// also leave every file as it was: only commands that print can feed a
// pipe, so no segment can change what a later one was validated
// against. A redirect is opened by the shell before any segment runs.
func (w *walkWorld) checkEnter(t *testing.T, line string, maxPolls int) {
	t.Helper()
	g := w.freshShell(t)
	before := w.dumpWorld()
	for _, r := range line {
		if err := g.Stdin(process.Data{process.Chars(string(r))}); err != nil {
			if strings.Contains(line, "kill ") {
				// An earlier check killed that process; files are
				// restored between checks, the process table isn't.
				return
			}
			t.Errorf("%q can't be typed on a fresh shell (%q rejected)", line, string(r))
			return
		}
	}
	drain(g)
	_ = g.Stdin(process.Data{process.TermEnter})

	var stderr strings.Builder
	interrupted := false
	for i := 0; ; i++ {
		if i >= maxPolls {
			// A command may run forever (yes), but then Ctrl+C must be
			// offered and must end it.
			if interrupted || g.Stdin(process.Data{process.SigInt}) != nil {
				t.Errorf("hang: %q + Enter did not return to a prompt", line)
				return
			}
			interrupted, i = true, 0
		}
		stderr.WriteString(process.Text(g.Read(), process.Stderr))
		if _, exited := g.Exit(); exited || isFreshPrompt(g.Next()) {
			break
		}
	}
	stderr.WriteString(process.Text(g.Read(), process.Stderr))
	if msg := stderr.String(); !consequence(msg) {
		t.Errorf("mistake: %q + Enter wrote %q", line, msg)
	}
	// (A redirect changes its target, and only at the end of the line.)
	if strings.Contains(line, "|") && !strings.Contains(line, ">") && w.dumpWorld() != before {
		t.Errorf("pipeline %q changed the world", line)
	}
}

func (w *walkWorld) dumpWorld() string {
	var sb strings.Builder
	for _, h := range []string{"laptop", "nixy", "server"} {
		if c, err := w.sess.Game.Sim.GetComputer(h); err == nil {
			dumpFS(&sb, c.Filesystem, h+":/")
		}
	}
	return sb.String()
}

func copyFS(f *file.F) *file.F {
	c := *f
	if f.Files != nil {
		c.Files = make(map[string]*file.F, len(f.Files))
		for n, child := range f.Files {
			c.Files[n] = copyFS(child)
		}
	}
	return &c
}

func drain(g interface{ Read() []process.Output }) {
	for range 50 {
		if len(g.Read()) == 0 {
			return
		}
	}
}

// isFreshPrompt reports whether the keyboard looks like an empty prompt:
// command first-chars and no Backspace.
func isFreshPrompt(next []process.Datum) bool {
	chars := 0
	for _, d := range next {
		switch d := d.(type) {
		case process.TermCode:
			if d == process.TermBackspace {
				return false
			}
		case process.Chars:
			chars++
		}
	}
	return chars > 0
}

// typeCommand types cmd key by key and presses Enter, failing if any key
// is not in the valid set when it is pressed.
func typeCommand(t *testing.T, s *Session, term *terminal.T, cmd string) {
	t.Helper()
	typeAccepted(t, s, term, cmd)
	pressAccepted(t, s, term, process.TermEnter)
}

func typeAccepted(t *testing.T, s *Session, term *terminal.T, str string) {
	t.Helper()
	for _, r := range str {
		pressAccepted(t, s, term, process.Chars(string(r)))
	}
}

// pressAccepted presses one key after checking that the keyboard offers
// it, and that the hint (if any) is itself offered.
func pressAccepted(t *testing.T, s *Session, term *terminal.T, d process.Datum) {
	t.Helper()
	valid := term.State.ValidKeys
	if hint := term.State.HintKey; hint != nil && !containsKey(valid, hint) {
		t.Fatalf("hint %q is not a valid key (line %q)", keyName(hint), term.State.Line)
	}
	if !containsKey(valid, d) {
		t.Fatalf("key %q not accepted at line %q", keyName(d), term.State.Line)
	}
	pressKey(t, s, term, d)
}

func containsKey(keys []process.Datum, d process.Datum) bool {
	for _, k := range keys {
		if k == d {
			return true
		}
	}
	return false
}

func keyName(d process.Datum) string {
	switch d := d.(type) {
	case process.Chars:
		return string(d)
	case process.TermCode:
		switch d {
		case process.TermEnter:
			return "<Enter>"
		case process.TermBackspace:
			return "<Backspace>"
		}
	}
	return "?"
}
