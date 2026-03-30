package sudo

import (
	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("sudo", &simulation.Binary{
		Launch: launch,
	})
}

func launch(
	sim *simulation.S,
	_ string, // original owner ignored; we run as root
	hostname string,
	cwd []string,
	args []string,
) (process.P, error) {
	if len(args) == 0 {
		return command.NewErrorProcess(file.OwnerRoot, "sudo: missing command\n"), nil
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	// Look up binary in /bin
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}
	f, err := c.Filesystem.Navigate([]string{"bin", cmdName})
	if err != nil {
		return command.NewErrorProcess(file.OwnerRoot, "sudo: command not found: "+cmdName+"\n"), nil
	}
	if f.Type != file.Binary {
		return command.NewErrorProcess(file.OwnerRoot, "sudo: not executable: "+cmdName+"\n"), nil
	}
	b, err := simulation.GetBinary(f.Data)
	if err != nil {
		return nil, err
	}

	// Launch as root
	return b.Launch(sim, file.OwnerRoot, hostname, cwd, cmdArgs)
}
