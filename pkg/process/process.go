package process

import "fmt"

type P interface {
	Read() (Data, bool, error)
	Write(Data) (bool, error)
	Test([]Data) []error
	Owner() string
	Parent() P
	Kill() error
}

type Space struct {
	next      int
	processes map[int]P
}

func NewSpace() *Space {
	return &Space{
		processes: map[int]P{},
	}
}

func (ps *Space) Add(p P) {
	ps.processes[ps.next] = p
	ps.next++
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
