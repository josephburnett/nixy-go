package game

import (
	"strings"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// MachineEntry defines a machine that can be unlocked.
type MachineEntry struct {
	Hostname   string
	Filesystem func() *file.F
	UnlockedBy Achievement // empty = available from start
}

// MachineRegistry manages which machines are available.
type MachineRegistry struct {
	entries []MachineEntry
	booted  map[string]bool
}

func NewMachineRegistry(entries []MachineEntry) *MachineRegistry {
	return &MachineRegistry{
		entries: entries,
		booted:  map[string]bool{},
	}
}

// BootInitialMachines boots machines that have no unlock requirement.
func (r *MachineRegistry) BootInitialMachines(sim *simulation.S) error {
	for _, e := range r.entries {
		if e.UnlockedBy == "" {
			if err := r.bootMachine(sim, e); err != nil {
				return err
			}
		}
	}
	return nil
}

// CheckUnlocks boots any machines whose unlock achievement has been granted.
func (r *MachineRegistry) CheckUnlocks(sim *simulation.S, achievements *AchievementSet) error {
	for _, e := range r.entries {
		if e.UnlockedBy == "" {
			continue
		}
		if r.booted[e.Hostname] {
			continue
		}
		if achievements.Has(e.UnlockedBy) {
			if err := r.bootMachine(sim, e); err != nil {
				return err
			}
			// Add to /etc/hosts on all booted machines
			r.addToAllHosts(sim, e.Hostname)
		}
	}
	return nil
}

func (r *MachineRegistry) bootMachine(sim *simulation.S, e MachineEntry) error {
	if r.booted[e.Hostname] {
		return nil
	}
	err := sim.Boot(e.Hostname, e.Filesystem())
	if err != nil {
		return err
	}
	r.booted[e.Hostname] = true
	return nil
}

func (r *MachineRegistry) addToAllHosts(sim *simulation.S, newHost string) {
	for hostname := range r.booted {
		c, err := sim.GetComputer(hostname)
		if err != nil {
			continue
		}
		hostsFile, err := c.Filesystem.Navigate([]string{"etc", "hosts"})
		if err != nil {
			continue
		}
		// Check if already listed
		lines := strings.Split(hostsFile.Data, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == newHost {
				return
			}
		}
		hostsFile.Data += "\n" + newHost
	}
}

// AllEntries returns all machine entries for display in nx log.
func (r *MachineRegistry) AllEntries() []MachineEntry {
	return r.entries
}

// IsBooted returns whether a machine has been booted.
func (r *MachineRegistry) IsBooted(hostname string) bool {
	return r.booted[hostname]
}
