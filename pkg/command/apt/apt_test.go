package apt

import (
	"strings"
	"testing"

	_ "github.com/josephburnett/nixy-go/pkg/command/pwd"
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
					"apt": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "apt"},
				}},
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
	out, _, _ := p.Stdout()
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

func TestAptInstall(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, file.OwnerRoot, "test", nil, []string{"install", "grep"})
	errOut := stderr(t, p)
	if errOut != "" {
		t.Fatalf("unexpected error: %q", errOut)
	}
	c, _ := sim.GetComputer("test")
	f, err := c.Filesystem.Navigate([]string{"bin", "grep"})
	if err != nil {
		t.Fatal("grep should be installed in /bin")
	}
	if f.Data != "grep" {
		t.Fatalf("expected binary data 'grep', got %q", f.Data)
	}
	if f.Owner != file.OwnerRoot {
		t.Fatalf("installed by root, but owned by %q", f.Owner)
	}
}

// TestAptInstallNeedsRoot: installing writes /bin, which only root can.
func TestAptInstallNeedsRoot(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", nil, []string{"install", "grep"})
	if errOut := stderr(t, p); !strings.Contains(errOut, "Permission denied") || !strings.Contains(errOut, "are you root?") {
		t.Fatalf("expected apt's lock error, got %q", errOut)
	}
	c, _ := sim.GetComputer("test")
	if _, err := c.Filesystem.Navigate([]string{"bin", "grep"}); err == nil {
		t.Fatal("grep installed without root")
	}
}

func TestAptInstallAlreadyInstalled(t *testing.T) {
	sim := testSetup(t)
	// apt is already in /bin
	p, _ := launch(sim, "user", "test", nil, []string{"install", "apt"})
	errOut := stderr(t, p)
	if errOut != "" {
		t.Fatalf("unexpected error for already installed: %q", errOut)
	}
}

func TestAptInstallUnknown(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", nil, []string{"install", "foobar"})
	errOut := stderr(t, p)
	if errOut != "E: Unable to locate package foobar\n" {
		t.Fatalf("expected not found, got %q", errOut)
	}
}

func TestAptList(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", nil, []string{"list"})
	out := stdout(t, p)
	want := "apt\ncat\ngrep\nls\nmv\npwd\nrm\nssh\nsudo\ntouch\n"
	if out != want {
		t.Fatalf("expected sorted package list %q, got %q", want, out)
	}
}

func TestAptNoSubcommand(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", nil, nil)
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "missing subcommand") {
		t.Fatalf("expected missing subcommand, got %q", errOut)
	}
}
