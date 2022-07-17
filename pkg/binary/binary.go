package binary

import (
	"github.com/josephburnett/nixy-go/pkg/environment"
	"github.com/josephburnett/nixy-go/pkg/process"
)

type Func func(args Args) (process.Process, error)

type Args struct {
	Env       *environment.Environment
	Parent    process.Process
	Hostname  string
	Directory []string
	Args      string
	DryRun    bool
}

var registry = map[string]Func{}

func Register(name string, f Func) {
	registry[name] = f
}
