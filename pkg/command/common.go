package command

import "github.com/josephburnett/nixy-go/pkg/process"

var _ process.P = &singleValueProcess{}

type singleValueProcess struct {
	owner string
	value string
	eof   bool
}

func NewSingleValueProcess(owner, value string) process.P {
	return &singleValueProcess{
		owner: owner,
		value: value,
	}
}

func (s *singleValueProcess) Owner() string {
	return s.owner
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

func (s *singleValueProcess) Kill() error {
	s.eof = true
	return nil
}
