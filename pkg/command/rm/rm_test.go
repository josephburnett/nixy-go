package rm

import (
	"strings"
	"testing"

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
			"home": {Type: file.Folder, Owner: "user",
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"junk.txt": {Type: file.Text, Owner: "user",
						OwnerPermission: file.Write, CommonPermission: file.Read},
				}},
			"etc": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.None,
				Files: map[string]*file.F{
					"config": {Type: file.Text, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.None},
				}},
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

func TestRmFile(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"home/junk.txt"})
	errOut := stderr(t, p)
	if errOut != "" {
		t.Fatalf("unexpected error: %q", errOut)
	}
	c, _ := sim.GetComputer("test")
	_, err := c.Filesystem.Navigate([]string{"home", "junk.txt"})
	if err == nil {
		t.Fatal("file should be deleted")
	}
}

func TestRmNonexistent(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"nope.txt"})
	errOut := stderr(t, p)
	if errOut == "" {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestRmPermissionDenied(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"etc/config"})
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "permission denied") {
		t.Fatalf("expected permission denied, got %q", errOut)
	}
}

func TestRmNoArgs(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, nil)
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "missing operand") {
		t.Fatalf("expected missing operand, got %q", errOut)
	}
}
