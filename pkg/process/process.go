package process

type Process interface {
	Read() (string, error)
	Write(string) error
	Kill() error
}

type ProcessSpace struct {
	next      int
	processes map[int]*Process
}

func NewProcessSpace() *ProcessSpace {
	return &ProcessSpace{
		processes: map[int]*Process{},
	}
}

func (ps *ProcessSpace) Add(p *Process) {
	ps.processes[ps.next] = p
	ps.next++
}
