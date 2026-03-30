package ssh

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
	localFS := &file.F{
		Type: file.Folder, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Files: map[string]*file.F{
			"bin": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"ssh": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "ssh"},
				}},
			"etc": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"hosts": {Type: file.Text, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read,
						Data: "local\nremote"},
				}},
		},
	}
	remoteFS := &file.F{
		Type: file.Folder, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Files: map[string]*file.F{
			"bin": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"pwd": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "pwd"},
				}},
		},
	}
	sim := simulation.New()
	if err := sim.Boot("local", localFS); err != nil {
		t.Fatal(err)
	}
	if err := sim.Boot("remote", remoteFS); err != nil {
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

func TestSshConnect(t *testing.T) {
	sim := testSetup(t)
	p, err := launch(sim, "user", "local", []string{}, []string{"remote"})
	if err != nil {
		t.Fatal(err)
	}
	// We should get a remote shell. Write pwd + enter, read output.
	for _, c := range "pwd" {
		p.Stdin(process.Data{process.Chars(string(c))})
	}
	p.Stdin(process.Data{process.TermEnter})
	out, _, _ := p.Stdout()
	var s string
	for _, d := range out {
		if c, ok := d.(process.Chars); ok {
			s += string(c)
		}
	}
	if !strings.Contains(s, "/") {
		t.Fatalf("expected pwd output, got %q", s)
	}
}

func TestSshUnknownHost(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "local", []string{}, []string{"unknown"})
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "could not resolve") {
		t.Fatalf("expected resolve error, got %q", errOut)
	}
}

func TestSshNoArgs(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "local", []string{}, nil)
	errOut := stderr(t, p)
	if !strings.Contains(errOut, "missing hostname") {
		t.Fatalf("expected missing hostname, got %q", errOut)
	}
}

func TestSshExit(t *testing.T) {
	sim := testSetup(t)
	p, err := launch(sim, "user", "local", []string{}, []string{"remote"})
	if err != nil {
		t.Fatal(err)
	}
	// Type "exit" to close remote shell
	for _, c := range "exit" {
		p.Stdin(process.Data{process.Chars(string(c))})
	}
	p.Stdin(process.Data{process.TermEnter})
	_, eof, _ := p.Stdout()
	if !eof {
		t.Fatal("expected eof after exit")
	}
}
