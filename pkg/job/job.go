package job

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/simulation"
)

type J interface {
	Name() string
	Foreground(ctx simulation.Context)
	Background(ctx simulation.Context)
	Done(ctx simulation.Context) bool
}

var registry map[string]J

func Register(name string, j J) error {
	if _, registered := registry[name]; registered {
		return fmt.Errorf("job %v already registered", name)
	}
	registry[name] = j
	return nil
}
