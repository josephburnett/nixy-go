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
			"home": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"user": {Type: file.Folder, Owner: "user",
						OwnerPermission: file.Write, CommonPermission: file.Read,
						Files: map[string]*file.F{}},
				}},
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
	return process.Text(p.Read(), process.Stderr)
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
	s := process.Text(p.Read(), process.Stdout)
	// ssh opens a login shell: it starts in the user's home.
	if !strings.Contains(s, "/home/user\n") {
		t.Fatalf("expected pwd to print /home/user, got %q", s)
	}
}

func TestSshUnknownHost(t *testing.T) {
	sim := testSetup(t)
	p, _ := launch(sim, "user", "local", []string{}, []string{"unknown"})
	errOut := stderr(t, p)
	if errOut != "ssh: Could not resolve hostname unknown: Name or service not known\n" {
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
	p.Read()
	if _, exited := p.Exit(); !exited {
		t.Fatal("expected the remote shell to exit")
	}
}
