package command

import (
	"github.com/josephburnett/nixy-go/pkg/binary"
	"github.com/josephburnett/nixy-go/pkg/process"
)

func init() {
	binary.Register("shell", shell)
}

func shell(args binary.Args) (process.Process, error) {
	return &shellType{Args: args}, nil
}

type shellType struct {
	binary.Args
}

func (s *shellType) Read() (string, error)   { return "", nil }
func (s *shellType) Write(string) error      { return nil }
func (s *shellType) Owner() string           { return "" }
func (s *shellType) Parent() process.Process { return nil }
func (s *shellType) Kill() error             { return nil }
