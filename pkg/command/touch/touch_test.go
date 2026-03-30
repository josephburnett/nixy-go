package touch

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
					"existing.txt": {Type: file.Text, Owner: "user",
						OwnerPermission: file.Write, CommonPermission: file.Read,
						Data: "data"},
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

func TestTouchNewFile(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"home/new.txt"})
	errOut := stderr(t, p)
	if errOut != "" {
		t.Fatalf("unexpected error: %q", errOut)
	}
	c, _ := sim.GetComputer("test")
	f, err := c.Filesystem.Navigate([]string{"home", "new.txt"})
	if err != nil {
		t.Fatal("file should exist")
	}
	if f.Type != file.Text {
		t.Fatalf("expected Text, got %v", f.Type)
	}
}

func TestTouchExisting(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"home/existing.txt"})
	errOut := stderr(t, p)
	if errOut != "" {
		t.Fatalf("unexpected error: %q", errOut)
	}
	// Data should be unchanged
	c, _ := sim.GetComputer("test")
	f, _ := c.Filesystem.Navigate([]string{"home", "existing.txt"})
	if f.Data != "data" {
		t.Fatalf("touch should not modify existing file data")
	}
}

func TestTouchPermissionDenied(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"etc/new.txt"})
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "permission denied") {
		t.Fatalf("expected permission denied, got %q", errOut)
	}
}

func TestTouchNoArgs(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, nil)
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "missing operand") {
		t.Fatalf("expected missing operand, got %q", errOut)
	}
}
