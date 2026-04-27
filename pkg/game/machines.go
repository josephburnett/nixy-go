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
	entries  []MachineEntry
	booted   map[string]bool
	username string // player's chosen name; used to provision /home/<username> on each machine
}

func NewMachineRegistry(entries []MachineEntry, username string) *MachineRegistry {
	return &MachineRegistry{
		entries:  entries,
		booted:   map[string]bool{},
		username: username,
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
	return r.provisionUserHome(sim, e.Hostname)
}

// provisionUserHome ensures /home/<username> exists on the just-booted
// machine, owned by the player. Skips silently if /home doesn't exist
// on this machine (some test fixtures), or if /home/<username> already
// exists (e.g. the player picked "nixy" and the world ships /home/nixy).
func (r *MachineRegistry) provisionUserHome(sim *simulation.S, hostname string) error {
	if r.username == "" {
		return nil
	}
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return err
	}
	if _, err := c.Filesystem.Navigate([]string{"home"}); err != nil {
		return nil // no /home on this machine
	}
	if _, err := c.Filesystem.Navigate([]string{"home", r.username}); err == nil {
		return nil // already exists
	}
	home := &file.F{
		Type:             file.Folder,
		Owner:            r.username,
		OwnerPermission:  file.Write,
		CommonPermission: file.Read,
		Files:            map[string]*file.F{},
	}
	return c.Filesystem.CreateFile([]string{"home"}, r.username, home, file.OwnerRoot)
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
		already := false
		for _, line := range strings.Split(hostsFile.Data, "\n") {
			if strings.TrimSpace(line) == newHost {
				already = true
				break
			}
		}
		if already {
			continue
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
