package shell

import (
	"github.com/josephburnett/nixy-go/pkg/binary"
	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
)

func init() {
	binary.Register("shell", binary.Binary{
		Launch:   launch,
		Validate: validate,
	})
}

func launch(context binary.Context, args string, input process.Process) (process.Process, error) {
	return &shell{
		Context:          context,
		args:             args,
		currentDirectory: context.Directory, // clone me
		currentCommand:   "",
	}, nil
}

func validate(context binary.Context, args []string) []error {
	return make([]error, len(args))
}

type shell struct {
	eof bool
	binary.Context
	args             string
	currentDirectory []string
	currentCommand   string
	echo             string
}

func (s *shell) Read() (string, bool, error) {
	if s.eof {
		return "", true, nil
	}
	e := s.echo
	s.echo = ""
	return e, false, nil
}

func (s *shell) Write(in string) error {
	if s.eof {
		return command.ErrEndOfFile
	}
	s.echo += in
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
