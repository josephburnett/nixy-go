package whoami

import (
	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("whoami", &simulation.Binary{
		Launch:    launch,
		ValidArgs: command.Positional{Writes: true}.Validate,
	})
}

// launch prints who the command runs as: the shell's owner, the name
// in the prompt, or root under sudo.
func launch(_ *simulation.S, owner, _ string, _, _ []string) (process.P, error) {
	return command.NewSingleValueProcess(owner, owner+"\n"), nil
}
