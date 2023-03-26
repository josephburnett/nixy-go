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
		Test:   test,
	})
}

func launch(
	_ *simulation.S,
	owner string,
	_ string,
	cwd []string,
	args string,
	_ process.P,
) (process.P, error) {
	if len(args) != 0 {
		return nil, errAcceptNoArgs
	}
	return command.NewSingleValueProcess(
		owner,
		strings.Join(cwd, "/"),
	), nil
}

func test(
	_ *simulation.S,
	_ string,
	_ string,
	_ []string,
	args []string,
) []error {
	errs := make([]error, len(args))
	for i := range args {
		errs[i] = errAcceptNoArgs
	}
	return errs
}

var errAcceptNoArgs = fmt.Errorf("pwd does not accept parameters")
