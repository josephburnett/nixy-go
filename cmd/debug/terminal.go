package debug

import (
	"github.com/josephburnett/nixy-go/pkg/environment"
)

type Terminal struct {
	env *environment.Environment
}

func NewTerminal() (*Terminal, error) {
	e, err := environment.NewEnvironment(nil)
	if err != nil {
		return nil, err
	}
	return &Terminal{
		env: e,
	}, nil
}
