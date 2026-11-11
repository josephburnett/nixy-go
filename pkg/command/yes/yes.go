package yes

import (
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("yes", &simulation.Binary{
		Launch: launch,
		// `yes [WORD...]` prints its words (or y) forever, until
		// interrupted (Ctrl+C) or its reader goes away (SIGPIPE).
		ValidArgs: command.Positional{
			Args:     []command.Arg{command.Word},
			Variadic: true,
			Writes:   true,
		}.Validate,
	})
}

func launch(
	_ *simulation.S,
	owner string,
	_ string,
	_ []string,
	args []string,
) (process.P, error) {
	line := "y"
	if len(args) > 0 {
		line = strings.Join(args, " ")
	}
	return &yes{Proc: command.NewProc(owner), line: process.CharsData(line + "\n")}, nil
}

// yes never exits on its own. Processes act only when called, so it
// writes one line per Read and can't run away with the game.
type yes struct {
	command.Proc
	line process.Data
}

func (y *yes) Read() []process.Output {
	y.Write(process.Stdout, y.line)
	return y.Written.Read()
}
