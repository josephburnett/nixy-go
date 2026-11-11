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

func TestComputerBoot(t *testing.T) {
	fs := &file.F{Type: file.Folder, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read}
	c := New(fs)
	err := c.Boot()
	if err != nil {
		t.Fatal(err)
	}
}

func TestComputerAddProcess(t *testing.T) {
	fs := &file.F{Type: file.Folder, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read}
	c := New(fs)
	// Should not panic
	c.Add(&mockProcess{})
}
