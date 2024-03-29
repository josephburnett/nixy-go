package shell

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("shell", &simulation.Binary{
		Launch: launch,
		Test:   test,
	})
}

func launch(
	sim *simulation.S,
	owner string,
	hostname string,
	cwd []string,
	args string,
	input process.P,
) (process.P, error) {

	return &shell{
		simulation:       sim,
		owner:            owner,
		hostname:         hostname,
		args:             args,
		currentDirectory: cwd,
		currentCommand:   "",
	}, nil
}

func test(
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
	simulation    *simulation.S
	owner         string
	hostname      string
	args          string
	parentProcess process.P

	childProcess     process.P
	currentDirectory []string
	currentCommand   string
	echo             process.Data
}

func (s *shell) Read() (process.Data, bool, error) {
	data := s.echo
	s.echo = nil
	if s.childProcess != nil {
		// Connected to child process
		d, eof, err := s.childProcess.Read()
		if eof {
			// Terminate child childProcess
			s.childProcess.Kill()
			s.childProcess = nil
		}
		data = append(data, d...)
		return data, false, err
	}
	return data, false, nil
}

func (s *shell) Write(d process.Data) (bool, error) {
	if s.childProcess != nil {
		// Connected to child child process
		eof, err := s.childProcess.Write(d)
		if eof {
			// Terminate child childProcess
			s.childProcess.Kill()
			s.childProcess = nil
		}
		return false, err
	}

	if len(d) != 1 {
		return false, fmt.Errorf("shell only supports 1 datum at a time: %v", len(d))
	}
	in := d[0]
	s.echo = append(s.echo, in)

	switch in := in.(type) {
	case process.Chars:
		s.currentCommand += string(in)
		return false, nil
	case process.TermCode:
		switch in {
		case process.TermEnter:
			b, err := s.getBinary()
			s.currentCommand = ""
			if err != nil {
				return false, err
			}
			p, err := b.Launch(
				s.simulation,
				s.owner,
				s.hostname,
				s.currentDirectory,
				"",
				s,
			)
			if err != nil {
				return false, err
			}
			s.childProcess = p
		case process.TermBackspace:
			if len(s.currentCommand) > 0 {
				s.currentCommand = s.currentCommand[:len(s.currentCommand)-1]
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unhandled input type %T", in)
	}
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
	return nil
}

func (s *shell) getBinary() (*simulation.Binary, error) {
	ss := strings.Fields(s.currentCommand)
	if len(ss) == 0 {
		return nil, fmt.Errorf("command not found")
	}
	c, err := s.simulation.GetComputer(s.hostname)
	if err != nil {
		return nil, err
	}
	filename := ss[0]
	f, err := c.GetFile([]string{"bin", filename})
	if err != nil {
		return nil, err
	}
	if f.Type != file.Binary {
		return nil, fmt.Errorf("file %v is not executable", filename)
	}
	return simulation.GetBinary(f.Data)
}
