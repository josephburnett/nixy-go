package echo

import (
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("echo", &simulation.Binary{
		Launch: launch,
		// `echo [-n] [WORD...]` prints its words; -n leaves off the
		// newline.
		ValidArgs: command.Positional{
			Args:     []command.Arg{command.Word},
			Variadic: true,
			Flags:    "n",
			Writes:   true,
		}.Validate,
	})
}

func launch(_ *simulation.S, owner, _ string, _, args []string) (process.P, error) {
	flags, words := command.Options(args)
	out := strings.Join(words, " ")
	if !strings.ContainsRune(flags, 'n') {
		out += "\n"
	}
	return command.NewSingleValueProcess(owner, out), nil
}
