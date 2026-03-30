package ssh

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("ssh", &simulation.Binary{
		Launch: launch,
	})
}

func launch(
	sim *simulation.S,
	owner string,
	hostname string,
	cwd []string,
	args []string,
) (process.P, error) {
	if len(args) == 0 {
		return command.NewErrorProcess(owner, "ssh: missing hostname\n"), nil
	}

	targetHost := args[0]

	// Check /etc/hosts on current machine
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}
	hostsFile, err := c.Filesystem.Navigate([]string{"etc", "hosts"})
	if err != nil {
		return command.NewErrorProcess(owner, "ssh: /etc/hosts not found\n"), nil
	}

	// Check if target is in hosts
	found := false
	for _, line := range strings.Split(hostsFile.Data, "\n") {
		if strings.TrimSpace(line) == targetHost {
			found = true
			break
		}
	}
	if !found {
		return command.NewErrorProcess(owner, fmt.Sprintf("ssh: could not resolve hostname '%v'\n", targetHost)), nil
	}

	// Check target computer exists in simulation
	_, err = sim.GetComputer(targetHost)
	if err != nil {
		return command.NewErrorProcess(owner, fmt.Sprintf("ssh: connection refused: %v\n", targetHost)), nil
	}

	// Launch a shell on the target machine
	remoteShell, err := sim.Launch(targetHost, owner, "shell", nil, []string{})
	if err != nil {
		return command.NewErrorProcess(owner, fmt.Sprintf("ssh: %v\n", err)), nil
	}

	return remoteShell, nil
}
