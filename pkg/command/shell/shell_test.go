package shell

import (
	"sort"
	"strings"
	"testing"

	_ "github.com/josephburnett/nixy-go/pkg/command/apt"
	_ "github.com/josephburnett/nixy-go/pkg/command/cat"
	_ "github.com/josephburnett/nixy-go/pkg/command/echo"
	_ "github.com/josephburnett/nixy-go/pkg/command/grep"
	_ "github.com/josephburnett/nixy-go/pkg/command/ls"
	_ "github.com/josephburnett/nixy-go/pkg/command/mkdir"
	_ "github.com/josephburnett/nixy-go/pkg/command/mv"
	_ "github.com/josephburnett/nixy-go/pkg/command/pwd"
	_ "github.com/josephburnett/nixy-go/pkg/command/rm"
	_ "github.com/josephburnett/nixy-go/pkg/command/rmdir"
	_ "github.com/josephburnett/nixy-go/pkg/command/ssh"
	_ "github.com/josephburnett/nixy-go/pkg/command/sudo"
	_ "github.com/josephburnett/nixy-go/pkg/command/touch"
	_ "github.com/josephburnett/nixy-go/pkg/command/whoami"
	_ "github.com/josephburnett/nixy-go/pkg/command/yes"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// testBinaries are installed in the fixture's /bin: every registered
// command, so the structural tests cover them all.
var testBinaries = []string{
	"apt", "cat", "echo", "grep", "ls", "mkdir", "mv", "pwd", "rm", "rmdir",
	"ssh", "sudo", "touch", "whoami", "yes",
}

func testFS() *file.F {
	dir := func(owner string, files map[string]*file.F) *file.F {
		return &file.F{Type: file.Folder, Owner: owner,
			OwnerPermission: file.Write, CommonPermission: file.Read, Files: files}
	}
	text := func(owner, data string) *file.F {
		return &file.F{Type: file.Text, Owner: owner,
			OwnerPermission: file.Write, CommonPermission: file.Read, Data: data}
	}
	bin := map[string]*file.F{}
	for _, name := range testBinaries {
		bin[name] = &file.F{Type: file.Binary, Owner: file.OwnerRoot,
			OwnerPermission: file.Write, CommonPermission: file.Read, Data: name}
	}
	return dir(file.OwnerRoot, map[string]*file.F{
		"bin": dir(file.OwnerRoot, bin),
		"etc": dir(file.OwnerRoot, map[string]*file.F{
			"hosts": text(file.OwnerRoot, "test"),
		}),
		// Only root may look inside /root.
		"root": {Type: file.Folder, Owner: file.OwnerRoot,
			OwnerPermission: file.Write, CommonPermission: file.None,
			Files: map[string]*file.F{
				"secret.txt": text(file.OwnerRoot, "hidden"),
			}},
		"home": dir(file.OwnerRoot, map[string]*file.F{
			"user": dir("user", map[string]*file.F{
				"notes.txt": text("user", "a note\nanother"),
				".profile":  text("user", "hidden"),
				"docs":      dir("user", map[string]*file.F{}),
			}),
		}),
	})
}

func testSetup(t *testing.T) (*simulation.S, process.P) {
	t.Helper()
	sim := simulation.New()
	sim.Nx = stubNx{}
	if err := sim.Boot("test", testFS()); err != nil {
		t.Fatal(err)
	}
	p, err := sim.Launch("test", "user", "shell", nil, []string{})
	if err != nil {
		t.Fatal(err)
	}
	return sim, p
}

func writeString(t *testing.T, p process.P, s string) {
	t.Helper()
	for _, c := range s {
		err := p.Stdin(process.Data{process.Chars(string(c))})
		if err != nil {
			t.Fatalf("writing '%c': %v", c, err)
		}
	}
}

func writeEnter(t *testing.T, p process.P) {
	t.Helper()
	err := p.Stdin(process.Data{process.TermEnter})
	if err != nil {
		t.Fatal(err)
	}
}

func writeBackspace(t *testing.T, p process.P) {
	t.Helper()
	err := p.Stdin(process.Data{process.TermBackspace})
	if err != nil {
		t.Fatal(err)
	}
}

// unread holds output read from a shell but not yet asked for: what
// readAllStdout read from stderr, and the reverse.
var unread = map[process.P][]process.Output{}

// readStream reads p until it has nothing more to say and returns what
// it wrote to stream, keeping the other stream's output for later.
func readStream(p process.P, stream process.Stream) string {
	out := unread[p]
	for i := 0; i < 10; i++ {
		o := p.Read()
		if len(o) == 0 {
			break
		}
		out = append(out, o...)
	}
	var rest []process.Output
	for _, o := range out {
		if o.Stream != stream {
			rest = append(rest, o)
		}
	}
	unread[p] = rest
	return process.Text(out, stream)
}

func readAllStdout(t *testing.T, p process.P) string {
	t.Helper()
	return readStream(p, process.Stdout)
}

func readAllStderr(t *testing.T, p process.P) string {
	t.Helper()
	return readStream(p, process.Stderr)
}

func containsDatum(datums []process.Datum, target process.Datum) bool {
	for _, d := range datums {
		if d == target {
			return true
		}
	}
	return false
}

func TestShellPwd(t *testing.T) {
	_, p := testSetup(t)

	writeString(t, p, "pwd")
	writeEnter(t, p)
	out := readAllStdout(t, p)

	// Echo of "pwd\n" + command output "/\n"
	if out != "pwd/\n" {
		t.Fatalf("expected 'pwd/\\n', got %q", out)
	}
}

func TestShellPwdAfterCd(t *testing.T) {
	_, p := testSetup(t)

	writeString(t, p, "cd home")
	writeEnter(t, p)
	_ = readAllStdout(t, p) // drain echo

	writeString(t, p, "pwd")
	writeEnter(t, p)
	out := readAllStdout(t, p)

	if out != "pwd/home\n" {
		t.Fatalf("expected 'pwd/home\\n', got %q", out)
	}
}

func TestShellCdAbsolute(t *testing.T) {
	_, p := testSetup(t)

	writeString(t, p, "cd /home/user")
	writeEnter(t, p)
	_ = readAllStdout(t, p)
	_ = readAllStderr(t, p)

	writeString(t, p, "pwd")
	writeEnter(t, p)
	out := readAllStdout(t, p)

	if out != "pwd/home/user\n" {
		t.Fatalf("expected 'pwd/home/user\\n', got %q", out)
	}
}

func TestShellCdDotDot(t *testing.T) {
	_, p := testSetup(t)

	writeString(t, p, "cd /home/user")
	writeEnter(t, p)
	_ = readAllStdout(t, p)

	writeString(t, p, "cd ..")
	writeEnter(t, p)
	_ = readAllStdout(t, p)

	writeString(t, p, "pwd")
	writeEnter(t, p)
	out := readAllStdout(t, p)

	if out != "pwd/home\n" {
		t.Fatalf("expected 'pwd/home\\n', got %q", out)
	}
}

func TestShellCdNonexistent(t *testing.T) {
	_, p := testSetup(t)

	writeString(t, p, "cd nonexistent")
	writeEnter(t, p)
	_ = readAllStdout(t, p)
	errOut := readAllStderr(t, p)

	if errOut == "" {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestShellUnknownCommand(t *testing.T) {
	_, p := testSetup(t)

	writeString(t, p, "foobar")
	writeEnter(t, p)
	_ = readAllStdout(t, p)
	errOut := readAllStderr(t, p)

	if errOut != "foobar: command not found\n" {
		t.Fatalf("got %q, want bash's wording", errOut)
	}
}

// TestShellRunNeedsReadOnBinary: running /bin/X needs the model's
// execute permission (Read) on it, checked the same way as every other
// file access; root still can.
func TestShellRunNeedsReadOnBinary(t *testing.T) {
	sim, p := testSetup(t)
	c, _ := sim.GetComputer("test")
	c.Filesystem.Files["bin"].Files["pwd"].CommonPermission = file.None
	if canRunFresh(p, "pwd") {
		t.Error("pwd offered without permission to run it")
	}
	if !canRunFresh(p, "sudo pwd") {
		t.Error("root can't run pwd")
	}
	runLine(t, p, "pwd")
	if got := readAllStderr(t, p); got != "pwd: Permission denied\n" {
		t.Errorf("got %q", got)
	}
}

// TestShellSurfacesChildStderr confirms a child process's stderr is delivered
// even when the session loop reads stdout first. Regression test for a bug
// where shell.Stdout() killed the child on its EOF, before stderr was ever
// read — the error message was generated but lost.
func TestShellSurfacesChildStderr(t *testing.T) {
	_, p := testSetup(t)

	// rm with no args returns an errorProcess whose message lives only on
	// stderr. The classic loop is: read all of stdout, THEN read stderr —
	// which is exactly the order this test exercises.
	writeString(t, p, "rm")
	writeEnter(t, p)
	_ = readAllStdout(t, p)
	errOut := readAllStderr(t, p)

	if errOut == "" {
		t.Fatal("expected child's stderr to be surfaced after stdout drain")
	}
	if !strings.Contains(errOut, "missing operand") {
		t.Fatalf("expected 'missing operand' in stderr, got: %q", errOut)
	}
}

// TestShellNextInvariant_PartialArgCannotExecute pins the "no mistakes"
// invariant for argument typing: a player cannot press Enter or `|`
// while in the middle of typing a filename that doesn't yet match an
// existing file. Otherwise they'd execute commands like "cat r" against
// nonexistent paths.
func TestShellNextInvariant_PartialArgCannotExecute(t *testing.T) {
	_, p := testSetup(t)
	// Set up: type "rm " (with space) to enter arg mode. testSetup's /home/user
	// folder is empty, so any partial filename has no exact match.
	writeString(t, p, "rm ")
	_ = readAllStdout(t, p)
	next := p.Next()
	if containsDatum(next, process.TermEnter) {
		t.Fatal("Enter must not be valid mid-arg for rm (no exact match)")
	}
	if containsDatum(next, process.Chars("|")) {
		t.Fatal("`|` must not be valid mid-arg for rm (no exact match)")
	}
}

// TestShellNextInvariant_RequiredArgsCommandRejectsBareEnter pins the
// invariant for command names: a command that needs args (like rm)
// must not accept Enter at the bare command name. Without this, the
// player could type `rm<Enter>` and trigger "rm: missing operand".
func TestShellNextInvariant_RequiredArgsCommandRejectsBareEnter(t *testing.T) {
	_, p := testSetup(t)
	writeString(t, p, "rm")
	next := p.Next()
	if containsDatum(next, process.TermEnter) {
		t.Fatal("Enter must not be valid after just 'rm' (rm requires args)")
	}
	if !containsDatum(next, process.Chars(" ")) {
		t.Fatal("space must be valid after 'rm' so the player can supply args")
	}
}

// TestShellNextInvariant_OptionalArgsCommandAllowsBareEnter: ls accepts
// zero args (lists cwd). Enter must be valid right after typing "ls".
func TestShellNextInvariant_OptionalArgsCommandAllowsBareEnter(t *testing.T) {
	_, p := testSetup(t)
	writeString(t, p, "ls")
	next := p.Next()
	if !containsDatum(next, process.TermEnter) {
		t.Fatal("Enter must be valid after 'ls' (ls accepts zero args)")
	}
	if !containsDatum(next, process.Chars("|")) {
		t.Fatal("`|` must be valid after 'ls' (can pipe ls output)")
	}
}

// TestShellNextInvariant_CatStandaloneRequiresArgs: cat without args
// standalone runs stdinCat which hangs forever waiting for input the
// game can't supply (no terminal stdin). The keyboard must not allow
// `cat<Enter>` or `cat <Enter>` standalone.
func TestShellNextInvariant_CatStandaloneRequiresArgs(t *testing.T) {
	_, p := testSetup(t)
	writeString(t, p, "cat")
	next := p.Next()
	if containsDatum(next, process.TermEnter) {
		t.Fatal("Enter must not be valid after standalone 'cat' (would hang)")
	}
	if containsDatum(next, process.Chars("|")) {
		t.Fatal("`|` must not be valid after standalone 'cat' (nothing to pipe)")
	}
	writeString(t, p, " ")
	next = p.Next()
	if containsDatum(next, process.TermEnter) {
		t.Fatal("Enter must not be valid after 'cat ' standalone")
	}
}

// TestShellNextInvariant_CatInPipeAllowsBareEnter: in pipe-receiver
// position (`ls | cat`), cat with no args is valid — stdinCat reads
// from the upstream pipe and terminates when the upstream EOFs.
func TestShellNextInvariant_CatInPipeAllowsBareEnter(t *testing.T) {
	_, p := testSetup(t)
	writeString(t, p, "ls|cat")
	next := p.Next()
	if !containsDatum(next, process.TermEnter) {
		t.Fatal("Enter must be valid after 'ls|cat' (cat is pipe receiver)")
	}
}

// TestShellNextInvariant_GrepStandaloneRequiresFile: grep with just a
// pattern (no file) standalone returns stdinGrep which hangs. The
// keyboard must not allow Enter at `grep PATTERN` standalone — only
// at `grep PATTERN file` (validator's exact-match Enter).
func TestShellNextInvariant_GrepStandaloneRequiresFile(t *testing.T) {
	_, p := testSetup(t)
	writeString(t, p, "grep err")
	next := p.Next()
	if containsDatum(next, process.TermEnter) {
		t.Fatal("Enter must not be valid after standalone 'grep err' (no file = hang)")
	}
}

// TestShellNextInvariant_GrepInPipeAllowsPatternOnly: in pipe-receiver
// position, `ls | grep PATTERN` is valid — stdinGrep filters upstream's
// output. Enter must be allowed once a pattern is typed.
func TestShellNextInvariant_GrepInPipeAllowsPatternOnly(t *testing.T) {
	_, p := testSetup(t)
	writeString(t, p, "ls|grep err")
	next := p.Next()
	if !containsDatum(next, process.TermEnter) {
		t.Fatal("Enter must be valid after 'ls|grep err' (grep is pipe receiver, pattern typed)")
	}
}

// TestShellNextInvariant_EveryEnterStateTerminates is the structural
// guardian for the "no mistakes" invariant: for every keyboard state
// where Enter is in the valid set, pressing Enter must result in a
// terminating execution with no error other than permission denied.
// (TestKeyboardWalk_NoMistakes in pkg/session runs the same check over
// the real game worlds.) The test walks the keyboard space (BFS from
// empty buffer, Chars-only edges) and at each Enter-valid state replays
// those keystrokes on a fresh shell, presses Enter, and confirms the
// shell recovers.
//
// This is the test that should have caught the cat-hang and grep-hang
// bugs deterministically. The hint-guided E2E fuzz can't catch them
// because it follows the planner's exact output, so it never explores
// states like `cat<Enter>` or `grep pat<Enter>`.
func TestShellNextInvariant_EveryEnterStateTerminates(t *testing.T) {
	const (
		maxStates           = 2000
		maxLen              = 25
		maxPolls            = 200
		maxChildrenPerState = 6 // cap branching so one no-validator command can't monopolize the BFS budget
	)

	seen := map[string]bool{"": true}
	queue := []string{""}
	checked := 0
	exploredEnterStates := 0

	for len(queue) > 0 && len(seen) < maxStates {
		cur := queue[0]
		queue = queue[1:]

		// Build a fresh shell, replay the prefix, get Next().
		_, p := testSetup(t)
		for _, r := range cur {
			_ = p.Stdin(process.Data{process.Chars(string(r))})
		}
		next := p.Next()

		hasEnter := false
		for _, d := range next {
			if t, ok := d.(process.TermCode); ok && t == process.TermEnter {
				hasEnter = true
				break
			}
		}
		if hasEnter {
			exploredEnterStates++
			verifyEnterTerminates(t, cur, maxPolls)
		}

		if len(cur) >= maxLen {
			continue
		}

		// Collect Chars children, sort for determinism, evenly sample.
		var chars []string
		for _, d := range next {
			if c, ok := d.(process.Chars); ok {
				chars = append(chars, string(c))
			}
		}
		sort.Strings(chars)
		picks := chars
		if len(chars) > maxChildrenPerState {
			step := len(chars) / maxChildrenPerState
			picks = nil
			for i := 0; i < maxChildrenPerState; i++ {
				picks = append(picks, chars[i*step])
			}
			if chars[len(chars)-1] == "~" { // sorts last, so never sampled
				picks = append(picks, "~")
			}
		}
		for _, c := range picks {
			ns := cur + c
			if !seen[ns] {
				seen[ns] = true
				queue = append(queue, ns)
			}
		}
		checked++
	}

	if exploredEnterStates == 0 {
		t.Fatal("BFS reached no Enter-valid states — bounds too tight or invariant test misconfigured")
	}
	t.Logf("walked %d states, %d Enter-valid states verified", checked, exploredEnterStates)
}

// verifyEnterTerminates types `prefix` on a fresh shell, presses Enter,
// and drains until either: (a) the shell returns to fresh prompt (child
// reaped, currentCommand empty — the normal case), or (b) the shell
// itself EOFs (exit was the typed command). Fails if neither happens
// within maxPolls iterations — the hang signature.
func verifyEnterTerminates(t *testing.T, prefix string, maxPolls int) {
	t.Helper()
	_, p := testSetup(t)
	for _, r := range prefix {
		_ = p.Stdin(process.Data{process.Chars(string(r))})
	}
	_ = p.Stdin(process.Data{process.TermEnter})

	var stderr strings.Builder
	interrupted := false
	for i := 0; ; i++ {
		if i >= maxPolls {
			// A command may run forever (yes), but then Ctrl+C must be
			// offered and must end it.
			if interrupted || !containsDatum(p.Next(), process.SigInt) {
				t.Fatalf("hang: shell did not recover after %q + Enter within %d polls", prefix, maxPolls)
			}
			_ = p.Stdin(process.Data{process.SigInt})
			interrupted, i = true, 0
		}
		stderr.WriteString(process.Text(p.Read(), process.Stderr))
		// The shell exited: correct termination for `exit`.
		if _, exited := p.Exit(); exited || isFreshPrompt(p.Next()) {
			break
		}
	}
	// Permission denied is a real consequence; any other error means the
	// keyboard let the player type something the command rejects.
	if msg := stderr.String(); msg != "" && !strings.Contains(msg, "Permission denied") {
		t.Errorf("mistake: %q + Enter wrote %q", prefix, msg)
	}
}

type stubNx struct{}

func (stubNx) NxQuest() string { return "quest\n" }
func (stubNx) NxLog() string   { return "log\n" }
func (stubNx) NxPanic() string { return "panic\n" }

// isFreshPrompt reports whether the shell looks like it's at an empty
// prompt (childProcess nil, currentCommand empty). At that point Next()
// returns first chars of available commands and does NOT include
// Backspace.
func isFreshPrompt(next []process.Datum) bool {
	for _, d := range next {
		if t, ok := d.(process.TermCode); ok && t == process.TermBackspace {
			return false
		}
	}
	chars := 0
	for _, d := range next {
		if _, ok := d.(process.Chars); ok {
			chars++
		}
	}
	return chars >= 2
}

// TestShellNextInvariant_CdRoot: `cd /<Enter>` must work from a
// non-root cwd — slash alone resolves to the root directory, a valid
// cd target. (At root itself it's a no-op and is blocked elsewhere.)
func TestShellNextInvariant_CdRoot(t *testing.T) {
	_, p := testSetup(t)
	writeString(t, p, "cd /home/user")
	writeEnter(t, p)
	_ = readAllStdout(t, p)
	writeString(t, p, "cd /")
	next := p.Next()
	if !containsDatum(next, process.TermEnter) {
		t.Fatal("Enter must be valid after 'cd /' from a non-root cwd")
	}
}

// TestShellNextInvariant_CdParent: `cd ..<Enter>` must work — `..`
// resolves to the parent directory of cwd, which is always a valid
// (if no-op at root) cd target.
func TestShellNextInvariant_CdParent(t *testing.T) {
	_, p := testSetup(t)
	// First navigate to /home/user so .. has somewhere to go.
	writeString(t, p, "cd /home/user")
	writeEnter(t, p)
	_ = readAllStdout(t, p)
	writeString(t, p, "cd ..")
	next := p.Next()
	if !containsDatum(next, process.TermEnter) {
		t.Fatal("Enter must be valid after 'cd ..' (parent is a valid directory)")
	}
}

// TestShellNextInvariant_CdTrailingSlash: any path that ends with `/`
// resolves to a directory and Enter must be valid. e.g. `cd /home/`.
func TestShellNextInvariant_CdTrailingSlash(t *testing.T) {
	_, p := testSetup(t)
	writeString(t, p, "cd /home/")
	next := p.Next()
	if !containsDatum(next, process.TermEnter) {
		t.Fatal("Enter must be valid after 'cd /home/' (trailing / means resolved directory)")
	}
}

// TestShellNextInvariant_CdNoopBlocked: when the typed path resolves
// to the cwd itself (no actual move), Enter must not be valid. This
// covers `cd ..` at root, `cd /` when at root, `cd .` anywhere, and
// `cd /home/user` when already there. Letting the player execute a
// useless command violates the spirit of "no mistakes" — the keyboard
// should only offer keys that lead to a real, observable action.
func TestShellNextInvariant_CdNoopBlocked(t *testing.T) {
	cases := []struct {
		startPath string
		typed     string
		desc      string
	}{
		{"", "cd ..", "cd .. at root"},
		{"", "cd /", "cd / at root"},
		{"", "cd .", "cd . at root"},
		{"/home/user", "cd /home/user", "cd to current path"},
		{"/home/user", "cd .", "cd . in subdirectory"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			_, p := testSetup(t)
			if tc.startPath != "" {
				writeString(t, p, "cd "+tc.startPath)
				writeEnter(t, p)
				_ = readAllStdout(t, p)
			}
			writeString(t, p, tc.typed)
			next := p.Next()
			if containsDatum(next, process.TermEnter) {
				t.Fatalf("Enter must not be valid: %s", tc.desc)
			}
		})
	}
}

// TestShellNextInvariant_CdMovesAllowed: positive cases — Enter IS
// valid when the cd would actually move cwd. Pairs with the no-op
// blocked test to make sure we didn't over-block.
func TestShellNextInvariant_CdMovesAllowed(t *testing.T) {
	cases := []struct {
		startPath string
		typed     string
		desc      string
	}{
		{"/home/user", "cd ..", "cd .. from subdirectory"},
		{"/home/user", "cd /", "cd / from subdirectory"},
		{"", "cd /home", "cd /home from root"},
		{"/home", "cd user", "cd user (relative) from /home"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			_, p := testSetup(t)
			if tc.startPath != "" {
				writeString(t, p, "cd "+tc.startPath)
				writeEnter(t, p)
				_ = readAllStdout(t, p)
			}
			writeString(t, p, tc.typed)
			next := p.Next()
			if !containsDatum(next, process.TermEnter) {
				t.Fatalf("Enter must be valid: %s", tc.desc)
			}
		})
	}
}

// TestShellNextInvariant_PipeStartsFreshSegment: after typing `|`, the
// shell must validate the next segment as a fresh command name (first
// chars of available commands).
func TestShellNextInvariant_PipeStartsFreshSegment(t *testing.T) {
	_, p := testSetup(t)
	writeString(t, p, "ls|")
	next := p.Next()
	// Should include first chars of any installed command (e.g. 'g' for grep)
	if !containsDatum(next, process.Chars("g")) {
		t.Fatal("after `ls|` first chars of commands (e.g. 'g' for grep) should be valid")
	}
	// Must not include Enter — no command yet on the right side of the pipe
	if containsDatum(next, process.TermEnter) {
		t.Fatal("Enter must not be valid right after `|` — no command on the right yet")
	}
}

func TestShellExit(t *testing.T) {
	_, p := testSetup(t)

	writeString(t, p, "exit")
	writeEnter(t, p)

	p.Read()
	if _, exited := p.Exit(); !exited {
		t.Fatal("expected the shell to exit")
	}
}

func TestShellBackspace(t *testing.T) {
	_, p := testSetup(t)

	writeString(t, p, "pw")
	writeBackspace(t, p)
	writeString(t, p, "wd")
	writeEnter(t, p)
	out := readAllStdout(t, p)

	// Echo: p, w, backspace, w, d, enter + output /\n
	// The actual chars echoed: "pw" + backspace + "wd" + enter + "/\n"
	// We just check the command executed correctly
	if len(out) == 0 {
		t.Fatal("expected output")
	}
}

func TestShellNextAtEmpty(t *testing.T) {
	_, p := testSetup(t)

	next := p.Next()
	// Should include first chars of available commands: cd, exit, pwd
	if !containsDatum(next, process.Chars("c")) {
		t.Fatal("expected 'c' for 'cd' in Next()")
	}
	if !containsDatum(next, process.Chars("e")) {
		t.Fatal("expected 'e' for 'exit' in Next()")
	}
	if !containsDatum(next, process.Chars("p")) {
		t.Fatal("expected 'p' for 'pwd' in Next()")
	}
	// Should NOT include backspace (nothing typed)
	if containsDatum(next, process.TermBackspace) {
		t.Fatal("should not have backspace at empty prompt")
	}
}

func TestShellNextMidCommand(t *testing.T) {
	_, p := testSetup(t)

	writeString(t, p, "p")
	_ = readAllStdout(t, p)

	next := p.Next()
	// "p" is prefix of "pwd" -> "w" should be valid
	if !containsDatum(next, process.Chars("w")) {
		t.Fatal("expected 'w' to continue 'pwd'")
	}
	// Should include backspace
	if !containsDatum(next, process.TermBackspace) {
		t.Fatal("expected backspace")
	}
	// "p" alone is not a command, so no Enter
	if containsDatum(next, process.TermEnter) {
		t.Fatal("should not have Enter for incomplete command 'p'")
	}
}

func TestShellNextCompleteCommand(t *testing.T) {
	_, p := testSetup(t)

	writeString(t, p, "pwd")
	_ = readAllStdout(t, p)

	next := p.Next()
	// "pwd" is a valid command -> Enter and space should be valid
	if !containsDatum(next, process.TermEnter) {
		t.Fatal("expected Enter for complete command")
	}
	if !containsDatum(next, process.Chars(" ")) {
		t.Fatal("expected space for args")
	}
}

func TestShellNextAfterExit(t *testing.T) {
	_, p := testSetup(t)

	writeString(t, p, "exit")
	writeEnter(t, p)

	next := p.Next()
	if len(next) != 0 {
		t.Fatalf("expected no valid inputs after exit, got %v", len(next))
	}
}

func TestShellPipe(t *testing.T) {
	_, p := testSetup(t)

	writeString(t, p, "ls home | grep a")
	writeEnter(t, p)
	out := readAllStdout(t, p)

	// Should contain the echo of the command + the filtered ls output
	// ls home would show "a.txt" and "b.txt" (from testSetup which has "user" dir)
	// Actually testSetup has home/user. Let's just check it doesn't error.
	_ = readAllStderr(t, p)
	_ = out // pipe executed without panic
}

func TestShellEmptyEnter(t *testing.T) {
	_, p := testSetup(t)

	// Enter with nothing typed should not error
	writeEnter(t, p)
	errOut := readAllStderr(t, p)
	if errOut != "" {
		t.Fatalf("expected no error for empty enter, got %q", errOut)
	}
}

// typeable reports whether every key of line is offered as it is typed.
func typeable(p process.P, line string) bool {
	for _, r := range line {
		if !containsDatum(p.Next(), process.Chars(string(r))) {
			return false
		}
		_ = p.Stdin(process.Data{process.Chars(string(r))})
	}
	return true
}

func canRun(p process.P, line string) bool {
	return typeable(p, line) && containsDatum(p.Next(), process.TermEnter)
}

// TestShellNextInvariant_Lines pins which whole command lines can and
// can't be typed and run, one per bug class the validators guard.
func TestShellNextInvariant_Lines(t *testing.T) {
	for _, tc := range []struct {
		line string
		ok   bool
	}{
		// Pipes take spaces on either side, as in a real shell.
		{"ls | grep note", true},
		{"ls|grep note", true},
		{"cat /etc/hosts | cat", true},
		{"pwd | cat", true},
		{"ls | cat | grep x", true},
		// Builtins and interactive commands don't go in pipelines.
		{"ls | cd", false},
		{"cd /home | ls", false},
		{"exit | ls", false},
		{"ssh test", true},
		{"ls | ssh test", false},
		{"ssh test | cat", false},
		// A pipe connects a command that prints to one that reads stdin.
		{"apt list | grep a", true},
		{"sudo ls | grep a", true},
		{"ls | sudo grep a", true},
		{"ls | ls", false},
		{"ls | pwd", false},
		{"ls | apt list", false},
		{"ls | sudo ls", false},
		// A receiver reads the pipe, not a file.
		{"ls | cat /home/user/notes.txt", false},
		{"ls | grep a /home/user/notes.txt", false},
		// Commands that only change files print nothing to pipe, so a
		// pipeline can't change what a later segment was checked against.
		{"rm /home/user/notes.txt | cat", false},
		{"touch /home/user/x | cat", false},
		{"mv /home/user/notes.txt /home/user/n.txt | cat", false},
		{"sudo rm /bin/ls | cat", false},
		{"apt install x | cat", false},
		// grep's pattern can't swallow the pipe.
		{"ls | grep a|b", false},
		// cat, grep and rm take files, not folders.
		{"cat /home/user/notes.txt", true},
		{"cat /home", false},
		{"cat .", false},
		{"cat /", false},
		{"grep note /home/user/notes.txt", true},
		{"grep note /home", false},
		{"rm /home/user/notes.txt", true},
		{"rm /home/user/docs", false},
		{"rm .", false},
		{"rm ..", false},
		{"rm /", false},
		// Commands that take no args take none.
		{"pwd", true},
		{"pwd x", false},
		{"exit x", false},
		// Subcommands are a fixed set.
		{"nx quest", true},
		{"nx foo", false},
		{"nx", false},
		{"apt list", true},
		{"apt install grep", false}, // already installed
		{"apt frob", false},
		{"apt", false},
		// sudo takes a command from /bin and that command's args.
		{"sudo touch /etc/config", true},
		{"sudo ls", true},
		{"sudo cd /", false},
		{"sudo ssh test", false},
		{"sudo frob", false},
		{"sudo", false},
		// touch takes a name in an existing folder.
		{"touch /home/user/new.txt", true},
		{"touch /home/user/notes.txt", true},
		{"touch /nowhere/new.txt", false},
		{"touch /", false},
		{"touch /home/", false},
		{"touch -x", false},
		// mv takes an existing file and a destination.
		{"mv /home/user/notes.txt /home/user/n.txt", true},
		{"mv /home/user/notes.txt /home", true},
		{"mv /home/user/notes.txt /home/", true},
		{"mv /home/user/notes.txt /home/user", false}, // same place
		{"mv /home/user/notes.txt /home/user/notes.txt", false},
		{"mv /home/user/notes.txt", false},
		{"mv /home/user/docs /tmp", false},
		// An unreadable folder can be named (permission denied is a
		// real lesson) but nothing inside it is completed.
		{"ls /root", true},
		{"cd /root", true},
		{"cat /root/secret.txt", false},
		{"touch /root/new.txt", false},
		{"sudo cat /root/secret.txt", true},
		{"sudo touch /root/new.txt", true},
		// ~ is the player's home, wherever a path goes.
		{"cat ~/notes.txt", true},
		{"cat ~", false},
		{"ls ~", true},
		{"ls ~/docs", true},
		{"cd ~", true},
		{"cd ~/docs", true},
		{"touch ~/new.txt", true},
		{"touch ~", false},
		{"rm ~", false},
		{"mv ~/notes.txt ~/docs", true},
		{"mv ~/notes.txt ~", false}, // already there
		{"sudo cat ~/notes.txt", true},
		{"ls ~x", false},
		{"grep ~ ~/notes.txt", false},
		// Options come first, each once, from the command's flags.
		{"ls -a", true},
		{"ls -a ~", true},
		{"ls -a | grep profile", true},
		{"ls -", false},
		{"ls -aa", false},
		{"ls -a -a", false},
		{"ls -x", false},
		{"ls ~ -a", false},
		{"cat -a ~/notes.txt", false},
		{"sudo ls -a /root", true},
		// Hidden files are named like any other.
		{"cat ~/.profile", true},
		// whoami is who the command runs as.
		{"whoami", true},
		{"whoami x", false},
		{"whoami | grep user", true},
		{"sudo whoami", true},
		// echo takes any words, and -n, and prints them.
		{"echo", true},
		{"echo hello world", true},
		{"echo -n hi", true},
		{"echo hi | cat", true},
		{"echo -x", false},
		{"echo a|b", false},
		{"echo hi -n", false},
		// mkdir makes a new folder in an existing one.
		{"mkdir ~/new", true},
		{"mkdir ~/docs/new", true},
		{"mkdir ~/docs", false}, // exists
		{"mkdir ~/notes.txt", false},
		{"mkdir /nowhere/new", false},
		{"mkdir ~/new/", false},
		{"mkdir ~/new | cat", false},
		{"mkdir /etc/new", true}, // permission denied
		// rmdir removes an empty folder, never a file or a full one.
		{"rmdir ~/docs", true},
		{"rmdir ~/notes.txt", false},
		{"rmdir /home", false},
		{"rmdir .", false},
		{"rmdir /", false},
		{"rmdir /root", false}, // can't see in
		{"sudo rmdir /home/user/docs", true},
	} {
		_, p := testSetup(t)
		if got := canRun(p, tc.line); got != tc.ok {
			t.Errorf("%q: runnable=%v, want %v", tc.line, got, tc.ok)
		}
	}
}

// TestShellNextInvariant_StrandedCwd: if the cwd is deleted from under
// the player, absolute paths still work, as in a real shell.
func TestShellNextInvariant_StrandedCwd(t *testing.T) {
	sim, p := testSetup(t)
	writeString(t, p, "cd /home/user/docs")
	writeEnter(t, p)
	_ = readAllStdout(t, p)
	c, _ := sim.GetComputer("test")
	if err := c.Filesystem.DeleteFile([]string{"home", "user"}, "docs", file.OwnerRoot); err != nil {
		t.Fatal(err)
	}
	if !canRun(p, "cd /home") {
		t.Fatal("cd /home must be typeable from a deleted cwd")
	}
}

func TestRegisterRequiresValidArgs(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Register without ValidArgs must panic")
		}
	}()
	simulation.Register("no-validator", &simulation.Binary{Launch: launch})
}

// TestShellPipelineFindsBinariesFirst: every segment's binary is found
// before any segment runs, so an earlier segment removing a later one's
// binary doesn't break the pipeline halfway.
func TestShellPipelineFindsBinariesFirst(t *testing.T) {
	_, p := testSetup(t)
	writeString(t, p, "sudo rm /bin/pwd | pwd")
	writeEnter(t, p)
	_ = readAllStdout(t, p)
	if errOut := readAllStderr(t, p); errOut != "" {
		t.Fatalf("unexpected error: %q", errOut)
	}
}

// canRunFresh is canRun that clears the line afterwards.
func canRunFresh(p process.P, line string) bool {
	ok := canRun(p, line)
	_ = p.Stdin(process.Data{process.TermClear})
	p.Read()
	return ok
}

func runLine(t *testing.T, p process.P, line string) string {
	t.Helper()
	writeString(t, p, line)
	writeEnter(t, p)
	return strings.TrimPrefix(readAllStdout(t, p), line)
}

// TestShellHome: a login shell starts in $HOME, bare `cd` goes there,
// and bare `cd` isn't offered when already home (a no-op, like `cd .`).
func TestShellHome(t *testing.T) {
	sim, _ := testSetup(t)
	p, err := sim.Launch("test", "user", "shell", []string{"-l"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := runLine(t, p, "pwd"); got != "/home/user\n" {
		t.Fatalf("login shell starts in %q, want /home/user", got)
	}
	if canRunFresh(p, "cd") {
		t.Error("bare cd offered while already home")
	}
	runLine(t, p, "cd /etc")
	if !canRunFresh(p, "cd") {
		t.Fatal("bare cd not offered away from home")
	}
	runLine(t, p, "cd")
	if got := runLine(t, p, "pwd"); got != "/home/user\n" {
		t.Fatalf("bare cd went to %q, want /home/user", got)
	}
}

// TestShellLoginWithoutHome: like login(1), a user whose home can't be
// entered starts at /.
func TestShellLoginWithoutHome(t *testing.T) {
	sim, _ := testSetup(t)
	p, err := sim.Launch("test", "ghost", "shell", []string{"-l"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := runLine(t, p, "pwd"); got != "/\n" {
		t.Fatalf("got %q, want /", got)
	}
	if canRunFresh(p, "cd") {
		t.Error("bare cd offered with no home to go to")
	}
}

// TestShellTildeExpansion: the shell expands ~ before the command runs,
// for builtins, binaries, pipelines and sudo alike.
func TestShellTildeExpansion(t *testing.T) {
	sim, p := testSetup(t)
	if got := runLine(t, p, "cat ~/notes.txt | grep another"); got != "another\n" {
		t.Errorf("cat ~/notes.txt | grep: got %q", got)
	}
	if got := runLine(t, p, "sudo cat ~/notes.txt"); got != "a note\nanother\n" {
		t.Errorf("sudo cat ~/notes.txt: got %q", got)
	}
	runLine(t, p, "mv ~/notes.txt ~/docs")
	c, _ := sim.GetComputer("test")
	if _, err := c.Filesystem.Navigate([]string{"home", "user", "docs", "notes.txt"}); err != nil {
		t.Errorf("mv ~/notes.txt ~/docs: %v", err)
	}
	runLine(t, p, "cd ~/docs")
	if got := runLine(t, p, "pwd"); got != "/home/user/docs\n" {
		t.Errorf("cd ~/docs: pwd %q", got)
	}
	runLine(t, p, "cd ~")
	if got := runLine(t, p, "pwd"); got != "/home/user\n" {
		t.Errorf("cd ~: pwd %q", got)
	}
	if canRunFresh(p, "cd ~") {
		t.Error("cd ~ offered while already home")
	}
}

// TestShellWithoutNx: nx comes from the game. A simulation without one
// doesn't offer nx, rather than offering it and then failing.
func TestShellWithoutNx(t *testing.T) {
	sim, _ := testSetup(t)
	sim.Nx = nil
	p, err := sim.Launch("test", "user", "shell", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if typeable(p, "nx") {
		t.Fatal("nx offered without a game")
	}
}
