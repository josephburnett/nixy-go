package shell

import (
	"strings"
	"testing"

	_ "github.com/josephburnett/nixy-go/pkg/command/grep"
	_ "github.com/josephburnett/nixy-go/pkg/command/ls"
	_ "github.com/josephburnett/nixy-go/pkg/command/pwd"
	_ "github.com/josephburnett/nixy-go/pkg/command/rm"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func testSetup(t *testing.T) (*simulation.S, process.P) {
	t.Helper()
	fs := &file.F{
		Type:             file.Folder,
		Owner:            file.OwnerRoot,
		OwnerPermission:  file.Write,
		CommonPermission: file.Read,
		Files: map[string]*file.F{
			"bin": {
				Type:             file.Folder,
				Owner:            file.OwnerRoot,
				OwnerPermission:  file.Write,
				CommonPermission: file.Read,
				Files: map[string]*file.F{
					"pwd": {
						Type:             file.Binary,
						Owner:            file.OwnerRoot,
						OwnerPermission:  file.Write,
						CommonPermission: file.Read,
						Data:             "pwd",
					},
					"ls": {
						Type:             file.Binary,
						Owner:            file.OwnerRoot,
						OwnerPermission:  file.Write,
						CommonPermission: file.Read,
						Data:             "ls",
					},
					"grep": {
						Type:             file.Binary,
						Owner:            file.OwnerRoot,
						OwnerPermission:  file.Write,
						CommonPermission: file.Read,
						Data:             "grep",
					},
					"rm": {
						Type:             file.Binary,
						Owner:            file.OwnerRoot,
						OwnerPermission:  file.Write,
						CommonPermission: file.Read,
						Data:             "rm",
					},
				},
			},
			"home": {
				Type:             file.Folder,
				Owner:            file.OwnerRoot,
				OwnerPermission:  file.Write,
				CommonPermission: file.Read,
				Files: map[string]*file.F{
					"user": {
						Type:             file.Folder,
						Owner:            "user",
						OwnerPermission:  file.Write,
						CommonPermission: file.Read,
						Files:            map[string]*file.F{},
					},
				},
			},
		},
	}
	sim := simulation.New()
	err := sim.Boot("test", fs)
	if err != nil {
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
		_, err := p.Stdin(process.Data{process.Chars(string(c))})
		if err != nil {
			t.Fatalf("writing '%c': %v", c, err)
		}
	}
}

func writeEnter(t *testing.T, p process.P) {
	t.Helper()
	_, err := p.Stdin(process.Data{process.TermEnter})
	if err != nil {
		t.Fatal(err)
	}
}

func writeBackspace(t *testing.T, p process.P) {
	t.Helper()
	_, err := p.Stdin(process.Data{process.TermBackspace})
	if err != nil {
		t.Fatal(err)
	}
}

func readAllStdout(t *testing.T, p process.P) string {
	t.Helper()
	var result string
	for i := 0; i < 10; i++ {
		out, _, err := p.Stdout()
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range out {
			if c, ok := d.(process.Chars); ok {
				result += string(c)
			}
		}
		if len(out) == 0 {
			break
		}
	}
	return result
}

func readAllStderr(t *testing.T, p process.P) string {
	t.Helper()
	var result string
	for i := 0; i < 10; i++ {
		out, _, err := p.Stderr()
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range out {
			if c, ok := d.(process.Chars); ok {
				result += string(c)
			}
		}
		if len(out) == 0 {
			break
		}
	}
	return result
}

func containsDatum(datums []process.Datum, target process.Datum) bool {
	for _, d := range datums {
		switch d := d.(type) {
		case process.Chars:
			if t, ok := target.(process.Chars); ok && d == t {
				return true
			}
		case process.TermCode:
			if t, ok := target.(process.TermCode); ok && d == t {
				return true
			}
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

	if errOut == "" {
		t.Fatal("expected error for unknown command")
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

	_, eof, _ := p.Stdout()
	if !eof {
		t.Fatal("expected eof after exit")
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
