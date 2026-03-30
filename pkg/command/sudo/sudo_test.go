package sudo

import (
	"strings"
	"testing"

	_ "github.com/josephburnett/nixy-go/pkg/command/shell"
	_ "github.com/josephburnett/nixy-go/pkg/command/touch"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func testSetup(t *testing.T) *simulation.S {
	t.Helper()
	fs := &file.F{
		Type: file.Folder, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Files: map[string]*file.F{
			"bin": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"sudo": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "sudo"},
					"touch": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "touch"},
				}},
			"etc": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.None,
				Files: map[string]*file.F{}},
		},
	}
	sim := simulation.New()
	if err := sim.Boot("test", fs); err != nil {
		t.Fatal(err)
	}
	return sim
}

func stderr(t *testing.T, p process.P) string {
	t.Helper()
	out, _, _ := p.Stderr()
	var s string
	for _, d := range out {
		if c, ok := d.(process.Chars); ok {
			s += string(c)
		}
	}
	return s
}

func TestSudoTouch(t *testing.T) {
	sim := testSetup(t)
	// Regular touch in /etc should fail (no write for non-root)
	// But sudo touch should succeed
	p, _ := launch(sim, "user", "test", []string{}, []string{"touch", "/etc/config"})
	errOut := stderr(t, p)
	if errOut != "" {
		t.Fatalf("unexpected error: %q", errOut)
	}
	c, _ := sim.GetComputer("test")
	_, err := c.Filesystem.Navigate([]string{"etc", "config"})
	if err != nil {
		t.Fatal("file should exist after sudo touch")
	}
}

func TestSudoNoArgs(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, nil)
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "missing command") {
		t.Fatalf("expected missing command, got %q", errOut)
	}
}

func TestSudoUnknownCommand(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"foobar"})
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "command not found") {
		t.Fatalf("expected command not found, got %q", errOut)
	}
}
