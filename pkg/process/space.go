package process

import "fmt"

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
	p.Kill()
	delete(ps.processes, id)
	return nil
}
