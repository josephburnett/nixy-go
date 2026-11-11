package kill

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("kill", &simulation.Binary{
		Launch: launch,
		// `kill [-9] PID` sends SIGTERM, or SIGKILL with -9, to a
		// process that exists. Signalling someone else's process is
		// allowed to type: "Operation not permitted" is the lesson.
		ValidArgs: command.Positional{Args: []command.Arg{pid}, Required: 1, Flags: "9"}.Validate,
	})
}

// pid accepts the PID of a process in this computer's table.
var pid = command.OneOf(func(ctx simulation.ArgContext) []string {
	c, err := ctx.Sim.GetComputer(ctx.Hostname)
	if err != nil {
		return nil
	}
	var pids []string
	for _, p := range c.Processes() {
		pids = append(pids, strconv.Itoa(p.PID))
	}
	return pids
})

func launch(sim *simulation.S, owner, hostname string, _, args []string) (process.P, error) {
	flags, args := command.Options(args)
	if len(args) == 0 {
		return command.NewErrorProcess(owner, "kill: usage: kill [-9] pid\n"), nil
	}
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}
	sig := process.SigTerm
	if strings.ContainsRune(flags, '9') {
		sig = process.SigKill
	}
	var errs strings.Builder
	for _, a := range args {
		err := error(process.ErrNoProcess)
		if n, perr := strconv.Atoi(a); perr == nil {
			err = c.Signal(n, owner, sig)
		}
		if err != nil {
			fmt.Fprintf(&errs, "kill: (%s) - %v\n", a, err)
		}
	}
	if errs.Len() > 0 {
		return command.NewErrorProcess(owner, errs.String()), nil
	}
	return command.NewSingleValueProcess(owner, ""), nil
}
