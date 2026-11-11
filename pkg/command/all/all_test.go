package all

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/command/apt"
	"github.com/josephburnett/nixy-go/pkg/command/man"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Every command has a manual page.
func TestEveryCommandHasAPage(t *testing.T) {
	for _, name := range simulation.Registered() {
		if _, ok := man.Pages[name]; !ok {
			t.Errorf("%s has no manual page (pkg/command/man)", name)
		}
	}
	for name := range man.Pages {
		if _, err := simulation.GetBinary(name); err != nil {
			t.Errorf("manual page for %s, which isn't a command", name)
		}
	}
}

// Every package apt can install must run once installed.
func TestEveryPackageIsRegistered(t *testing.T) {
	for pkg, bin := range apt.PackageRegistry {
		if _, err := simulation.GetBinary(bin); err != nil {
			t.Errorf("apt package %q: %v", pkg, err)
		}
	}
}
