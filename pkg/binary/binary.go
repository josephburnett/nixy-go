package binary

import (
	"github.com/josephburnett/nixy-go/pkg/environment"
	"github.com/josephburnett/nixy-go/pkg/process"
)

type FuncName struct {
	Name     string
	Namspace string
}

type Func func(env *environment.Environment, hostname string, args string) (*process.Process, error)

var registry = map[FuncName]Func{}

func Register(name FuncName, f Func) {
	registry[name] = f
}
