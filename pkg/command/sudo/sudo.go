package sudo

import (
	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("sudo", &simulation.Binary{
		Launch:    launch,
		ValidArgs: validArgs,
	})
}

// validArgs is `sudo COMMAND ARGS...`: a non-interactive command from
// /bin other than sudo itself, then whatever that command's own
// validator accepts.
func validArgs(ctx simulation.ArgContext, partial string) simulation.Suggestion {
	return command.ValidCommand(ctx, partial, sudoable(ctx), func(name string) (simulation.ValidArgs, bool) {
		b, err := ctx.Sim.LookupBinary(ctx.Hostname, name)
		if err != nil {
			return nil, false
		}
		return b.ValidArgs, true
	})
}

func sudoable(ctx simulation.ArgContext) []string {
	var names []string
	for _, name := range ctx.Sim.Commands(ctx.Hostname) {
		if name == "sudo" {
			continue
		}
		if b, err := ctx.Sim.LookupBinary(ctx.Hostname, name); err == nil && !b.Interactive {
			names = append(names, name)
		}
	}
	return names
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
	b, err := sim.LookupBinary(hostname, cmdName)
	if err != nil {
		return command.NewErrorProcess(file.OwnerRoot, "sudo: "+err.Error()+"\n"), nil
	}
	if b.Interactive {
		return command.NewErrorProcess(file.OwnerRoot, "sudo: "+cmdName+": interactive commands are not supported\n"), nil
	}
	return b.Launch(sim, file.OwnerRoot, hostname, cwd, args[1:])
}
