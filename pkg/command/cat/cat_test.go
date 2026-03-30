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

func TestCatFile(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, []string{"hello.txt"})
	out := stdout(t, p)
	if !strings.Contains(out, "hello world") {
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
	if !strings.Contains(errOut, "permission denied") {
		t.Fatalf("expected permission denied, got %q", errOut)
	}
}

func TestCatStdin(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "test", []string{}, nil)
	p.Stdin(process.CharsData("piped input"))
	out := stdout(t, p)
	if !strings.Contains(out, "piped input") {
		t.Fatalf("expected 'piped input', got %q", out)
	}
}
