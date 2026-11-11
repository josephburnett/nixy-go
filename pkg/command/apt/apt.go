package apt

import (
	"errors"
	"fmt"
	"sort"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("apt", &simulation.Binary{
		Launch: launch,
		ValidArgs: command.Subcommands(map[string]simulation.ValidArgs{
			"install": command.Positional{Args: []command.Arg{installable}, Required: 1}.Validate,
			"list":    command.Positional{Writes: true}.Validate,
		}),
	})
}

// installable accepts a known package not yet in /bin. When every
// package is installed, `apt install` isn't offered at all.
var installable = command.OneOf(func(ctx simulation.ArgContext) []string {
	c, err := ctx.Sim.GetComputer(ctx.Hostname)
	if err != nil {
		return nil
	}
	bin, err := c.Filesystem.LookupDir([]string{"bin"}, ctx.User)
	if err != nil {
		return nil
	}
	var out []string
	for _, pkg := range packageNames() {
		if _, installed := bin.Files[pkg]; !installed {
			out = append(out, pkg)
		}
	}
	return out
})

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
	"yes":   "yes",
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

// lockError is what real apt says when a normal user tries to install.
const lockError = "E: Could not open lock file /var/lib/dpkg/lock-frontend - open (13: Permission denied)\n" +
	"E: Unable to acquire the dpkg frontend lock (/var/lib/dpkg/lock-frontend), are you root?\n"

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
			return command.NewErrorProcess(owner, fmt.Sprintf("E: Unable to locate package %v\n", pkg)), nil
		}
		bin, err := c.Filesystem.LookupDir([]string{"bin"}, owner)
		if err == nil {
			if _, installed := bin.Files[pkg]; installed {
				continue
			}
		}
		// Installing writes /bin as whoever runs apt, so only root
		// can: a normal user hits real apt's lock error.
		err = c.Filesystem.CreateFile([]string{"bin"}, pkg, &file.F{
			Type:             file.Binary,
			Owner:            owner,
			OwnerPermission:  file.Write,
			CommonPermission: file.Read,
			Data:             binaryName,
		}, owner)
		if errors.Is(err, file.ErrPermission) {
			return command.NewErrorProcess(owner, lockError), nil
		}
		if err != nil {
			return command.NewErrorProcess(owner, fmt.Sprintf("E: %v\n", err)), nil
		}
	}

	return command.NewSingleValueProcess(owner, ""), nil
}

func aptList(owner string) (process.P, error) {
	var output string
	for _, pkg := range packageNames() {
		output += pkg + "\n"
	}
	return command.NewSingleValueProcess(owner, output), nil
}

// packageNames returns the known package names, sorted.
func packageNames() []string {
	names := make([]string, 0, len(PackageRegistry))
	for pkg := range PackageRegistry {
		names = append(names, pkg)
	}
	sort.Strings(names)
	return names
}
