package command

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/process"
)

var (
	ErrEndOfFile       = fmt.Errorf("end of file")
	ErrReadOnlyProcess = fmt.Errorf("read only process")
)

// Fail is a process that reports err the way coreutils does,
// "cmd: subject: reason". The subject is the operand as typed
// ("cat: foo: No such file or directory") or a phrase naming it
// ("rm: cannot remove 'foo': Permission denied").
func Fail(owner, cmd, subject string, err error) process.P {
	return NewErrorProcess(owner, fmt.Sprintf("%s: %s: %v\n", cmd, subject, err))
}
