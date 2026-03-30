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

func (s *singleValueProcess) Stdout() (process.Data, bool, error) {
	if s.eof {
		return nil, true, ErrEndOfFile
	}
	s.eof = true
	return process.CharsData(s.value), true, nil
}

func (s *singleValueProcess) Stderr() (process.Data, bool, error) {
	return nil, true, nil
}

func (s *singleValueProcess) Stdin(_ process.Data) (bool, error) {
	return false, ErrReadOnlyProcess
}

func (s *singleValueProcess) Next() []process.Datum {
	return nil
}

func (s *singleValueProcess) Kill() error {
	s.eof = true
	return nil
}

// errorProcess writes a message to stderr and exits.
var _ process.P = &errorProcess{}

type errorProcess struct {
	owner string
	msg   string
	eof   bool
}

func NewErrorProcess(owner, msg string) process.P {
	return &errorProcess{owner: owner, msg: msg}
}

func (e *errorProcess) Owner() string { return e.owner }

func (e *errorProcess) Stdout() (process.Data, bool, error) {
	return nil, true, nil
}

func (e *errorProcess) Stderr() (process.Data, bool, error) {
	if e.eof {
		return nil, true, nil
	}
	e.eof = true
	return process.CharsData(e.msg), true, nil
}

func (e *errorProcess) Stdin(_ process.Data) (bool, error) {
	return false, ErrReadOnlyProcess
}

func (e *errorProcess) Next() []process.Datum { return nil }

func (e *errorProcess) Kill() error {
	e.eof = true
	return nil
}
