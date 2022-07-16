package command

import (
	"github.com/josephburnett/nixy-go/pkg/binary"
	"github.com/josephburnett/nixy-go/pkg/environment"
	"github.com/josephburnett/nixy-go/pkg/process"
)

func init() {
	binary.Register("pwd", func(
		env *environment.Environment,
		hostname string,
		args string,
		dryrun bool,
	) (*process.Process, error) {
		return nil, nil
	})
}
