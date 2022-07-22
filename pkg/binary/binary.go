package binary

import (
	"github.com/josephburnett/nixy-go/pkg/environment"
	"github.com/josephburnett/nixy-go/pkg/process"
)

type Binary struct {
	Launch   Launch
	Validate Validate
}

type Launch func(context Context, args string, input process.Process) (process.Process, error)

type Validate func(context Context, args []string) []error

type Context struct {
	Env           *environment.Environment
	ParentProcess process.Process
	Hostname      string
	Directory     []string
}

var registry = map[string]Binary{}

func Register(name string, b Binary) {
	registry[name] = b
}
