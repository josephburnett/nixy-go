package cat

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
			"hello.txt": {Type: file.Text, Owner: "user",
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Data: "hello world"},
			"secret.txt": {Type: file.Text, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.None,
				Data: "top secret"},
			"sub": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
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

func TestCatFile(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"hello.txt"})
	out := stdout(t, p)
	if out != "hello world" { // as stored: cat adds no newline
		t.Fatalf("expected 'hello world', got %q", out)
	}
}

func TestCatNonexistent(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"nope.txt"})
	errOut := stderr(t, p)
	if errOut == "" {
		t.Fatal("expected error")
	}
}

func TestCatDirectory(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"sub"})
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "directory") {
		t.Fatalf("expected directory error, got %q", errOut)
	}
}

func TestCatPermissionDenied(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"secret.txt"})
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "Permission denied") {
		t.Fatalf("expected permission denied, got %q", errOut)
	}
}

func TestCatStdin(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, nil)
	p.Stdin(process.CharsData("piped input"))
	out := stdout(t, p)
	if _, exited := p.Exit(); exited {
		t.Fatal("cat should run until its input closes")
	}
	p.CloseStdin()
	if status, exited := p.Exit(); !exited || status != 0 {
		t.Fatalf("cat should exit 0 when its input closes, got %v %v", status, exited)
	}
	if !strings.Contains(out, "piped input") {
		t.Fatalf("expected 'piped input', got %q", out)
	}
}
