package shell

import (
	"github.com/josephburnett/nixy-go/pkg/binary"
	"github.com/josephburnett/nixy-go/pkg/process"
)

func init() {
	binary.Register("shell", binary.Binary{
		Launch:   launch,
		Validate: validate,
	})
}

func launch(context binary.Context, args string, input process.Process) (process.Process, error) {
	return &shellType{
		Context:          context,
		args:             args,
		currentDirectory: context.Directory, // clone me
	}, nil
}

func validate(context binary.Context, args []string) []error {
	return make([]error, len(args))
}

type shellType struct {
	binary.Context
	args             string
	currentDirectory []string
}

func (s *shellType) Read() (string, bool, error) { return "", false, nil }
func (s *shellType) Write(string) error          { return nil }
func (s *shellType) Owner() string               { return "" }
func (s *shellType) Parent() process.Process     { return nil }
func (s *shellType) Kill() error                 { return nil }
