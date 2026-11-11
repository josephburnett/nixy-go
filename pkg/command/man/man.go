package man

import (
	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("man", &simulation.Binary{
		Launch: launch,
		// `man COMMAND`: a command in /bin that has a page.
		ValidArgs: command.Positional{Args: []command.Arg{documented}, Required: 1, Writes: true}.Validate,
	})
}

// documented accepts the name of a command in /bin with a manual page.
// Pages come with their command, so a command that isn't installed has
// none.
var documented = command.OneOf(func(ctx simulation.ArgContext) []string {
	var names []string
	for _, name := range ctx.Sim.Commands(ctx.Hostname, ctx.User) {
		if _, ok := page(ctx.Sim, ctx.Hostname, ctx.User, name); ok {
			names = append(names, name)
		}
	}
	return names
})

// page finds the page for the command name in /bin, by the binary it
// runs (so a renamed command keeps its page).
func page(sim *simulation.S, hostname, user, name string) (string, bool) {
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return "", false
	}
	f, err := c.Filesystem.Lookup([]string{"bin", name}, user)
	if err != nil || f.Type != file.Binary {
		return "", false
	}
	p, ok := Pages[f.Data]
	return p, ok
}

func launch(sim *simulation.S, owner, hostname string, _, args []string) (process.P, error) {
	if len(args) == 0 {
		return command.NewErrorProcess(owner, "What manual page do you want?\n"), nil
	}
	p, ok := page(sim, hostname, owner, args[0])
	if !ok {
		return command.NewErrorProcess(owner, "No manual entry for "+args[0]+"\n"), nil
	}
	return command.NewSingleValueProcess(owner, p), nil
}
