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
		Launch:      launch,
		ValidArgs:   command.Positional{Args: []command.Arg{command.Hostname}, Required: 1}.Validate,
		Interactive: true, // opens a remote shell
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
	// Resolving the name reads /etc/hosts as the user, like the
	// resolver in real ssh.
	hosts, _ := c.Filesystem.ReadFile([]string{"etc", "hosts"}, owner)
	found := false
	for _, line := range strings.Split(hosts, "\n") {
		if strings.TrimSpace(line) == targetHost {
			found = true
			break
		}
	}
	if !found {
		return command.NewErrorProcess(owner, fmt.Sprintf("ssh: Could not resolve hostname %v: Name or service not known\n", targetHost)), nil
	}

	// Check target computer exists in simulation
	_, err = sim.GetComputer(targetHost)
	if err != nil {
		return command.NewErrorProcess(owner, fmt.Sprintf("ssh: connect to host %v port 22: Connection refused\n", targetHost)), nil
	}

	// Launch a shell on the target machine
	remoteShell, err := sim.Launch(targetHost, owner, "shell", []string{"-l"}, nil)
	if err != nil {
		return command.NewErrorProcess(owner, fmt.Sprintf("ssh: %v\n", err)), nil
	}

	return remoteShell, nil
}
