package pwd

import (
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("pwd", &simulation.Binary{
		Launch:    launch,
		ValidArgs: command.Positional{Writes: true}.Validate,
	})
}

func launch(
	_ *simulation.S,
	owner string,
	_ string,
	cwd []string,
	args []string,
) (process.P, error) {
	warning := ""
	if len(args) > 0 {
		warning = "pwd: ignoring non-option arguments\n" // as GNU pwd
	}
	return command.Exited(owner, "/"+strings.Join(cwd, "/")+"\n", warning, 0), nil
}
