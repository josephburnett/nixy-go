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

func launch(context binary.Context, _ string) (process.Process, error) {
	return command.NewSingleValueProcess(
		context.Parent,
		strings.Join(context.Directory, "/"),
	), nil
}

func validate(_ binary.Context, argsList []string) []error {
	errs := make([]error, len(argsList))
	for i := range argsList {
		errs[i] = errAcceptNoArgs
	}
	return errs
}

var errAcceptNoArgs = fmt.Errorf("args does not accept parameters")
