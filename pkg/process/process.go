package process

import "fmt"

type Process interface {
	Read() (Data, bool, error)
	Write(Data) (bool, error)
	Test([]Data) []error
	Owner() string
	Parent() Process
	Kill() error
}

type ProcessSpace struct {
	next      int
	processes map[int]Process
}

func NewProcessSpace() *ProcessSpace {
	return &ProcessSpace{
		processes: map[int]Process{},
	}
}

func (ps *ProcessSpace) Add(p Process) {
	ps.processes[ps.next] = p
	ps.next++
}

func (ps *ProcessSpace) List() map[int]Process {
	return ps.processes
}

func (ps *ProcessSpace) Kill(id int) error {
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
