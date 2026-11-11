package shell

import (
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
)

// What the basic commands print and do, run through the shell as the
// player types them.
func TestShellBasicCommands(t *testing.T) {
	sim, p := loginSetup(t)
	c, _ := sim.GetComputer("test")
	for _, tc := range []struct{ line, want string }{
		{"whoami", "user\n"},
		{"sudo whoami", "root\n"},
		{"echo hello world", "hello world\n"},
		{"echo", "\n"},
		{"echo -n hi", "hi"},
		{"mkdir new", ""},
		{"rmdir docs", ""},
	} {
		if got := runLine(t, p, tc.line); got != tc.want {
			t.Errorf("%q printed %q, want %q", tc.line, got, tc.want)
		}
		if errOut := readAllStderr(t, p); errOut != "" {
			t.Errorf("%q: stderr %q", tc.line, errOut)
		}
	}
	made, err := c.Filesystem.Navigate([]string{"home", "user", "new"})
	if err != nil || made.Type != file.Folder || made.Owner != "user" {
		t.Fatalf("mkdir new: %+v %v", made, err)
	}
	if _, err := c.Filesystem.Navigate([]string{"home", "user", "docs"}); err == nil {
		t.Fatal("rmdir docs left it")
	}
}

// head prints the first 10 lines, and stops a source that goes on; wc
// counts lines, words and bytes like GNU wc.
func TestShellHeadWc(t *testing.T) {
	_, p := loginSetup(t)
	for _, tc := range []struct{ line, want string }{
		{"yes | head", strings.Repeat("y\n", 10)},
		{"head notes.txt", "a note\nanother\n"},
		{"wc notes.txt", " 1  3 14 notes.txt\n"},
		{"wc -l notes.txt", "1 notes.txt\n"},
		{"ls | wc", "      2       2      15\n"},
		{"ls | wc -l", "2\n"},
		{"yes | head | wc -l", "10\n"},
	} {
		if !canRun(p, tc.line) {
			t.Fatalf("%q can't be typed", tc.line)
		}
		writeEnter(t, p)
		out, _, done := runUntilPrompt(p, 30)
		if !done {
			t.Fatalf("%q is still running", tc.line)
		}
		if got := strings.TrimPrefix(out, tc.line); got != tc.want {
			t.Errorf("%q printed %q, want %q", tc.line, got, tc.want)
		}
	}
}

// ps shows the table as `ps -o pid,user,cmd` would, itself included;
// kill signals by PID, only your own processes unless root.
func TestShellPsKill(t *testing.T) {
	_, p := testSetup(t) // PID 1 is a root shell, 2 this one
	for _, tc := range []struct{ line, stdout, stderr string }{
		{"ps", "  PID USER     CMD\n    2 user     shell\n    3 user     ps\n", ""},
		{"ps -e | grep shell", "    1 root     shell\n    2 user     shell\n    5 user     grep shell\n", ""},
		{"kill 1", "", "kill: (1) - Operation not permitted\n"},
		{"kill 2", "", ""}, // a shell ignores SIGTERM
		{"sudo kill -9 1", "", ""},
		{"ps -e", "  PID USER     CMD\n    2 user     shell\n    9 user     ps -e\n", ""},
	} {
		if !canRun(p, tc.line) {
			t.Fatalf("%q can't be typed", tc.line)
		}
		writeEnter(t, p)
		if got := strings.TrimPrefix(readAllStdout(t, p), tc.line); got != tc.stdout {
			t.Errorf("%q: stdout %q, want %q", tc.line, got, tc.stdout)
		}
		if got := readAllStderr(t, p); got != tc.stderr {
			t.Errorf("%q: stderr %q, want %q", tc.line, got, tc.stderr)
		}
	}
	if !canRun(p, "kill -9 2") {
		t.Fatal("can't kill -9 your own shell")
	}
	writeEnter(t, p)
	if status, exited := p.Exit(); !exited || status != 137 {
		t.Fatalf("kill -9 of the shell: exited=%v status %d", exited, status)
	}
}

// > writes a command's stdout to a file the shell opens as the player
// before the command runs; >> appends.
func TestShellRedirect(t *testing.T) {
	sim, p := loginSetup(t)
	c, _ := sim.GetComputer("test")
	data := func(name string) string {
		t.Helper()
		f, err := c.Filesystem.Navigate([]string{"home", "user", name})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return f.Data
	}
	for _, tc := range []struct{ line, stderr string }{
		{"echo hi > note.txt", ""},
		{"echo there >> note.txt", ""},
		{"ls > files", ""},
		{"echo hi > /etc/motd", "/etc/motd: Permission denied\n"},
		{"sudo ls /root > root.txt", ""},
		{"yes | grep y > ys", ""},
	} {
		if !canRun(p, tc.line) {
			t.Fatalf("%q can't be typed", tc.line)
		}
		writeEnter(t, p)
		if strings.HasPrefix(tc.line, "yes") {
			_ = p.Read()
			if err := p.Stdin(process.Data{process.SigInt}); err != nil {
				t.Fatal(err)
			}
		}
		if got := strings.TrimPrefix(readAllStdout(t, p), tc.line); strings.Trim(got, "^C\n") != "" {
			t.Errorf("%q printed %q to the terminal", tc.line, got)
		}
		if got := readAllStderr(t, p); got != tc.stderr {
			t.Errorf("%q: stderr %q, want %q", tc.line, got, tc.stderr)
		}
	}
	if got := data("note.txt"); got != "hi\nthere\n" {
		t.Errorf("note.txt = %q", got)
	}
	// The shell made files before ls ran, so ls lists it.
	if got := data("files"); got != "docs\nfiles\nnote.txt\nnotes.txt\n" {
		t.Errorf("files = %q", got)
	}
	// sudo runs ls as root, but the shell opens the file as the player.
	f, _ := c.Filesystem.Navigate([]string{"home", "user", "root.txt"})
	if f.Owner != "user" || f.Data != "secret.txt\n" {
		t.Errorf("root.txt: owner %s data %q", f.Owner, f.Data)
	}
	if got := data("ys"); got == "" || strings.Trim(got, "y\n") != "" {
		t.Errorf("ys = %q", got)
	}
	if _, err := c.Filesystem.Navigate([]string{"etc", "motd"}); err == nil {
		t.Error("/etc/motd was created")
	}
}

// Making or removing a folder takes write permission on its parent.
func TestShellMkdirRmdirPermission(t *testing.T) {
	_, p := loginSetup(t)
	for _, tc := range []struct{ line, want string }{
		{"mkdir /etc/new", "mkdir: cannot create directory '/etc/new': Permission denied\n"},
		{"sudo mkdir /etc/new", ""},
		{"rmdir /etc/new", "rmdir: failed to remove '/etc/new': Permission denied\n"},
		{"sudo rmdir /etc/new", ""},
	} {
		if !canRun(p, tc.line) {
			t.Fatalf("%q can't be typed", tc.line)
		}
		writeEnter(t, p)
		_ = readAllStdout(t, p)
		if got := readAllStderr(t, p); got != tc.want {
			t.Errorf("%q: stderr %q, want %q", tc.line, got, tc.want)
		}
	}
}
