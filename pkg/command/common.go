package command

import "github.com/josephburnett/nixy-go/pkg/process"

type singleValueProcess struct {
	parent process.Process
	value  string
	eof    bool
}

func NewSingleValueProcess(parent process.Process, value string) process.Process {
	return &singleValueProcess{
		parent: parent,
		value:  value,
	}
}

func (s *singleValueProcess) Read() (string, bool, error) {
	if s.eof {
		return "", s.eof, ErrEndOfFile
	}
	s.eof = true
	return s.value, s.eof, nil
}

func (s *singleValueProcess) Write(_ string) error {
	return ErrReadOnlyProcess
}

func (s *singleValueProcess) Owner() string {
	return s.parent.Owner()
}

func (s *singleValueProcess) Parent() process.Process {
	return s.parent
}

func (s *singleValueProcess) Kill() error {
	s.eof = true
	return nil
}
