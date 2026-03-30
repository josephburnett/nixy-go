package mv

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
					"old.txt": {Type: file.Text, Owner: "user",
						OwnerPermission: file.Write, CommonPermission: file.Read,
						Data: "content"},
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

func TestMvRename(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"home/old.txt", "home/new.txt"})
	errOut := stderr(t, p)
	if errOut != "" {
		t.Fatalf("unexpected error: %q", errOut)
	}
	c, _ := sim.GetComputer("test")
	_, err := c.Filesystem.Navigate([]string{"home", "old.txt"})
	if err == nil {
		t.Fatal("old file should be gone")
	}
	f, err := c.Filesystem.Navigate([]string{"home", "new.txt"})
	if err != nil {
		t.Fatal("new file should exist")
	}
	if f.Data != "content" {
		t.Fatalf("expected 'content', got %q", f.Data)
	}
}

func TestMvNonexistent(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"home/nope.txt", "home/new.txt"})
	errOut := stderr(t, p)
	if errOut == "" {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestMvNoArgs(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, nil)
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "missing operand") {
		t.Fatalf("expected missing operand, got %q", errOut)
	}
}
