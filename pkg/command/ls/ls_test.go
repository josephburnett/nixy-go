package ls

import (
	"strings"
	"testing"

	_ "github.com/josephburnett/nixy-go/pkg/command/shell"
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
					"ls": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "ls"},
				}},
			"home": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"a.txt": {Type: file.Text, Owner: "user",
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "hello"},
					"b.txt": {Type: file.Text, Owner: "user",
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "world"},
				}},
			"secret": {Type: file.Folder, Owner: file.OwnerRoot,
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

func stdout(t *testing.T, p process.P) string {
	t.Helper()
	out, _, err := p.Stdout()
	if err != nil {
		t.Fatal(err)
	}
	var s string
	for _, d := range out {
		if c, ok := d.(process.Chars); ok {
			s += string(c)
		}
	}
	return s
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

func TestLsCwd(t *testing.T) {
	sim := testSetup(t)
	p, err := launch(sim, "user", "test", []string{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := stdout(t, p)
	if !strings.Contains(out, "bin") || !strings.Contains(out, "home") {
		t.Fatalf("expected bin and home, got %q", out)
	}
}

func TestLsPath(t *testing.T) {
	sim := testSetup(t)
	p, err := launch(sim, "user", "test", []string{}, []string{"home"})
	if err != nil {
		t.Fatal(err)
	}
	out := stdout(t, p)
	if !strings.Contains(out, "a.txt") || !strings.Contains(out, "b.txt") {
		t.Fatalf("expected a.txt and b.txt, got %q", out)
	}
}

func TestLsNonexistent(t *testing.T) {
	sim := testSetup(t)
	p, err := launch(sim, "user", "test", []string{}, []string{"nope"})
	if err != nil {
		t.Fatal(err)
	}
	errOut := stderr(t, p)
	if errOut == "" {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestLsPermissionDenied(t *testing.T) {
	sim := testSetup(t)
	p, err := launch(sim, "user", "test", []string{}, []string{"secret"})
	if err != nil {
		t.Fatal(err)
	}
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "permission denied") {
		t.Fatalf("expected permission denied, got %q", errOut)
	}
}

func TestLsEmpty(t *testing.T) {
	sim := testSetup(t)
	p, err := launch(sim, file.OwnerRoot, "test", []string{}, []string{"secret"})
	if err != nil {
		t.Fatal(err)
	}
	out := stdout(t, p)
	if out != "" {
		t.Fatalf("expected empty output for empty dir, got %q", out)
	}
}
