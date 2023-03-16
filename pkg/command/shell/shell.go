package shell

import (
	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/environment"
	"github.com/josephburnett/nixy-go/pkg/process"
)

func init() {
	environment.Register("shell", environment.Binary{
		Launch: launch,
		Test:   test,
	})
}

func launch(context environment.Context, args string, input process.Process) (process.Process, error) {
	return &shell{
		Context:          context,
		args:             args,
		currentDirectory: context.Directory, // clone me
		currentCommand:   "",
	}, nil
}

func test(context environment.Context, args []string) []error {
	return make([]error, len(args))
}

var _ process.Process = &shell{}

type shell struct {
	eof bool
	environment.Context
	args             string
	currentDirectory []string
	currentCommand   string
	echo             process.Data
}

func (s *shell) Read() (process.Data, bool, error) {
	if s.eof {
		return nil, true, nil
	}
	data := s.echo
	s.echo = nil
	return data, false, nil
}

func (s *shell) Write(in process.Data) (bool, error) {
	if s.eof {
		return true, command.ErrEndOfFile
	}
	s.echo = append(s.echo, in...)
	return false, nil
}

func (s *shell) Test(in []process.Data) []error {
	return nil
}

func (s *shell) Owner() string {
	return s.ParentProcess.Owner()
}

func (s *shell) Parent() process.Process {
	return s.ParentProcess
}

func (s *shell) Kill() error {
	s.eof = true
	return nil
}
