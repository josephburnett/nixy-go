package pwd

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("pwd", &simulation.Binary{
		Launch: launch,
	})
}

func launch(
	_ *simulation.S,
	owner string,
	_ string,
	cwd []string,
	args []string,
) (process.P, error) {
	if len(args) != 0 {
		return nil, errAcceptNoArgs
	}
	return command.NewSingleValueProcess(
		owner,
		"/"+strings.Join(cwd, "/")+"\n",
	), nil
}

var errAcceptNoArgs = fmt.Errorf("pwd does not accept parameters")
