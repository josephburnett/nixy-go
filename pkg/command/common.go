package command

import "github.com/josephburnett/nixy-go/pkg/process"

var _ process.P = &singleValueProcess{}

type singleValueProcess struct {
	parent process.P
	value  string
	eof    bool
}

func NewSingleValueProcess(parent process.P, value string) process.P {
	return &singleValueProcess{
		parent: parent,
		value:  value,
	}
}

func (s *singleValueProcess) Read() (process.Data, bool, error) {
	if s.eof {
		return nil, s.eof, ErrEndOfFile
	}
	s.eof = true
	return process.CharsData(s.value), s.eof, nil
}

func (s *singleValueProcess) Write(_ process.Data) (bool, error) {
	return false, ErrReadOnlyProcess
}

func (s *singleValueProcess) Test(_ []process.Data) []error {
	return nil
}

func (s *singleValueProcess) Owner() string {
	return s.parent.Owner()
}

func (s *singleValueProcess) Parent() process.P {
	return s.parent
}

func (s *singleValueProcess) Kill() error {
	s.eof = true
	return nil
}
