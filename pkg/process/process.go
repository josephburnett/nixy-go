package process

import "fmt"

type P interface {
	// Stdout reads output from the process.
	Stdout() (out Data, eof bool, err error)
	// Stderr reads error output from the process.
	Stderr() (out Data, eof bool, err error)
	// Stdin writes input to the process.
	Stdin(in Data) (eof bool, err error)
	// Next returns the set of valid next inputs for this process.
	Next() []Datum
	// Owner returns the user who owns this process.
	Owner() string
	// Kill terminates the process.
	Kill() error
}

type Space struct {
	i         int
	processes map[int]P
}

func NewSpace() *Space {
	return &Space{
		processes: map[int]P{},
	}
}

func (ps *Space) Add(p P) int {
	id := ps.i
	ps.processes[id] = p
	ps.i++
	return id
}

func (ps *Space) List() map[int]P {
	return ps.processes
}

func (ps *Space) Kill(id int) error {
	p, ok := ps.processes[id]
	if !ok {
		return fmt.Errorf("invalid process id %v", id)
	}
	err := p.Kill()
	if err != nil {
		return err
	}
	delete(ps.processes, id)
	return nil
}
