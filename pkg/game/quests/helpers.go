package quests

import (
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func fileExists(sim *simulation.S, hostname string, path []string) bool {
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return false
	}
	_, err = c.Filesystem.Navigate(path)
	return err == nil
}

func binaryInstalled(sim *simulation.S, hostname string, name string) bool {
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return false
	}
	f, err := c.Filesystem.Navigate([]string{"bin", name})
	if err != nil {
		return false
	}
	return f.Type == file.Binary
}

// planNavigate returns a command to get to the target machine.
// Returns "" if already on the right machine.
func planNavigate(currentHost, targetHost string, _ []string) string {
	if currentHost == targetHost {
		return ""
	}
	if currentHost != "laptop" {
		return "exit"
	}
	return "ssh " + targetHost
}

func pathEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
