package command

import (
	"github.com/josephburnett/nixy-go/pkg/binary"
	"github.com/josephburnett/nixy-go/pkg/environment"
	"github.com/josephburnett/nixy-go/pkg/process"
)

func init() {
	binary.Register("pwd", pwd)
}

func pwd(
	env *environment.Environment,
	hostname string,
	parent process.Process,
	args string,
	dryrun bool,
) (process.Process, error) {
	return nil, nil
}
