package shell

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/file"
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
