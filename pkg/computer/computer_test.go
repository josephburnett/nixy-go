package computer

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
)

type mockProcess struct{ process.Written }

func (p *mockProcess) Stdin(process.Data) error { return nil }
func (p *mockProcess) Next() []process.Datum    { return nil }
func (p *mockProcess) Owner() string            { return "user" }

func TestNewComputer(t *testing.T) {
	fs := &file.F{Type: file.Folder, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Files: map[string]*file.F{}}
	c := New(fs)
	if c.Filesystem != fs {
		t.Fatal("filesystem should be set")
	}
}

func TestComputerKillNeedsOwnerOrRoot(t *testing.T) {
	c := New(&file.F{Type: file.Folder, Owner: file.OwnerRoot})
	pid := c.Add(&mockProcess{}, "cat") // owned by "user"
	if err := c.Kill(pid, "eve"); err != process.ErrNotPermitted {
		t.Fatalf("another user killed it: %v", err)
	}
	if err := c.Kill(pid, "user"); err != nil {
		t.Fatalf("owner can't kill: %v", err)
	}
	pid = c.Add(&mockProcess{}, "cat")
	if err := c.Kill(pid, file.OwnerRoot); err != nil {
		t.Fatalf("root can't kill: %v", err)
	}
	if err := c.Kill(pid, file.OwnerRoot); err != process.ErrNoProcess {
		t.Fatalf("killed twice: %v", err)
	}
	if len(c.Processes()) != 0 {
		t.Fatalf("processes left: %v", c.Processes())
	}
}
