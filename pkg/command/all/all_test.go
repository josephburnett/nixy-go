package all

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/command/apt"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Every package apt can install must run once installed.
func TestEveryPackageIsRegistered(t *testing.T) {
	for pkg, bin := range apt.PackageRegistry {
		if _, err := simulation.GetBinary(bin); err != nil {
			t.Errorf("apt package %q: %v", pkg, err)
		}
	}
}
