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
					".hidden": {Type: file.Text, Owner: "user",
						OwnerPermission: file.Write, CommonPermission: file.Read},
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
	return process.Text(p.Read(), process.Stdout)
}

func stderr(t *testing.T, p process.P) string {
	t.Helper()
	return process.Text(p.Read(), process.Stderr)
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

// Dotfiles are hidden unless -a, which also lists . and ..
func TestLsHidden(t *testing.T) {
	sim := testSetup(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"home"}, "a.txt\nb.txt\n"},
		{[]string{"-a", "home"}, ".\n..\n.hidden\na.txt\nb.txt\n"},
	} {
		p, err := launch(sim, "user", "test", []string{}, tc.args)
		if err != nil {
			t.Fatal(err)
		}
		if got := stdout(t, p); got != tc.want {
			t.Errorf("ls %v = %q, want %q", tc.args, got, tc.want)
		}
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
	if !strings.Contains(errOut, "Permission denied") {
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
