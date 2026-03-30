package apt

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("apt", &simulation.Binary{
		Launch: launch,
	})
}

// PackageRegistry maps package names to the binary name they provide.
// Commands must be registered in the simulation binary registry separately.
var PackageRegistry = map[string]string{
	"ls":    "ls",
	"cat":   "cat",
	"grep":  "grep",
	"rm":    "rm",
	"touch": "touch",
	"mv":    "mv",
	"pwd":   "pwd",
	"ssh":   "ssh",
	"sudo":  "sudo",
	"apt":   "apt",
}

func launch(
	sim *simulation.S,
	owner string,
	hostname string,
	_ []string,
	args []string,
) (process.P, error) {
	if len(args) == 0 {
		return command.NewErrorProcess(owner, "apt: missing subcommand\n"), nil
	}

	switch args[0] {
	case "install":
		return aptInstall(sim, owner, hostname, args[1:])
	case "list":
		return aptList(owner)
	default:
		return command.NewErrorProcess(owner, fmt.Sprintf("apt: unknown subcommand '%v'\n", args[0])), nil
	}
}

func aptInstall(sim *simulation.S, owner, hostname string, args []string) (process.P, error) {
	if len(args) == 0 {
		return command.NewErrorProcess(owner, "apt install: missing package name\n"), nil
	}

	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}

	for _, pkg := range args {
		binaryName, ok := PackageRegistry[pkg]
		if !ok {
			return command.NewErrorProcess(owner, fmt.Sprintf("apt: package '%v' not found\n", pkg)), nil
		}

		// Check if already installed
		binDir, err := c.Filesystem.Navigate([]string{"bin"})
		if err != nil {
			return command.NewErrorProcess(owner, "apt: /bin not found\n"), nil
		}
		if _, exists := binDir.Files[pkg]; exists {
			continue // Already installed
		}

		// Install: create binary in /bin
		newBin := &file.F{
			Type:             file.Binary,
			Owner:            file.OwnerRoot,
			OwnerPermission:  file.Write,
			CommonPermission: file.Read,
			Data:             binaryName,
		}
		// apt needs root to install, but we install as root regardless
		// (apt itself handles privilege escalation in real life)
		err = c.Filesystem.CreateFile([]string{"bin"}, pkg, newBin, file.OwnerRoot)
		if err != nil {
			return command.NewErrorProcess(owner, fmt.Sprintf("apt: %v\n", err)), nil
		}
	}

	return command.NewSingleValueProcess(owner, ""), nil
}

func aptList(owner string) (process.P, error) {
	var output string
	for pkg := range PackageRegistry {
		output += pkg + "\n"
	}
	return command.NewSingleValueProcess(owner, output), nil
}
