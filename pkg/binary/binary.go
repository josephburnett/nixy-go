package binary

import (
	"github.com/josephburnett/nixy-go/pkg/environment"
	"github.com/josephburnett/nixy-go/pkg/process"
)

type Func func(
	env *environment.Environment,
	hostname string,
	parent process.Process,
	args string,
	dryrun bool,
) (process.Process, error)

var registry = map[string]Func{}

func Register(name string, f Func) {
	registry[name] = f
}
