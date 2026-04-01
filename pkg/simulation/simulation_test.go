package simulation

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
)

func init() {
	Register("test-cmd", &Binary{
		Launch: func(_ *S, owner, hostname string, cwd, args []string) (process.P, error) {
			return &testProcess{owner: owner}, nil
		},
	})
}

type testProcess struct {
	owner string
}

func (p *testProcess) Stdout() (process.Data, bool, error) { return nil, true, nil }
func (p *testProcess) Stderr() (process.Data, bool, error) { return nil, true, nil }
func (p *testProcess) Stdin(_ process.Data) (bool, error)   { return false, nil }
func (p *testProcess) Next() []process.Datum                { return nil }
func (p *testProcess) Owner() string                        { return p.owner }
func (p *testProcess) Kill() error                          { return nil }

func testFS() *file.F {
	return &file.F{
		Type: file.Folder, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Files: map[string]*file.F{
			"bin": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{}},
		},
	}
}

func TestBootAndGetComputer(t *testing.T) {
	s := New()
	err := s.Boot("host1", testFS())
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.GetComputer("host1")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("expected computer")
	}
}

func TestBootDuplicate(t *testing.T) {
	s := New()
	s.Boot("host1", testFS())
	err := s.Boot("host1", testFS())
	if err == nil {
		t.Fatal("expected error for duplicate hostname")
	}
}

func TestGetComputerNotFound(t *testing.T) {
	s := New()
	_, err := s.GetComputer("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown hostname")
	}
}

func TestLaunch(t *testing.T) {
	s := New()
	s.Boot("host1", testFS())
	p, err := s.Launch("host1", "user", "test-cmd", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Owner() != "user" {
		t.Fatalf("expected owner 'user', got %q", p.Owner())
	}
}

func TestLaunchUnknownHost(t *testing.T) {
	s := New()
	_, err := s.Launch("nope", "user", "test-cmd", nil, nil)
	if err == nil {
		t.Fatal("expected error for unknown host")
	}
}

func TestLaunchUnknownBinary(t *testing.T) {
	s := New()
	s.Boot("host1", testFS())
	_, err := s.Launch("host1", "user", "nonexistent", nil, nil)
	if err == nil {
		t.Fatal("expected error for unknown binary")
	}
}

func TestRegisterAndGetBinary(t *testing.T) {
	// "test-cmd" already registered in init()
	b, err := GetBinary("test-cmd")
	if err != nil {
		t.Fatal(err)
	}
	if b == nil {
		t.Fatal("expected binary")
	}
}

func TestGetBinaryNotFound(t *testing.T) {
	_, err := GetBinary("does-not-exist")
	if err == nil {
		t.Fatal("expected error for unregistered binary")
	}
}
