package pwd

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/binary"
	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
)

func init() {
	binary.Register("pwd", binary.Binary{
		Launch:   launch,
		Validate: validate,
	})
}

func launch(context binary.Context, _ string, _ process.Process) (process.Process, error) {
	return command.NewSingleValueProcess(
		context.ParentProcess,
		strings.Join(context.Directory, "/"),
	), nil
}

func validate(_ binary.Context, args []string) []error {
	errs := make([]error, len(args))
	for i := range args {
		errs[i] = errAcceptNoArgs
	}
	return errs
}

var errAcceptNoArgs = fmt.Errorf("pwd does not accept parameters")
