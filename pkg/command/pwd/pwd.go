package pwd

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/environment"
	"github.com/josephburnett/nixy-go/pkg/process"
)

func init() {
	environment.Register("pwd", environment.Binary{
		Launch:   launch,
		Validate: validate,
	})
}

func launch(context environment.Context, _ string, _ process.Process) (process.Process, error) {
	return command.NewSingleValueProcess(
		context.ParentProcess,
		strings.Join(context.Directory, "/"),
	), nil
}

func validate(_ environment.Context, args []string) []error {
	errs := make([]error, len(args))
	for i := range args {
		errs[i] = errAcceptNoArgs
	}
	return errs
}

var errAcceptNoArgs = fmt.Errorf("pwd does not accept parameters")
