package shell

import (
	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("shell", simulation.Binary{
		Launch: launch,
		Test:   test,
	})
}

var launch simulation.Launch = func(
	sim *simulation.S,
	owner string,
	hostname string,
	cwd []string,
	args string,
	input process.P,
) (process.P, error) {

	return &shell{
		simulation:       sim,
		args:             args,
		currentDirectory: cwd,
		currentCommand:   "",
	}, nil
}

var test simulation.Test = func(
	sim *simulation.S,
	owner string,
	hostname string,
	cwd []string,
	args []string,
) []error {
	return make([]error, len(args))
}

var _ process.P = &shell{}

type shell struct {
	eof              bool
	simulation       *simulation.S
	args             string
	currentDirectory []string
	currentCommand   string
	echo             process.Data
	parentProcess    process.P
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
	return make([]error, len(in))
}

func (s *shell) Owner() string {
	return s.parentProcess.Owner()
}

func (s *shell) Parent() process.P {
	return s.parentProcess
}

func (s *shell) Kill() error {
	s.eof = true
	return nil
}
